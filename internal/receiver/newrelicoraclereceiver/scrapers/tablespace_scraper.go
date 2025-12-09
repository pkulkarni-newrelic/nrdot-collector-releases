// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/errors"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

type TablespaceScraper struct {
	client             client.OracleClient
	mb                 *metadata.MetricsBuilder
	logger             *zap.Logger
	instanceName       string
	config             metadata.MetricsBuilderConfig
	includeTablespaces []string
	excludeTablespaces []string

	isCDBCapable       *bool
	isPDBCapable       *bool
	environmentChecked bool
	currentContainer   string
	currentContainerID string
	contextChecked     bool
	detectionMutex     sync.RWMutex
}

func NewTablespaceScraper(c client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, config metadata.MetricsBuilderConfig, includeTablespaces, excludeTablespaces []string) *TablespaceScraper {
	return &TablespaceScraper{
		client:             c,
		mb:                 mb,
		logger:             logger,
		instanceName:       instanceName,
		config:             config,
		includeTablespaces: includeTablespaces,
		excludeTablespaces: excludeTablespaces,
	}
}

func (s *TablespaceScraper) ScrapeTablespaceMetrics(ctx context.Context) []error {
	var errors []error
	metricCount := 0

	if err := s.checkEnvironmentCapability(ctx); err != nil {
		return []error{err}
	}

	if err := s.checkCurrentContext(ctx); err != nil {
		return []error{err}
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	errors = append(errors, s.scrapeTablespaceUsageMetrics(ctx, now, &metricCount)...)
	errors = append(errors, s.scrapeGlobalNameTablespaceMetrics(ctx, now, &metricCount)...)
	errors = append(errors, s.scrapeDBIDTablespaceMetrics(ctx, now, &metricCount)...)

	if s.isCDBSupported() {
		errors = append(errors, s.scrapeCDBDatafilesOfflineTablespaceMetrics(ctx, now, &metricCount)...)

		if s.isPDBSupported() {
			errors = append(errors, s.scrapePDBDatafilesOfflineTablespaceMetrics(ctx, now, &metricCount)...)
			errors = append(errors, s.scrapePDBNonWriteTablespaceMetrics(ctx, now, &metricCount)...)
		}
	}

	s.logger.Debug("Tablespace metrics scrape completed",
		zap.Int("metrics", metricCount),
		zap.Int("errors", len(errors)))

	return errors
}

func (s *TablespaceScraper) scrapeTablespaceUsageMetrics(ctx context.Context, now pcommon.Timestamp, metricCount *int) []error {
	if !s.isAnyTablespaceMetricEnabled() {
		return nil
	}

	tablespaces, err := s.client.QueryTablespaceUsage(ctx, s.includeTablespaces, s.excludeTablespaces)
	if err != nil {
		return []error{fmt.Errorf("error executing tablespace metrics query: %w", err)}
	}

	return s.processTablespaceUsage(tablespaces, now, metricCount)
}

// isAnyTablespaceMetricEnabled checks if any tablespace usage metric is enabled
func (s *TablespaceScraper) isAnyTablespaceMetricEnabled() bool {
	return s.config.Metrics.NewrelicoracledbTablespaceSpaceConsumedBytes.Enabled ||
		s.config.Metrics.NewrelicoracledbTablespaceSpaceReservedBytes.Enabled ||
		s.config.Metrics.NewrelicoracledbTablespaceSpaceUsedPercentage.Enabled ||
		s.config.Metrics.NewrelicoracledbTablespaceIsOffline.Enabled
}

func (s *TablespaceScraper) processTablespaceUsage(tablespaces []models.TablespaceUsage, now pcommon.Timestamp, metricCount *int) []error {
	for _, ts := range tablespaces {
		if s.config.Metrics.NewrelicoracledbTablespaceSpaceConsumedBytes.Enabled {
			s.mb.RecordNewrelicoracledbTablespaceSpaceConsumedBytesDataPoint(now, int64(ts.Used), s.instanceName, ts.TablespaceName)
		}
		if s.config.Metrics.NewrelicoracledbTablespaceSpaceReservedBytes.Enabled {
			s.mb.RecordNewrelicoracledbTablespaceSpaceReservedBytesDataPoint(now, int64(ts.Size), s.instanceName, ts.TablespaceName)
		}
		if s.config.Metrics.NewrelicoracledbTablespaceSpaceUsedPercentage.Enabled {
			s.mb.RecordNewrelicoracledbTablespaceSpaceUsedPercentageDataPoint(now, int64(ts.UsedPercent), s.instanceName, ts.TablespaceName)
		}
		if s.config.Metrics.NewrelicoracledbTablespaceIsOffline.Enabled {
			s.mb.RecordNewrelicoracledbTablespaceIsOfflineDataPoint(now, int64(ts.Offline), s.instanceName, ts.TablespaceName)
		}

		*metricCount++
	}

	return nil
}

func (s *TablespaceScraper) scrapeGlobalNameTablespaceMetrics(ctx context.Context, now pcommon.Timestamp, metricCount *int) []error {
	if !s.config.Metrics.NewrelicoracledbTablespaceGlobalName.Enabled {
		return nil
	}

	tablespaces, err := s.client.QueryTablespaceGlobalName(ctx, s.includeTablespaces, s.excludeTablespaces)
	if err != nil {
		return []error{fmt.Errorf("error executing global name tablespace query: %w", err)}
	}

	for _, ts := range tablespaces {
		s.mb.RecordNewrelicoracledbTablespaceGlobalNameDataPoint(now, 1, s.instanceName, ts.TablespaceName, ts.GlobalName)
		*metricCount++
	}

	return nil
}

func (s *TablespaceScraper) scrapeDBIDTablespaceMetrics(ctx context.Context, now pcommon.Timestamp, metricCount *int) []error {
	if !s.config.Metrics.NewrelicoracledbTablespaceDbID.Enabled {
		return nil
	}

	tablespaces, err := s.client.QueryTablespaceDBID(ctx, s.includeTablespaces, s.excludeTablespaces)
	if err != nil {
		return []error{fmt.Errorf("error executing DB ID tablespace query: %w", err)}
	}

	for _, ts := range tablespaces {
		s.mb.RecordNewrelicoracledbTablespaceDbIDDataPoint(now, ts.DBID, s.instanceName, ts.TablespaceName, strconv.FormatInt(ts.DBID, 10))
		*metricCount++
	}

	return nil
}

func (s *TablespaceScraper) scrapeCDBDatafilesOfflineTablespaceMetrics(ctx context.Context, now pcommon.Timestamp, metricCount *int) []error {
	if !s.config.Metrics.NewrelicoracledbTablespaceOfflineCdbDatafiles.Enabled {
		return nil
	}

	tablespaces, err := s.client.QueryTablespaceCDBDatafilesOffline(ctx, s.includeTablespaces, s.excludeTablespaces)
	if err != nil {
		return []error{fmt.Errorf("error executing CDB datafiles offline tablespace query: %w", err)}
	}

	for _, ts := range tablespaces {
		s.mb.RecordNewrelicoracledbTablespaceOfflineCdbDatafilesDataPoint(now, ts.OfflineCount, s.instanceName, ts.TablespaceName)
		*metricCount++
	}

	return nil
}

func (s *TablespaceScraper) scrapePDBDatafilesOfflineTablespaceMetrics(ctx context.Context, now pcommon.Timestamp, metricCount *int) []error {
	if !s.config.Metrics.NewrelicoracledbTablespaceOfflinePdbDatafiles.Enabled {
		return nil
	}

	var tablespaces []models.TablespacePDBDatafilesOffline
	var err error

	if s.isConnectedToCDBRoot() {
		tablespaces, err = s.client.QueryTablespacePDBDatafilesOffline(ctx, s.includeTablespaces, s.excludeTablespaces)
	} else if s.isConnectedToPDB() {
		tablespaces, err = s.client.QueryTablespacePDBDatafilesOfflineCurrentContainer(ctx, s.includeTablespaces, s.excludeTablespaces)
	} else {
		return nil
	}

	if err != nil {
		return []error{fmt.Errorf("error executing PDB datafiles offline query: %w", err)}
	}

	for _, ts := range tablespaces {
		s.mb.RecordNewrelicoracledbTablespaceOfflinePdbDatafilesDataPoint(now, ts.OfflineCount, s.instanceName, ts.TablespaceName)
		*metricCount++
	}

	return nil
}

func (s *TablespaceScraper) scrapePDBNonWriteTablespaceMetrics(ctx context.Context, now pcommon.Timestamp, metricCount *int) []error {
	if !s.config.Metrics.NewrelicoracledbTablespacePdbNonWriteMode.Enabled {
		return nil
	}

	var tablespaces []models.TablespacePDBNonWrite
	var err error

	if s.isConnectedToCDBRoot() {
		tablespaces, err = s.client.QueryTablespacePDBNonWrite(ctx, s.includeTablespaces, s.excludeTablespaces)
	} else if s.isConnectedToPDB() {
		tablespaces, err = s.client.QueryTablespacePDBNonWriteCurrentContainer(ctx, s.includeTablespaces, s.excludeTablespaces)
	} else {
		return nil
	}

	if err != nil {
		return []error{fmt.Errorf("error executing PDB non-write mode query: %w", err)}
	}

	for _, ts := range tablespaces {
		s.mb.RecordNewrelicoracledbTablespacePdbNonWriteModeDataPoint(now, ts.NonWriteCount, s.instanceName, ts.TablespaceName)
		*metricCount++
	}

	return nil
}

func (s *TablespaceScraper) checkEnvironmentCapability(ctx context.Context) error {
	s.detectionMutex.Lock()
	defer s.detectionMutex.Unlock()

	if s.environmentChecked {
		return nil
	}

	isCDB, err := s.client.CheckCDBFeature(ctx)
	if err != nil {
		if errors.IsPermanentError(err) {
			cdbCapable := false
			pdbCapable := false
			s.isCDBCapable = &cdbCapable
			s.isPDBCapable = &pdbCapable
			s.environmentChecked = true
			return nil
		}
		return errors.NewQueryError("cdb_capability_check", "CheckCDBFeatureSQL", err,
			map[string]interface{}{"instance": s.instanceName})
	}

	cdbCapable := isCDB == 1
	s.isCDBCapable = &cdbCapable

	if cdbCapable {
		pdbCount, err := s.client.CheckPDBCapability(ctx)
		if err != nil {
			pdbCapable := false
			s.isPDBCapable = &pdbCapable
		} else {
			pdbCapable := pdbCount > 0
			s.isPDBCapable = &pdbCapable
		}
	} else {
		pdbCapable := false
		s.isPDBCapable = &pdbCapable
	}

	s.environmentChecked = true

	return nil
}

// isCDBSupported returns true if the database supports CDB features
func (s *TablespaceScraper) isCDBSupported() bool {
	s.detectionMutex.RLock()
	defer s.detectionMutex.RUnlock()
	return s.isCDBCapable != nil && *s.isCDBCapable
}

// isPDBSupported returns true if the database supports PDB features
func (s *TablespaceScraper) isPDBSupported() bool {
	s.detectionMutex.RLock()
	defer s.detectionMutex.RUnlock()
	return s.isPDBCapable != nil && *s.isPDBCapable
}

func (s *TablespaceScraper) checkCurrentContext(ctx context.Context) error {
	s.detectionMutex.Lock()
	defer s.detectionMutex.Unlock()

	if s.contextChecked {
		return nil
	}

	container, err := s.client.CheckCurrentContainer(ctx)
	if err != nil {
		return errors.NewQueryError("container_context_check", "CheckCurrentContainerSQL", err,
			map[string]interface{}{"instance": s.instanceName})
	}

	if container.ContainerName.Valid {
		s.currentContainer = container.ContainerName.String
	}
	if container.ContainerID.Valid {
		s.currentContainerID = container.ContainerID.String
	}
	s.contextChecked = true

	return nil
}

func (s *TablespaceScraper) isConnectedToCDBRoot() bool {
	s.detectionMutex.RLock()
	defer s.detectionMutex.RUnlock()
	return s.currentContainer == "CDB$ROOT"
}

func (s *TablespaceScraper) isConnectedToPDB() bool {
	s.detectionMutex.RLock()
	defer s.detectionMutex.RUnlock()
	return s.currentContainer != "" && s.currentContainer != "CDB$ROOT"
}
