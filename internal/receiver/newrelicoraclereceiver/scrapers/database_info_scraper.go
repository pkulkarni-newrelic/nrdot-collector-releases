// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

// DatabaseInfoScraper collects Oracle database version and hosting environment info
type DatabaseInfoScraper struct {
	client       client.OracleClient
	mb           *metadata.MetricsBuilder
	logger       *zap.Logger
	instanceName string
	config       metadata.MetricsBuilderConfig

	// Cache static info to avoid repeated DB queries
	cachedInfo      *DatabaseInfo
	cacheValidUntil time.Time
	cacheDuration   time.Duration
	cacheMutex      sync.RWMutex // Thread-safe cache access
}

// DatabaseInfo holds database and system environment details
type DatabaseInfo struct {
	// DB version info from Oracle instance
	Version     string
	VersionFull string
	Edition     string
	Compatible  string

	// System environment detected from host
	HostingProvider     string // aws, azure, gcp, oci, kubernetes, container, on-premises
	DeploymentPlatform  string // ec2, lambda, aks, docker, etc.
	CombinedEnvironment string // provider.platform format
	Architecture        string // amd64, arm64, 386
	OperatingSystem     string // linux, windows, darwin
}

// NewDatabaseInfoScraper creates a new database info scraper
func NewDatabaseInfoScraper(c client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, config metadata.MetricsBuilderConfig) *DatabaseInfoScraper {
	return &DatabaseInfoScraper{
		client:        c,
		mb:            mb,
		logger:        logger,
		instanceName:  instanceName,
		config:        config,
		cacheDuration: 1 * time.Hour, // Version info rarely changes
	}
}

func (s *DatabaseInfoScraper) ScrapeDatabaseInfo(ctx context.Context) []error {
	var errs []error

	if !s.config.Metrics.NewrelicoracledbDatabaseInfo.Enabled {
		return errs
	}

	if err := s.ensureCacheValid(ctx); err != nil {
		errs = append(errs, err)
		return errs
	}

	s.cacheMutex.RLock()
	cachedInfo := s.cachedInfo
	s.cacheMutex.RUnlock()

	if cachedInfo == nil {
		return errs
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	s.mb.RecordNewrelicoracledbDatabaseInfoDataPoint(
		now,
		int64(1),
		s.instanceName,
		cachedInfo.Version,
		cachedInfo.VersionFull,
		cachedInfo.Edition,
		cachedInfo.Compatible,
	)

	s.logger.Debug("Database info metrics recorded",
		zap.String("version", cachedInfo.Version),
		zap.String("edition", cachedInfo.Edition))

	return errs
}

func (s *DatabaseInfoScraper) ScrapeHostingInfo(ctx context.Context) []error {
	var errs []error

	if !s.config.Metrics.NewrelicoracledbHostingInfo.Enabled {
		return errs
	}

	if err := s.ensureCacheValid(ctx); err != nil {
		errs = append(errs, err)
		return errs
	}

	s.cacheMutex.RLock()
	cachedInfo := s.cachedInfo
	s.cacheMutex.RUnlock()

	if cachedInfo == nil {
		return errs
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	s.mb.RecordNewrelicoracledbHostingInfoDataPoint(
		now,
		int64(1),
		s.instanceName,
		extractCloudProviderForOTEL(cachedInfo.HostingProvider),
		cachedInfo.DeploymentPlatform,
		cachedInfo.CombinedEnvironment,
		cachedInfo.Architecture,
		cachedInfo.OperatingSystem,
	)

	s.logger.Debug("Hosting info metrics recorded",
		zap.String("provider", cachedInfo.HostingProvider),
		zap.String("platform", cachedInfo.DeploymentPlatform))

	return errs
}

// ScrapeDatabaseRole scrapes the database role information (PRIMARY, STANDBY, etc.)
func (s *DatabaseInfoScraper) ScrapeDatabaseRole(ctx context.Context) []error {
	var errs []error

	if !s.config.Metrics.NewrelicoracledbDatabaseRole.Enabled {
		return errs
	}

	role, err := s.client.QueryDatabaseRole(ctx)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	if role != nil {
		roleStr := "UNKNOWN"
		if role.DatabaseRole.Valid {
			roleStr = role.DatabaseRole.String
		}

		openMode := "UNKNOWN"
		if role.OpenMode.Valid {
			openMode = role.OpenMode.String
		}

		protectionMode := "UNKNOWN"
		if role.ProtectionMode.Valid {
			protectionMode = role.ProtectionMode.String
		}

		protectionLevel := "UNKNOWN"
		if role.ProtectionLevel.Valid {
			protectionLevel = role.ProtectionLevel.String
		}

		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordNewrelicoracledbDatabaseRoleDataPoint(
			now,
			int64(1),
			s.instanceName,
			roleStr,
			openMode,
			protectionMode,
			protectionLevel,
		)
	}

	return errs
}

func extractCloudProviderForOTEL(hostingProvider string) string {
	switch hostingProvider {
	case "aws", "azure", "gcp", "oci":
		return hostingProvider
	default:
		return ""
	}
}

func (s *DatabaseInfoScraper) ensureCacheValid(ctx context.Context) error {
	now := time.Now()

	s.cacheMutex.RLock()
	if s.cachedInfo != nil && now.Before(s.cacheValidUntil) {
		s.cacheMutex.RUnlock()
		return nil
	}
	s.cacheMutex.RUnlock()

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if s.cachedInfo != nil && now.Before(s.cacheValidUntil) {
		return nil
	}

	s.logger.Debug("Refreshing database info cache")
	return s.refreshCacheUnsafe(ctx)
}

func (s *DatabaseInfoScraper) refreshCacheUnsafe(ctx context.Context) error {
	s.logger.Debug("Executing database info query")

	metrics, err := s.client.QueryDatabaseInfo(ctx)
	if err != nil {
		return err
	}

	return s.processDatabaseInfoMetrics(metrics)
}

func (s *DatabaseInfoScraper) processDatabaseInfoMetrics(metrics []models.DatabaseInfoMetric) error {
	for _, metric := range metrics {
		cleanVersion := extractVersionFromFull(metric.VersionFull.String)
		detectedEdition := detectEditionFromVersion(metric.VersionFull.String)

		hostingProvider, deploymentPlatform, combinedEnv := detectSystemEnvironmentWithDBHints(
			metric.HostName.String,
			metric.DatabaseName.String,
			metric.PlatformName.String,
		)

		s.cachedInfo = &DatabaseInfo{
			Version:             cleanVersion,
			VersionFull:         metric.VersionFull.String,
			Edition:             detectedEdition,
			Compatible:          cleanVersion,
			HostingProvider:     hostingProvider,
			DeploymentPlatform:  deploymentPlatform,
			CombinedEnvironment: combinedEnv,
			Architecture:        runtime.GOARCH,
			OperatingSystem:     runtime.GOOS,
		}

		s.cacheValidUntil = time.Now().Add(s.cacheDuration)

		s.logger.Debug("Database info cache refreshed",
			zap.String("version", cleanVersion),
			zap.String("edition", detectedEdition),
			zap.String("hosting_provider", hostingProvider))

		break
	}

	return nil
}

func detectCloudProvider(hostname string) (provider, platform string) {
	hostname = strings.ToLower(hostname)

	if isContainerHostname(hostname) {
		return "", "container"
	}

	switch {
	case strings.Contains(hostname, "rds"):
		return "aws", "aws_rds"
	case strings.Contains(hostname, "ec2") || strings.Contains(hostname, "aws") || strings.Contains(hostname, "amazon"):
		return "aws", "aws_ec2"
	case strings.Contains(hostname, "azure-sql"):
		return "azure", "azure_sql_db"
	case strings.Contains(hostname, "azure") || strings.Contains(hostname, "microsoft"):
		return "azure", "azure_vm"
	case strings.Contains(hostname, "gcp") || strings.Contains(hostname, "google"):
		return "gcp", "gcp_compute_engine"
	case strings.Contains(hostname, "oci") || (strings.Contains(hostname, "oracle") && !strings.Contains(hostname, "database")):
		return "oci", "oci_compute"
	default:
		return "", ""
	}
}

func isContainerHostname(hostname string) bool {
	if len(hostname) == 12 {
		for _, char := range hostname {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				return false
			}
		}
		return true
	}
	return false
}

func extractArchitecture(platformName string) string {
	platformName = strings.ToLower(platformName)
	switch {
	case strings.Contains(platformName, "x86_64") || strings.Contains(platformName, "amd64"):
		return "amd64"
	case strings.Contains(platformName, "aarch64") || strings.Contains(platformName, "arm64"):
		return "arm64"
	case strings.Contains(platformName, "i386") || strings.Contains(platformName, "i686"):
		return "386"
	default:
		return "unknown"
	}
}

func extractVersionNumber(banner string) string {
	if banner == "" {
		return "unknown"
	}

	if !strings.Contains(banner, "Oracle Database") {
		versionParts := strings.Split(strings.TrimSpace(banner), ".")
		if len(versionParts) >= 2 {
			return strings.Join(versionParts[:2], ".")
		}
		return banner
	}

	if strings.Contains(banner, "Release ") {
		parts := strings.Split(banner, "Release ")
		if len(parts) > 1 {
			version := strings.Fields(parts[1])[0]
			versionParts := strings.Split(version, ".")
			if len(versionParts) >= 2 {
				return strings.Join(versionParts[:2], ".")
			}
			return version
		}
	}

	if strings.Contains(banner, "Oracle Database ") {
		parts := strings.Split(banner, "Oracle Database ")
		if len(parts) > 1 {
			versionPart := strings.Fields(parts[1])[0]
			if strings.HasSuffix(versionPart, "ai") {
				return strings.TrimSuffix(versionPart, "ai")
			}
			if strings.HasSuffix(versionPart, "c") || strings.HasSuffix(versionPart, "g") {
				return strings.TrimSuffix(strings.TrimSuffix(versionPart, "c"), "g")
			}
			return versionPart
		}
	}

	return "unknown"
}

func determineDeploymentEnvironment(cloudProvider, cloudPlatform string) string {
	if cloudProvider != "" {
		return "cloud"
	}
	if cloudPlatform == "container" {
		return "container"
	}
	return "on-premises"
}

// normalizePlatformName converts platform string to standard format
func normalizePlatformName(platform string) string {
	platform = strings.ToLower(platform)
	switch {
	case strings.Contains(platform, "linux"):
		return "linux"
	case strings.Contains(platform, "windows"):
		return "windows"
	case strings.Contains(platform, "solaris"):
		return "solaris"
	case strings.Contains(platform, "aix"):
		return "aix"
	case strings.Contains(platform, "hp-ux"):
		return "hpux"
	default:
		return "unknown"
	}
}

func detectSystemEnvironmentWithDBHints(hostName, databaseName, platformName string) (hostingProvider, deploymentPlatform, combinedEnvironment string) {
	envProvider, envPlatform, envCombined := detectSystemEnvironment()
	if envProvider != "on-premises" {
		return envProvider, envPlatform, envCombined
	}

	dbProvider, dbPlatform := detectCloudFromDatabaseHints(hostName, databaseName, platformName)
	if dbProvider != "" {
		return dbProvider, dbPlatform, dbProvider + "." + dbPlatform
	}

	return envProvider, envPlatform, envCombined
}

func detectCloudFromDatabaseHints(hostName, databaseName, platformName string) (provider, platform string) {
	hostName = strings.ToLower(hostName)
	databaseName = strings.ToLower(databaseName)
	platformName = strings.ToLower(platformName)

	if strings.Contains(hostName, "rds.amazonaws.com") ||
		strings.Contains(hostName, "rds-") ||
		strings.Contains(databaseName, "rds") {
		return "aws", "rds"
	}

	if strings.Contains(hostName, "ec2") ||
		strings.Contains(hostName, ".amazonaws.com") ||
		strings.Contains(hostName, "ip-") {
		return "aws", "ec2"
	}

	if strings.Contains(hostName, "database.windows.net") ||
		strings.Contains(hostName, "database.azure.com") {
		return "azure", "azure_sql_db"
	}

	if strings.Contains(hostName, ".cloudapp.azure.com") ||
		strings.Contains(hostName, "azure") ||
		strings.Contains(platformName, "microsoft") {
		return "azure", "azure_vm"
	}

	if strings.Contains(hostName, "googleusercontent.com") ||
		strings.Contains(hostName, "gcp-") {
		return "gcp", "cloud_sql"
	}

	if strings.Contains(hostName, "c.googlecompute.com") ||
		strings.Contains(hostName, "gce-") ||
		strings.Contains(platformName, "google") {
		return "gcp", "compute_engine"
	}

	if strings.Contains(hostName, "oraclecloud.com") ||
		strings.Contains(hostName, "oci-") ||
		(strings.Contains(platformName, "oracle") && !strings.Contains(platformName, "database")) {
		return "oci", "compute"
	}

	return "", ""
}

func detectSystemEnvironment() (hostingProvider, deploymentPlatform, combinedEnvironment string) {
	if checkEnvVar("AWS_REGION") || checkEnvVar("AWS_EXECUTION_ENV") || checkEnvVar("EC2_INSTANCE_ID") {
		if checkEnvVar("AWS_LAMBDA_FUNCTION_NAME") {
			return "aws", "lambda", "aws.lambda"
		}
		if strings.Contains(os.Getenv("AWS_EXECUTION_ENV"), "Fargate") {
			return "aws", "fargate", "aws.fargate"
		}
		return "aws", "ec2", "aws.ec2"
	}

	if checkEnvVar("AZURE_CLIENT_ID") || checkEnvVar("WEBSITE_INSTANCE_ID") || checkEnvVar("AZURE_SUBSCRIPTION_ID") {
		if checkEnvVar("WEBSITE_INSTANCE_ID") {
			return "azure", "app-service", "azure.app-service"
		}
		if checkEnvVar("AZURE_FUNCTIONS_ENVIRONMENT") {
			return "azure", "functions", "azure.functions"
		}
		return "azure", "vm", "azure.vm"
	}

	if checkEnvVar("AZURE_TENANT_ID") || checkEnvVar("MSI_ENDPOINT") || checkEnvVar("AZURE_RESOURCE_GROUP") {
		return "azure", "azure_vm", "azure.vm"
	}

	if checkEnvVar("GOOGLE_CLOUD_PROJECT") || checkEnvVar("GCP_PROJECT") || checkEnvVar("FUNCTION_NAME") {
		if checkEnvVar("FUNCTION_NAME") {
			return "gcp", "cloud-functions", "gcp.cloud-functions"
		}
		if checkEnvVar("K_SERVICE") {
			return "gcp", "cloud-run", "gcp.cloud-run"
		}
		return "gcp", "compute-engine", "gcp.compute-engine"
	}

	if checkEnvVar("OCI_RESOURCE_PRINCIPAL_VERSION") || checkEnvVar("OCI_REGION") {
		return "oci", "compute", "oci.compute"
	}

	if checkEnvVar("KUBERNETES_SERVICE_HOST") || fileExists("/var/run/secrets/kubernetes.io/serviceaccount") {
		return "kubernetes", "generic", "kubernetes.generic"
	}

	if fileExists("/.dockerenv") || containsAny(readFile("/proc/1/cgroup"), []string{"/docker/", "/kubepods/", "/containerd/"}) {
		return "container", "docker", "container.docker"
	}

	if runtime.GOOS == "linux" {
		if provider := detectCloudFromDMI(); provider != "" {
			switch provider {
			case "amazon":
				return "aws", "ec2", "aws.ec2"
			case "microsoft":
				return "azure", "azure_vm", "azure.vm"
			case "google":
				return "gcp", "compute-engine", "gcp.compute-engine"
			case "oracle":
				return "oci", "compute", "oci.compute"
			}
		}
	}

	return "on-premises", "bare-metal", "on-premises.bare-metal"
}

func checkEnvVar(name string) bool {
	return os.Getenv(name) != ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func containsAny(text string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(text, substring) {
			return true
		}
	}
	return false
}

func detectCloudFromDMI() string {
	vendor := strings.ToLower(readFile("/sys/class/dmi/id/sys_vendor"))
	product := strings.ToLower(readFile("/sys/class/dmi/id/product_name"))
	bios := strings.ToLower(readFile("/sys/class/dmi/id/bios_vendor"))

	if strings.Contains(vendor, "amazon") || strings.Contains(product, "amazon") ||
		strings.Contains(bios, "amazon") || strings.Contains(product, "ec2") {
		return "amazon"
	}

	if strings.Contains(vendor, "microsoft") || strings.Contains(product, "virtual machine") {
		return "microsoft"
	}

	if strings.Contains(vendor, "google") || strings.Contains(product, "google") ||
		strings.Contains(product, "google compute engine") {
		return "google"
	}

	if strings.Contains(vendor, "oracle") && strings.Contains(product, "oracle") {
		return "oracle"
	}

	return ""
}

func extractVersionFromFull(versionFull string) string {
	if versionFull == "" {
		return "unknown"
	}

	versionParts := strings.Split(strings.TrimSpace(versionFull), ".")
	if len(versionParts) >= 2 {
		return strings.Join(versionParts[:2], ".")
	}

	if len(versionParts) == 1 {
		return versionParts[0]
	}

	return versionFull
}

func detectEditionFromVersion(versionFull string) string {
	if versionFull == "" {
		return "unknown"
	}

	majorVersion := strings.Split(versionFull, ".")[0]

	if majorVersion == "23" {
		return "free"
	}

	return "standard"
}
