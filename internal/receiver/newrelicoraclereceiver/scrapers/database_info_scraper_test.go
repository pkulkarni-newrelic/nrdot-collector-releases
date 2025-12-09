// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

func TestNewDatabaseInfoScraper(t *testing.T) {
	mockClient := &client.MockClient{}
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper := NewDatabaseInfoScraper(mockClient, mb, logger, "test-instance", config)

	assert.NotNil(t, scraper)
	assert.Equal(t, mockClient, scraper.client)
	assert.Equal(t, mb, scraper.mb)
	assert.Equal(t, logger, scraper.logger)
	assert.Equal(t, "test-instance", scraper.instanceName)
	assert.Equal(t, 1*time.Hour, scraper.cacheDuration)
}

func TestExtractCloudProviderForOTEL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"AWS", "aws", "aws"},
		{"Azure", "azure", "azure"},
		{"GCP", "gcp", "gcp"},
		{"OCI", "oci", "oci"},
		{"Kubernetes", "kubernetes", ""},
		{"Container", "container", ""},
		{"On-premises", "on-premises", ""},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCloudProviderForOTEL(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsContainerHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{"Valid Docker container ID", "a1b2c3d4e5f6", true},
		{"Valid hex 12 chars", "0123456789ab", true},
		{"Invalid - too short", "a1b2c3d4e5", false},
		{"Invalid - too long", "a1b2c3d4e5f67", false},
		{"Invalid - contains non-hex", "a1b2c3g4e5f6", false},
		{"Invalid - uppercase", "A1B2C3D4E5F6", false},
		{"Regular hostname", "db-server-01", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContainerHostname(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractArchitecture(t *testing.T) {
	tests := []struct {
		name         string
		platformName string
		expected     string
	}{
		{"x86_64", "Linux x86_64", "amd64"},
		{"amd64", "Linux amd64", "amd64"},
		{"aarch64", "Linux aarch64", "arm64"},
		{"arm64", "Linux arm64", "arm64"},
		{"i386", "Linux i386", "386"},
		{"i686", "Linux i686", "386"},
		{"Unknown", "Linux unknown", "unknown"},
		{"Empty", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractArchitecture(tt.platformName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVersionNumber(t *testing.T) {
	tests := []struct {
		name     string
		banner   string
		expected string
	}{
		{
			"Oracle 23ai Free",
			"Oracle Database 23ai Free Release 23.0.0.0.0 - Develop, Learn, and Run for Free",
			"23.0",
		},
		{
			"Oracle 19c Enterprise",
			"Oracle Database 19c Enterprise Edition Release 19.0.0.0.0 - Production",
			"19.0",
		},
		{
			"Oracle 21c Express",
			"Oracle Database 21c Express Edition Release 21.0.0.0.0 - Production",
			"21.0",
		},
		{
			"Oracle 11g Enterprise",
			"Oracle Database 11g Enterprise Edition Release 11.2.0.4.0 - 64bit Production",
			"11.2",
		},
		{
			"Direct version format",
			"23.9.0.25.07",
			"23.9",
		},
		{
			"Simple version",
			"19.3.0.0.0",
			"19.3",
		},
		{
			"Single part version",
			"23",
			"23",
		},
		{
			"Empty banner",
			"",
			"unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionNumber(tt.banner)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVersionFromFull(t *testing.T) {
	tests := []struct {
		name        string
		versionFull string
		expected    string
	}{
		{"Oracle 23", "23.0.0.0.0", "23.0"},
		{"Oracle 19", "19.3.0.0.0", "19.3"},
		{"Oracle 21", "21.0.0.0.0", "21.0"},
		{"Oracle 11", "11.2.0.4.0", "11.2"},
		{"Single part", "23", "23"},
		{"Empty", "", "unknown"},
		{"Two parts", "19.3", "19.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromFull(tt.versionFull)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectEditionFromVersion(t *testing.T) {
	tests := []struct {
		name        string
		versionFull string
		expected    string
	}{
		{"Oracle 23 (Free)", "23.0.0.0.0", "free"},
		{"Oracle 19 (Standard)", "19.3.0.0.0", "standard"},
		{"Oracle 21 (Standard)", "21.0.0.0.0", "standard"},
		{"Oracle 11 (Standard)", "11.2.0.4.0", "standard"},
		{"Empty", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectEditionFromVersion(tt.versionFull)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectCloudProvider(t *testing.T) {
	tests := []struct {
		name             string
		hostname         string
		expectedProvider string
		expectedPlatform string
	}{
		{"AWS RDS", "mydb.rds.amazonaws.com", "aws", "aws_rds"},
		{"AWS EC2", "ec2-instance.compute.amazonaws.com", "aws", "aws_ec2"},
		{"Azure SQL", "mydb.database.windows.net", "azure", "azure_sql_db"},
		{"Azure VM", "myvm.azure.microsoft.com", "azure", "azure_vm"},
		{"GCP", "mydb.gcp.google.com", "gcp", "gcp_compute_engine"},
		{"OCI", "mydb.oraclecloud.com", "oci", "oci_compute"},
		{"Docker container", "a1b2c3d4e5f6", "", "container"},
		{"On-premises", "db-server.company.local", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, platform := detectCloudProvider(tt.hostname)
			assert.Equal(t, tt.expectedProvider, provider)
			assert.Equal(t, tt.expectedPlatform, platform)
		})
	}
}

func TestDetectCloudFromDatabaseHints(t *testing.T) {
	tests := []struct {
		name             string
		hostName         string
		databaseName     string
		platformName     string
		expectedProvider string
		expectedPlatform string
	}{
		{
			"AWS RDS by hostname",
			"mydb.rds.amazonaws.com",
			"mydb",
			"Linux x86_64",
			"aws",
			"rds",
		},
		{
			"AWS EC2 by hostname",
			"ip-10-0-1-100.ec2.internal",
			"mydb",
			"Linux x86_64",
			"aws",
			"ec2",
		},
		{
			"Azure SQL by hostname",
			"myserver.database.windows.net",
			"mydb",
			"Linux x86_64",
			"azure",
			"azure_sql_db",
		},
		{
			"Azure VM",
			"myvm.cloudapp.azure.com",
			"mydb",
			"Linux x86_64",
			"azure",
			"azure_vm",
		},
		{
			"GCP Cloud SQL",
			"mydb.googleusercontent.com",
			"mydb",
			"Linux x86_64",
			"gcp",
			"cloud_sql",
		},
		{
			"GCP Compute Engine",
			"instance-1.c.googlecompute.com",
			"mydb",
			"Linux x86_64",
			"gcp",
			"compute_engine",
		},
		{
			"OCI",
			"mydb.oraclecloud.com",
			"mydb",
			"Linux x86_64",
			"oci",
			"compute",
		},
		{
			"On-premises",
			"db-server",
			"mydb",
			"Linux x86_64",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, platform := detectCloudFromDatabaseHints(tt.hostName, tt.databaseName, tt.platformName)
			assert.Equal(t, tt.expectedProvider, provider)
			assert.Equal(t, tt.expectedPlatform, platform)
		})
	}
}

func TestCheckEnvVar(t *testing.T) {
	t.Setenv("TEST_VAR", "test_value")

	assert.True(t, checkEnvVar("TEST_VAR"))
	assert.False(t, checkEnvVar("NON_EXISTENT_VAR"))
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		substrings []string
		expected   bool
	}{
		{"Contains first", "hello world", []string{"hello", "foo"}, true},
		{"Contains second", "hello world", []string{"foo", "world"}, true},
		{"Contains none", "hello world", []string{"foo", "bar"}, false},
		{"Empty text", "", []string{"hello"}, false},
		{"Empty substrings", "hello world", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.text, tt.substrings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScrapeDatabaseInfo_MetricDisabled(t *testing.T) {
	mockClient := &client.MockClient{}
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDatabaseInfo.Enabled = false
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper := NewDatabaseInfoScraper(mockClient, mb, logger, "test-instance", config)
	ctx := context.Background()

	errs := scraper.ScrapeDatabaseInfo(ctx)
	assert.Empty(t, errs)
}

func TestScrapeHostingInfo_MetricDisabled(t *testing.T) {
	mockClient := &client.MockClient{}
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbHostingInfo.Enabled = false
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper := NewDatabaseInfoScraper(mockClient, mb, logger, "test-instance", config)
	ctx := context.Background()

	errs := scraper.ScrapeHostingInfo(ctx)
	assert.Empty(t, errs)
}

func TestEnsureCacheValid_UsesCacheWhenValid(t *testing.T) {
	mockClient := &client.MockClient{}
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper := NewDatabaseInfoScraper(mockClient, mb, logger, "test-instance", config)

	scraper.cachedInfo = &DatabaseInfo{
		Version:     "19.3",
		VersionFull: "19.3.0.0.0",
		Edition:     "standard",
	}
	scraper.cacheValidUntil = time.Now().Add(30 * time.Minute)

	ctx := context.Background()
	err := scraper.ensureCacheValid(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "19.3", scraper.cachedInfo.Version)
}

func TestNormalizePlatformName(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{"Linux", "Linux x86_64", "linux"},
		{"Windows", "Windows Server 2019", "windows"},
		{"Solaris", "Solaris 11", "solaris"},
		{"AIX", "AIX 7.2", "aix"},
		{"HP-UX", "HP-UX 11i", "hpux"},
		{"Unknown", "FreeBSD", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePlatformName(tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineDeploymentEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		cloudProvider string
		cloudPlatform string
		expected      string
	}{
		{"AWS", "aws", "ec2", "cloud"},
		{"Azure", "azure", "vm", "cloud"},
		{"GCP", "gcp", "compute-engine", "cloud"},
		{"Container", "", "container", "container"},
		{"On-premises", "", "bare-metal", "on-premises"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineDeploymentEnvironment(tt.cloudProvider, tt.cloudPlatform)
			assert.Equal(t, tt.expected, result)
		})
	}
}
