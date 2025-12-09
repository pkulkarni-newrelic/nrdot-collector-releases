// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/queries"
)

// LockScraper handles SQL Server lock metrics collection
type LockScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	mb            *metadata.MetricsBuilder
	engineEdition int
}

// NewLockScraper creates a new lock scraper
func NewLockScraper(conn SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder, engineEdition int) *LockScraper {
	return &LockScraper{
		connection:    conn,
		logger:        logger,
		mb:            mb,
		engineEdition: engineEdition,
	}
}

// getQueryForMetric retrieves the appropriate query for a metric based on engine edition with Default fallback
func (s *LockScraper) getQueryForMetric(metricName string) (string, bool) {
	query, found := queries.GetQueryForMetric(queries.LockQueries, metricName, s.engineEdition)
	if found {
		s.logger.Debug("Using query for metric",
			zap.String("metric_name", metricName),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
	}
	return query, found
}

// ScrapeLockResourceMetrics collects lock resource metrics using engine-specific queries
func (s *LockScraper) ScrapeLockResourceMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server lock resource metrics")

	// Check if this metric is compatible with the current engine edition
	if !queries.IsMetricCompatible(queries.LockQueries, "sqlserver.lock.resource", s.engineEdition) {
		s.logger.Debug("Lock resource metrics not supported for this engine edition",
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)),
			zap.Int("engine_edition", s.engineEdition))
		return nil // Return nil to indicate successful skip, not an error
	}

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.lock.resource")
	if !found {
		return fmt.Errorf("no lock resource query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing lock resource query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.LockResourceSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute lock resource query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute lock resource query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Debug("No lock resource results returned - no active locks")
		return nil
	}

	s.logger.Debug("Processing lock resource metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's lock resource metrics
	for _, result := range results {
		if result.DatabaseName == nil {
			s.logger.Warn("Database name is null in lock resource results")
			continue
		}

		if err := s.processLockResourceMetrics(result); err != nil {
			s.logger.Error("Failed to process lock resource metrics",
				zap.Error(err),
				zap.String("database_name", *result.DatabaseName))
			continue
		}

		s.logger.Debug("Successfully processed lock resource metrics",
			zap.String("database_name", *result.DatabaseName),
			zap.Int64("total_active_locks", getInt64Value(result.TotalActiveLocks)))
	}

	s.logger.Debug("Successfully scraped lock resource metrics",
		zap.Int("database_count", len(results)))

	return nil
}

// ScrapeLockModeMetrics collects lock mode metrics using engine-specific queries
func (s *LockScraper) ScrapeLockModeMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server lock mode metrics")

	// Check if this metric is compatible with the current engine edition
	if !queries.IsMetricCompatible(queries.LockQueries, "sqlserver.lock.mode", s.engineEdition) {
		s.logger.Debug("Lock mode metrics not supported for this engine edition",
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)),
			zap.Int("engine_edition", s.engineEdition))
		return nil // Return nil to indicate successful skip, not an error
	}

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.lock.mode")
	if !found {
		return fmt.Errorf("no lock mode query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing lock mode query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.LockModeSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute lock mode query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute lock mode query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Debug("No lock mode results returned - no active locks")
		return nil
	}

	s.logger.Debug("Processing lock mode metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's lock mode metrics
	for _, result := range results {
		if result.DatabaseName == nil {
			s.logger.Warn("Database name is null in lock mode results")
			continue
		}

		if err := s.processLockModeMetrics(result); err != nil {
			s.logger.Error("Failed to process lock mode metrics",
				zap.Error(err),
				zap.String("database_name", *result.DatabaseName))
			continue
		}

		s.logger.Debug("Successfully processed lock mode metrics",
			zap.String("database_name", *result.DatabaseName),
			zap.Int64("total_active_locks", getInt64Value(result.TotalActiveLocks)))
	}

	s.logger.Debug("Successfully scraped lock mode metrics",
		zap.Int("database_count", len(results)))

	return nil
}

// processLockResourceMetrics processes lock resource metrics and creates OpenTelemetry metrics
func (s *LockScraper) processLockResourceMetrics(result models.LockResourceSummary) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	databaseName := ""
	if result.DatabaseName != nil {
		databaseName = *result.DatabaseName
	}

	if result.TotalActiveLocks != nil {
		s.mb.RecordSqlserverLockResourceTotalDataPoint(now, int64(*result.TotalActiveLocks), databaseName)
	}

	if result.TableLocks != nil {
		s.mb.RecordSqlserverLockResourceTableDataPoint(now, int64(*result.TableLocks), databaseName)
	}

	if result.PageLocks != nil {
		s.mb.RecordSqlserverLockResourcePageDataPoint(now, int64(*result.PageLocks), databaseName)
	}

	if result.RowLocks != nil {
		s.mb.RecordSqlserverLockResourceRowDataPoint(now, int64(*result.RowLocks), databaseName)
	}

	if result.KeyLocks != nil {
		s.mb.RecordSqlserverLockResourceKeyDataPoint(now, int64(*result.KeyLocks), databaseName)
	}

	if result.ExtentLocks != nil {
		s.mb.RecordSqlserverLockResourceExtentDataPoint(now, int64(*result.ExtentLocks), databaseName)
	}

	if result.DatabaseLocks != nil {
		s.mb.RecordSqlserverLockResourceDatabaseLevelDataPoint(now, int64(*result.DatabaseLocks), databaseName)
	}

	if result.FileLocks != nil {
		s.mb.RecordSqlserverLockResourceFileDataPoint(now, int64(*result.FileLocks), databaseName)
	}

	if result.ApplicationLocks != nil {
		s.mb.RecordSqlserverLockResourceApplicationDataPoint(now, int64(*result.ApplicationLocks), databaseName)
	}

	if result.MetadataLocks != nil {
		s.mb.RecordSqlserverLockResourceMetadataDataPoint(now, int64(*result.MetadataLocks), databaseName)
	}

	if result.HobtLocks != nil {
		s.mb.RecordSqlserverLockResourceHobtDataPoint(now, int64(*result.HobtLocks), databaseName)
	}

	if result.AllocationUnitLocks != nil {
		s.mb.RecordSqlserverLockResourceAllocationUnitDataPoint(now, int64(*result.AllocationUnitLocks), databaseName)
	}

	return nil
}

// processLockModeMetrics processes lock mode metrics and creates OpenTelemetry metrics
func (s *LockScraper) processLockModeMetrics(result models.LockModeSummary) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	databaseName := ""
	if result.DatabaseName != nil {
		databaseName = *result.DatabaseName
	}

	if result.TotalActiveLocks != nil {
		s.mb.RecordSqlserverLockModeTotalDataPoint(now, int64(*result.TotalActiveLocks), databaseName)
	}

	if result.SharedLocks != nil {
		s.mb.RecordSqlserverLockModeSharedDataPoint(now, int64(*result.SharedLocks), databaseName)
	}

	if result.ExclusiveLocks != nil {
		s.mb.RecordSqlserverLockModeExclusiveDataPoint(now, int64(*result.ExclusiveLocks), databaseName)
	}

	if result.UpdateLocks != nil {
		s.mb.RecordSqlserverLockModeUpdateDataPoint(now, int64(*result.UpdateLocks), databaseName)
	}

	if result.IntentLocks != nil {
		s.mb.RecordSqlserverLockModeIntentDataPoint(now, int64(*result.IntentLocks), databaseName)
	}

	if result.SchemaLocks != nil {
		s.mb.RecordSqlserverLockModeSchemaDataPoint(now, int64(*result.SchemaLocks), databaseName)
	}

	if result.BulkUpdateLocks != nil {
		s.mb.RecordSqlserverLockModeBulkUpdateDataPoint(now, int64(*result.BulkUpdateLocks), databaseName)
	}

	if result.SharedIntentExclusiveLocks != nil {
		s.mb.RecordSqlserverLockModeSharedIntentExclusiveDataPoint(now, int64(*result.SharedIntentExclusiveLocks), databaseName)
	}

	return nil
}

// getInt64Value safely extracts int64 value from pointer, returns 0 if nil
func getInt64Value(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
