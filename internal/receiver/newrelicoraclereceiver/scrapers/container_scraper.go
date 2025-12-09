// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/errors"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

// ContainerScraper handles Oracle CDB/PDB container metrics
type ContainerScraper struct {
	client             client.OracleClient
	mb                 *metadata.MetricsBuilder
	logger             *zap.Logger
	instanceName       string
	config             metadata.MetricsBuilderConfig
	isCDBCapable       *bool    // Cache for CDB capability check
	isPDBCapable       *bool    // Cache for PDB capability check
	environmentChecked bool     // Track if environment has been checked
	currentContainer   string   // Current container context (CDB$ROOT, FREEPDB1, etc.)
	currentContainerID string   // Current container ID (1, 3, etc.)
	contextChecked     bool     // Track if context has been checked
	includeTablespaces []string // Tablespace filter: include list
	excludeTablespaces []string // Tablespace filter: exclude list
}

// NewContainerScraper creates a new Container scraper
func NewContainerScraper(
	oracleClient client.OracleClient,
	mb *metadata.MetricsBuilder,
	logger *zap.Logger,
	instanceName string,
	config metadata.MetricsBuilderConfig,
	includeTablespaces, excludeTablespaces []string,
) (*ContainerScraper, error) {
	if oracleClient == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if mb == nil {
		return nil, fmt.Errorf("metrics builder cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if instanceName == "" {
		return nil, fmt.Errorf("instance name cannot be empty")
	}

	return &ContainerScraper{
		client:             oracleClient,
		mb:                 mb,
		logger:             logger,
		instanceName:       instanceName,
		config:             config,
		includeTablespaces: includeTablespaces,
		excludeTablespaces: excludeTablespaces,
	}, nil
}

// ScrapeContainerMetrics collects Oracle CDB/PDB container metrics
func (s *ContainerScraper) ScrapeContainerMetrics(ctx context.Context) []error {
	var errors []error

	s.logger.Debug("Scraping Oracle CDB/PDB container metrics")

	// Check environment capability first
	if err := s.checkEnvironmentCapability(ctx); err != nil {
		s.logger.Error("Failed to check Oracle environment capabilities", zap.Error(err))
		return []error{err}
	}

	// Skip if CDB/PDB features are not supported
	if !s.isCDBSupported() {
		s.logger.Debug("CDB features not supported, skipping container metrics")
		return errors
	}

	// Check current container context
	if err := s.checkCurrentContext(ctx); err != nil {
		s.logger.Error("Failed to check Oracle container context", zap.Error(err))
		return []error{err}
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	// Scrape container status metrics (only from CDB$ROOT)
	if s.isConnectedToCDBRoot() {
		errors = append(errors, s.scrapeContainerStatus(ctx, now)...)
	} else {
		s.logger.Debug("Not connected to CDB$ROOT, skipping container status metrics",
			zap.String("current_container", s.currentContainer))
	}

	// Scrape PDB status metrics (only if PDB is supported and connected to CDB$ROOT)
	if s.isPDBSupported() && s.isConnectedToCDBRoot() {
		errors = append(errors, s.scrapePDBStatus(ctx, now)...)
	} else {
		s.logger.Debug("PDB features not supported or not in CDB$ROOT, skipping PDB metrics",
			zap.String("current_container", s.currentContainer))
	}

	// Scrape CDB tablespace usage (only from CDB$ROOT)
	if s.isConnectedToCDBRoot() {
		errors = append(errors, s.scrapeCDBTablespaceUsage(ctx, now)...)
	} else {
		s.logger.Debug("Not connected to CDB$ROOT, skipping CDB tablespace metrics",
			zap.String("current_container", s.currentContainer))
	}

	// Scrape CDB data files (only from CDB$ROOT)
	if s.isConnectedToCDBRoot() {
		errors = append(errors, s.scrapeCDBDataFiles(ctx, now)...)
	} else {
		s.logger.Debug("Not connected to CDB$ROOT, skipping CDB data file metrics",
			zap.String("current_container", s.currentContainer))
	}

	// Scrape CDB services (only from CDB$ROOT)
	if s.isConnectedToCDBRoot() {
		errors = append(errors, s.scrapeCDBServices(ctx, now)...)
	} else {
		s.logger.Debug("Not connected to CDB$ROOT, skipping CDB service metrics",
			zap.String("current_container", s.currentContainer))
	}

	return errors
}

// scrapeContainerStatus scrapes container status from GV$CONTAINERS
func (s *ContainerScraper) scrapeContainerStatus(ctx context.Context, now pcommon.Timestamp) []error {
	var errs []error

	// Check if this metric is enabled in config
	if !s.config.Metrics.NewrelicoracledbContainerStatus.Enabled &&
		!s.config.Metrics.NewrelicoracledbContainerRestricted.Enabled {
		s.logger.Debug("Container status metrics disabled, skipping")
		return errs
	}

	containers, err := s.client.QueryContainerStatus(ctx)
	if err != nil {
		s.logger.Error("Failed to query container status", zap.Error(err))
		return []error{err}
	}

	for _, container := range containers {
		// Validate required fields
		if !container.ConID.Valid || !container.ContainerName.Valid {
			continue
		}

		conIDStr := strconv.FormatInt(container.ConID.Int64, 10)
		containerNameStr := container.ContainerName.String
		openModeStr := ""
		restrictedStr := ""

		if container.OpenMode.Valid {
			openModeStr = container.OpenMode.String
		}
		if container.Restricted.Valid {
			restrictedStr = container.Restricted.String
		}

		// Record metrics only if enabled
		if s.config.Metrics.NewrelicoracledbContainerStatus.Enabled {
			// Container status metric (1=READ WRITE, 0=other)
			var statusValue int64 = 0
			if strings.ToUpper(openModeStr) == "READ WRITE" {
				statusValue = 1
			}
			s.mb.RecordNewrelicoracledbContainerStatusDataPoint(now, statusValue, s.instanceName, conIDStr, containerNameStr, openModeStr)
		}

		if s.config.Metrics.NewrelicoracledbContainerRestricted.Enabled {
			// Container restricted metric (1=YES, 0=NO)
			var restrictedValue int64 = 0
			if strings.ToUpper(restrictedStr) == "YES" {
				restrictedValue = 1
			}
			s.mb.RecordNewrelicoracledbContainerRestrictedDataPoint(now, restrictedValue, s.instanceName, conIDStr, containerNameStr, restrictedStr)
		}

		s.logger.Debug("Processed container status",
			zap.String("con_id", conIDStr),
			zap.String("container_name", containerNameStr),
			zap.String("open_mode", openModeStr),
			zap.String("restricted", restrictedStr))
	}

	return errs
}

// scrapePDBStatus scrapes PDB status from GV$PDBS
func (s *ContainerScraper) scrapePDBStatus(ctx context.Context, now pcommon.Timestamp) []error {
	s.logger.Info("Starting PDB status scraping")

	pdbs, err := s.client.QueryPDBStatus(ctx)
	if err != nil {
		s.logger.Error("Failed to execute PDB status query", zap.Error(err))
		return []error{err}
	}

	s.logger.Info("Successfully queried PDB status", zap.Int("pdb_count", len(pdbs)))

	for _, pdb := range pdbs {
		if !pdb.ConID.Valid || !pdb.PDBName.Valid {
			s.logger.Warn("Skipping PDB with invalid ConID or PDBName")
			continue
		}

		conIDStr := strconv.FormatInt(pdb.ConID.Int64, 10)
		pdbNameStr := pdb.PDBName.String
		openModeStr := ""
		if pdb.OpenMode.Valid {
			openModeStr = pdb.OpenMode.String
		}

		s.logger.Info("Processing PDB",
			zap.String("con_id", conIDStr),
			zap.String("pdb_name", pdbNameStr),
			zap.String("open_mode", openModeStr),
			zap.Bool("total_size_valid", pdb.TotalSize.Valid),
			zap.Int64("total_size_value", pdb.TotalSize.Int64))

		// PDB open mode metric (1=READ WRITE, 0=other)
		var openModeValue int64 = 0
		if strings.ToUpper(openModeStr) == "READ WRITE" {
			openModeValue = 1
		}
		s.mb.RecordNewrelicoracledbPdbOpenModeDataPoint(now, openModeValue, s.instanceName, conIDStr, pdbNameStr, openModeStr)

		// PDB total size metric
		if pdb.TotalSize.Valid {
			s.logger.Info("Recording PDB total size metric",
				zap.String("pdb_name", pdbNameStr),
				zap.Int64("size_bytes", pdb.TotalSize.Int64),
				zap.String("instance", s.instanceName))
			s.mb.RecordNewrelicoracledbPdbTotalSizeBytesDataPoint(now, pdb.TotalSize.Int64, s.instanceName, conIDStr, pdbNameStr)
		} else {
			s.logger.Warn("PDB total size is NULL, skipping metric",
				zap.String("pdb_name", pdbNameStr))
		}

		s.logger.Debug("Processed PDB status",
			zap.String("con_id", conIDStr),
			zap.String("pdb_name", pdbNameStr),
			zap.String("open_mode", openModeStr))
	}

	s.logger.Info("Completed PDB status scraping", zap.Int("pdbs_processed", len(pdbs)))
	return nil
}

// scrapeCDBTablespaceUsage scrapes CDB tablespace usage from CDB_TABLESPACE_USAGE_METRICS
func (s *ContainerScraper) scrapeCDBTablespaceUsage(ctx context.Context, now pcommon.Timestamp) []error {
	tablespaces, err := s.client.QueryCDBTablespaceUsage(ctx, s.includeTablespaces, s.excludeTablespaces)
	if err != nil {
		s.logger.Error("Failed to execute CDB tablespace usage query", zap.Error(err))
		return []error{err}
	}

	for _, ts := range tablespaces {
		if !ts.ConID.Valid || !ts.TablespaceName.Valid {
			continue
		}

		conIDStr := strconv.FormatInt(ts.ConID.Int64, 10)
		tablespaceName := ts.TablespaceName.String

		// Record tablespace usage metrics with container tagging
		if ts.UsedBytes.Valid {
			s.mb.RecordNewrelicoracledbTablespaceUsedBytesDataPoint(now, ts.UsedBytes.Int64, s.instanceName, conIDStr, tablespaceName)
		}

		if ts.TotalBytes.Valid {
			s.mb.RecordNewrelicoracledbTablespaceTotalBytesDataPoint(now, ts.TotalBytes.Int64, s.instanceName, conIDStr, tablespaceName)
		}

		if ts.UsedPercent.Valid {
			s.mb.RecordNewrelicoracledbTablespaceUsedPercentDataPoint(now, ts.UsedPercent.Float64, s.instanceName, conIDStr, tablespaceName)
		}

		s.logger.Debug("Processed CDB tablespace usage",
			zap.String("con_id", conIDStr),
			zap.String("tablespace_name", tablespaceName),
			zap.Int64("used_bytes", ts.UsedBytes.Int64),
			zap.Int64("total_bytes", ts.TotalBytes.Int64))
	}

	return nil
}

// scrapeCDBDataFiles scrapes CDB data files from CDB_DATA_FILES
func (s *ContainerScraper) scrapeCDBDataFiles(ctx context.Context, now pcommon.Timestamp) []error {
	datafiles, err := s.client.QueryCDBDataFiles(ctx)
	if err != nil {
		s.logger.Error("Failed to execute CDB data files query", zap.Error(err))
		return []error{err}
	}

	for _, df := range datafiles {
		if !df.ConID.Valid || !df.FileName.Valid || !df.TablespaceName.Valid {
			continue
		}

		conIDStr := strconv.FormatInt(df.ConID.Int64, 10)
		fileName := df.FileName.String
		tablespaceName := df.TablespaceName.String
		autoextensibleStr := ""

		if df.Autoextensible.Valid {
			autoextensibleStr = df.Autoextensible.String
		}

		// Record data file size
		if df.Bytes.Valid {
			s.mb.RecordNewrelicoracledbDatafileSizeBytesDataPoint(now, df.Bytes.Int64, s.instanceName, conIDStr, tablespaceName, fileName)
		}

		// Record user bytes
		if df.UserBytes.Valid {
			s.mb.RecordNewrelicoracledbDatafileUsedBytesDataPoint(now, df.UserBytes.Int64, s.instanceName, conIDStr, tablespaceName, fileName)
		}

		// Record autoextensible status (1=YES, 0=NO)
		var autoextensibleValue int64 = 0
		if strings.ToUpper(autoextensibleStr) == "YES" {
			autoextensibleValue = 1
		}
		s.mb.RecordNewrelicoracledbDatafileAutoextensibleDataPoint(now, autoextensibleValue, s.instanceName, conIDStr, tablespaceName, fileName, autoextensibleStr)

		s.logger.Debug("Processed CDB data file",
			zap.String("con_id", conIDStr),
			zap.String("tablespace_name", tablespaceName),
			zap.String("file_name", fileName),
			zap.Int64("bytes", df.Bytes.Int64))
	}

	return nil
}

// scrapeCDBServices scrapes CDB services from CDB_SERVICES
func (s *ContainerScraper) scrapeCDBServices(ctx context.Context, now pcommon.Timestamp) []error {
	services, err := s.client.QueryCDBServices(ctx)
	if err != nil {
		s.logger.Error("Failed to execute CDB services query", zap.Error(err))
		return []error{err}
	}

	serviceCount := make(map[string]int64)

	for _, svc := range services {
		if !svc.ConID.Valid || !svc.ServiceName.Valid {
			continue
		}

		conIDStr := strconv.FormatInt(svc.ConID.Int64, 10)
		serviceName := svc.ServiceName.String

		// Count services per container
		serviceCount[conIDStr]++

		// Record service status (1=enabled if enabled='YES', 0=disabled)
		var serviceStatus int64 = 0
		if svc.Enabled.Valid && strings.ToUpper(svc.Enabled.String) == "YES" {
			serviceStatus = 1
		}
		s.mb.RecordNewrelicoracledbServiceStatusDataPoint(now, serviceStatus, s.instanceName, conIDStr, serviceName)

		s.logger.Debug("Processed CDB service",
			zap.String("con_id", conIDStr),
			zap.String("service_name", serviceName),
			zap.String("pdb", svc.PDB.String),
			zap.String("enabled", svc.Enabled.String),
			zap.Int64("status", serviceStatus))
	}

	// Record service count per container
	for conIDStr, count := range serviceCount {
		s.mb.RecordNewrelicoracledbServiceCountDataPoint(now, count, s.instanceName, conIDStr)
	}

	return nil
}

// checkEnvironmentCapability checks if the Oracle database supports CDB/PDB features
func (s *ContainerScraper) checkEnvironmentCapability(ctx context.Context) error {
	if s.environmentChecked {
		return nil // Already checked
	}

	// Check if this is a CDB-capable database
	isCDB, err := s.client.CheckCDBFeature(ctx)
	if err != nil {
		if errors.IsPermanentError(err) {
			// Likely an older Oracle version that doesn't support CDB
			s.logger.Info("Database does not support CDB features",
				zap.String("instance", s.instanceName),
				zap.Error(err))
			cdbCapable := false
			pdbCapable := false
			s.isCDBCapable = &cdbCapable
			s.isPDBCapable = &pdbCapable
			s.environmentChecked = true
			return nil
		}
		return err
	}

	cdbCapable := isCDB == 1
	s.isCDBCapable = &cdbCapable

	// Check if PDB views are available
	if cdbCapable {
		pdbCount, err := s.client.CheckPDBCapability(ctx)
		if err != nil {
			return err
		}
		pdbCapable := pdbCount > 0
		s.isPDBCapable = &pdbCapable
	} else {
		pdbCapable := false
		s.isPDBCapable = &pdbCapable
	}

	s.environmentChecked = true
	s.logger.Info("Detected Oracle environment capabilities",
		zap.String("instance", s.instanceName),
		zap.Bool("cdb_capable", *s.isCDBCapable),
		zap.Bool("pdb_capable", *s.isPDBCapable))

	return nil
}

// isCDBSupported returns true if the database supports CDB features
func (s *ContainerScraper) isCDBSupported() bool {
	return s.isCDBCapable != nil && *s.isCDBCapable
}

// isPDBSupported returns true if the database supports PDB features
func (s *ContainerScraper) isPDBSupported() bool {
	return s.isPDBCapable != nil && *s.isPDBCapable
}

// checkCurrentContext determines which container context we're connected to
func (s *ContainerScraper) checkCurrentContext(ctx context.Context) error {
	if s.contextChecked {
		return nil
	}

	s.logger.Debug("Checking current Oracle container context",
		zap.String("instance", s.instanceName))

	// Query current container context
	containerContext, err := s.client.CheckCurrentContainer(ctx)
	if err != nil {
		return err
	}

	if containerContext.ContainerName.Valid {
		s.currentContainer = containerContext.ContainerName.String
	}
	if containerContext.ContainerID.Valid {
		s.currentContainerID = containerContext.ContainerID.String
	}
	s.contextChecked = true

	s.logger.Info("Detected Oracle container context",
		zap.String("instance", s.instanceName),
		zap.String("container_name", s.currentContainer),
		zap.String("container_id", s.currentContainerID))

	return nil
}

// isConnectedToCDBRoot returns true if connected to CDB$ROOT
func (s *ContainerScraper) isConnectedToCDBRoot() bool {
	return s.currentContainer == "CDB$ROOT"
}

// isConnectedToPDB returns true if connected to a specific PDB
func (s *ContainerScraper) isConnectedToPDB() bool {
	return s.currentContainer != "" && s.currentContainer != "CDB$ROOT"
}
