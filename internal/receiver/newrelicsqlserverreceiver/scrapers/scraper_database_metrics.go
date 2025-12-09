// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/queries"
)

// DatabaseScraper handles SQL Server database-level metrics collection
type DatabaseScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	mb            *metadata.MetricsBuilder
	engineEdition int
}

// NewDatabaseScraper creates a new database scraper
func NewDatabaseScraper(conn SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder, engineEdition int) *DatabaseScraper {
	return &DatabaseScraper{
		connection:    conn,
		logger:        logger,
		mb:            mb,
		engineEdition: engineEdition,
	}
}

// ScrapeDatabaseBufferMetrics collects database-level buffer pool metrics using engine-specific queries
func (s *DatabaseScraper) ScrapeDatabaseBufferMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database buffer metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.buffer_pool_size")
	if !found {
		return fmt.Errorf("no database buffer pool query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database buffer pool query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabaseBufferMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database buffer query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database buffer query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database buffer query - no databases available")
		return fmt.Errorf("no results returned from database buffer query")
	}

	s.logger.Debug("Processing database buffer metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's buffer metrics
	for _, result := range results {
		if result.BufferPoolSizeBytes == nil {
			s.logger.Warn("Buffer pool size is null for database",
				zap.String("database_name", result.DatabaseName))
			continue
		}

		if err := s.processDatabaseBufferMetrics(result); err != nil {
			s.logger.Error("Failed to process database buffer metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			continue
		}

		s.logger.Debug("Successfully processed database buffer metrics",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("buffer_pool_size_bytes", *result.BufferPoolSizeBytes))
	}

	s.logger.Debug("Successfully scraped database buffer metrics",
		zap.Int("database_count", len(results)))

	return nil
}

// getQueryForMetric retrieves the appropriate query for a metric based on engine edition
func (s *DatabaseScraper) getQueryForMetric(metricName string) (string, bool) {
	query, found := queries.GetQueryForMetric(queries.DatabaseQueries, metricName, s.engineEdition)
	if found {
		s.logger.Debug("Using query for metric",
			zap.String("metric_name", metricName),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
	}
	return query, found
}

// processDatabaseBufferMetrics processes buffer pool metrics and creates OpenTelemetry metrics
func (s *DatabaseScraper) processDatabaseBufferMetrics(result models.DatabaseBufferMetrics) error {
	if result.BufferPoolSizeBytes == nil {
		s.logger.Warn("Buffer pool size is null for database",
			zap.String("database_name", result.DatabaseName))
		return nil
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	s.mb.RecordSqlserverDatabaseBufferpoolSizePerDatabaseInBytesDataPoint(now, *result.BufferPoolSizeBytes, result.DatabaseName, "sys.dm_os_buffer_descriptors", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

	s.logger.Debug("Successfully recorded database buffer metric",
		zap.String("database_name", result.DatabaseName),
		zap.Int64("buffer_pool_size_bytes", *result.BufferPoolSizeBytes))

	return nil
}

// ScrapeDatabaseDiskMetrics collects database-level disk metrics using engine-specific queries
// Only available for Azure SQL Database (engine edition 5)
func (s *DatabaseScraper) ScrapeDatabaseDiskMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database disk metrics")

	// Only collect disk metrics for Azure SQL Database
	if s.engineEdition != queries.AzureSQLDatabaseEngineEdition {
		s.logger.Debug("Skipping database disk metrics - not supported for this engine edition",
			zap.Int("engine_edition", s.engineEdition),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
		return nil
	}

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.max_disk_size")
	if !found {
		return fmt.Errorf("no database disk metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database disk metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabaseDiskMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database disk query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database disk query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database disk query - no databases available")
		return fmt.Errorf("no results returned from database disk query")
	}

	s.logger.Debug("Processing database disk metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's disk metrics
	for _, result := range results {
		if result.MaxDiskSizeBytes == nil {
			s.logger.Warn("Max disk size is null for database",
				zap.String("database_name", result.DatabaseName))
			continue
		}

		if err := s.processDatabaseDiskMetrics(result); err != nil {
			s.logger.Error("Failed to process database disk metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			continue
		}

		s.logger.Debug("Successfully processed database disk metrics",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("max_disk_size_bytes", *result.MaxDiskSizeBytes))
	}

	s.logger.Debug("Successfully scraped database disk metrics",
		zap.Int("database_count", len(results)))

	return nil
}

// processDatabaseDiskMetrics processes disk metrics and creates OpenTelemetry metrics
func (s *DatabaseScraper) processDatabaseDiskMetrics(result models.DatabaseDiskMetrics) error {
	if result.MaxDiskSizeBytes != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordSqlserverDatabaseMaxDiskSizeInBytesDataPoint(now, *result.MaxDiskSizeBytes, result.DatabaseName, "DATABASEPROPERTYEX", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

		s.logger.Debug("Recorded database disk metric",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("max_disk_size_bytes", *result.MaxDiskSizeBytes))
	}

	return nil
}

// ScrapeDatabaseIOMetrics collects database-level IO stall metrics using engine-specific queries
func (s *DatabaseScraper) ScrapeDatabaseIOMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database IO metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.io_stall")
	if !found {
		return fmt.Errorf("no database IO metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database IO metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabaseIOMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database IO query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database IO query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database IO query - no databases available")
		return fmt.Errorf("no results returned from database IO query")
	}

	s.logger.Debug("Processing database IO metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's IO metrics
	for _, result := range results {
		if result.IOStallTimeMs == nil {
			s.logger.Warn("IO stall time is null for database",
				zap.String("database_name", result.DatabaseName))
			continue
		}

		if err := s.processDatabaseIOMetrics(result); err != nil {
			s.logger.Error("Failed to process database IO metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			continue
		}

		s.logger.Info("Successfully scraped SQL Server database IO metric",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("io_stall_time_ms", *result.IOStallTimeMs))

		s.logger.Debug("Successfully processed database IO metrics",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("io_stall_time_ms", *result.IOStallTimeMs))
	}

	s.logger.Debug("Successfully scraped database IO metrics",
		zap.Int("database_count", len(results)))

	return nil
}

// processDatabaseIOMetrics processes IO metrics and creates OpenTelemetry metrics
func (s *DatabaseScraper) processDatabaseIOMetrics(result models.DatabaseIOMetrics) error {
	if result.IOStallTimeMs != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordSqlserverDatabaseIoStallInMillisecondsDataPoint(now, *result.IOStallTimeMs, result.DatabaseName, "sys.dm_io_virtual_file_stats", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

		s.logger.Debug("Recorded database IO stall metric",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("io_stall_time_ms", *result.IOStallTimeMs))
	}

	return nil
}

// ScrapeDatabaseLogGrowthMetrics collects log growth metrics for SQL Server databases
func (s *DatabaseScraper) ScrapeDatabaseLogGrowthMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database log growth metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.log_growth")
	if !found {
		return fmt.Errorf("no database log growth metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database log growth metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabaseLogGrowthMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database log growth query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database log growth query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database log growth query - no databases available")
		return fmt.Errorf("no results returned from database log growth query")
	}

	s.logger.Debug("Processing database log growth metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's log growth metrics
	for _, result := range results {
		if result.LogGrowthCount == nil {
			s.logger.Warn("Log growth count is null for database",
				zap.String("database_name", result.DatabaseName))
			continue
		}

		if err := s.processDatabaseLogGrowthMetrics(result); err != nil {
			s.logger.Error("Failed to process database log growth metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			continue
		}
	}

	s.logger.Debug("Successfully processed database log growth metrics",
		zap.Int("metrics_processed", len(results)))

	return nil
}

// processDatabaseLogGrowthMetrics processes individual database log growth metrics using reflection
func (s *DatabaseScraper) processDatabaseLogGrowthMetrics(result models.DatabaseLogGrowthMetrics) error {
	if result.LogGrowthCount != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordSqlserverDatabaseLogTransactionGrowthDataPoint(now, *result.LogGrowthCount, result.DatabaseName, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

		s.logger.Debug("Recorded database log growth metric",
			zap.String("database_name", result.DatabaseName),
			zap.Int64("log_growth_count", *result.LogGrowthCount))
	}

	return nil
}

// ScrapeDatabasePageFileMetrics scrapes page file metrics for all databases
func (s *DatabaseScraper) ScrapeDatabasePageFileMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database page file metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.page_file_available")
	if !found {
		return fmt.Errorf("no database page file query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database page file query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Check if this is a database-specific query that needs database iteration
	if strings.Contains(query, "%DATABASE%") {
		return s.scrapeDatabasePageFileWithIteration(ctx, query)
	}

	// For Azure SQL Database, execute the query directly
	var results []models.DatabasePageFileMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute page file query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)))
		return fmt.Errorf("failed to execute page file query: %w", err)
	}

	// Process results
	for _, result := range results {
		if result.PageFileAvailableBytes != nil {
			s.logger.Info("Successfully scraped SQL Server database page file metric",
				zap.String("database_name", result.DatabaseName),
				zap.Float64("page_file_available_bytes", *result.PageFileAvailableBytes))

			if err := s.processDatabasePageFileMetrics(result); err != nil {
				s.logger.Error("Failed to process database page file metrics",
					zap.String("database_name", result.DatabaseName),
					zap.Error(err))
				continue
			}
		}
	}

	s.logger.Debug("Successfully completed database page file metrics scraping")
	return nil
}

// scrapeDatabasePageFileWithIteration handles page file metrics for engines that require database iteration
func (s *DatabaseScraper) scrapeDatabasePageFileWithIteration(ctx context.Context, queryTemplate string) error {
	// Get the appropriate database list query for this engine edition
	databasesQuery, found := s.getQueryForMetric("sqlserver.database.list")
	if !found {
		return fmt.Errorf("no database list query available for engine edition %d", s.engineEdition)
	}

	var databases []struct {
		Name string `db:"name"`
	}

	if err := s.connection.Query(ctx, &databases, databasesQuery); err != nil {
		return fmt.Errorf("failed to get database list: %w", err)
	}

	s.logger.Debug("Found databases for page file metrics",
		zap.Int("database_count", len(databases)))

	// Iterate through each database and collect page file metrics
	for _, db := range databases {
		s.logger.Debug("Processing page file metrics for database",
			zap.String("database_name", db.Name))

		// Replace the %DATABASE% placeholder with actual database name
		query := strings.ReplaceAll(queryTemplate, "%DATABASE%", db.Name)

		var results []models.DatabasePageFileMetrics
		if err := s.connection.Query(ctx, &results, query); err != nil {
			s.logger.Error("Failed to execute page file metrics query for database",
				zap.String("database_name", db.Name),
				zap.Error(err))
			continue
		}

		// Process results for this database
		for _, result := range results {
			if result.PageFileAvailableBytes != nil {
				s.logger.Info("Successfully scraped SQL Server database page file metric",
					zap.String("database_name", result.DatabaseName),
					zap.Float64("page_file_available_bytes", *result.PageFileAvailableBytes))

				if err := s.processDatabasePageFileMetrics(result); err != nil {
					s.logger.Error("Failed to process database page file metrics",
						zap.String("database_name", result.DatabaseName),
						zap.Error(err))
					continue
				}
			}
		}
	}

	return nil
}

// ScrapeDatabasePageFileTotalMetrics scrapes page file total metrics for all databases
func (s *DatabaseScraper) ScrapeDatabasePageFileTotalMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database page file total metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.page_file_total")
	if !found {
		return fmt.Errorf("no database page file total query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database page file total query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Check if this is a database-specific query that needs database iteration
	if strings.Contains(query, "%DATABASE%") {
		return s.scrapeDatabasePageFileTotalWithIteration(ctx, query)
	}

	// For Azure SQL Database, execute the query directly
	var results []models.DatabasePageFileTotalMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute page file total query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)))
		return fmt.Errorf("failed to execute page file total query: %w", err)
	}

	// Process results
	for _, result := range results {
		if result.PageFileTotalBytes != nil {
			s.logger.Info("Successfully scraped SQL Server database page file total metric",
				zap.String("database_name", result.DatabaseName),
				zap.Float64("page_file_total_bytes", *result.PageFileTotalBytes))

			if err := s.processDatabasePageFileTotalMetrics(result); err != nil {
				s.logger.Error("Failed to process database page file total metrics",
					zap.String("database_name", result.DatabaseName),
					zap.Error(err))
				continue
			}
		}
	}

	s.logger.Debug("Successfully completed database page file total metrics scraping")
	return nil
}

// scrapeDatabasePageFileTotalWithIteration handles page file total metrics for engines that require database iteration
func (s *DatabaseScraper) scrapeDatabasePageFileTotalWithIteration(ctx context.Context, queryTemplate string) error {
	// Get the appropriate database list query for this engine edition
	databasesQuery, found := s.getQueryForMetric("sqlserver.database.list")
	if !found {
		return fmt.Errorf("no database list query available for engine edition %d", s.engineEdition)
	}

	var databases []struct {
		Name string `db:"name"`
	}

	if err := s.connection.Query(ctx, &databases, databasesQuery); err != nil {
		return fmt.Errorf("failed to get database list: %w", err)
	}

	s.logger.Debug("Found databases for page file total metrics",
		zap.Int("database_count", len(databases)))

	// Iterate through each database and collect page file total metrics
	for _, db := range databases {
		s.logger.Debug("Processing page file total metrics for database",
			zap.String("database_name", db.Name))

		// Replace the %DATABASE% placeholder with actual database name
		query := strings.ReplaceAll(queryTemplate, "%DATABASE%", db.Name)

		var results []models.DatabasePageFileTotalMetrics
		if err := s.connection.Query(ctx, &results, query); err != nil {
			s.logger.Error("Failed to execute page file total metrics query for database",
				zap.String("database_name", db.Name),
				zap.Error(err))
			continue
		}

		// Process results for this database
		for _, result := range results {
			if result.PageFileTotalBytes != nil {
				s.logger.Info("Successfully scraped SQL Server database page file total metric",
					zap.String("database_name", result.DatabaseName),
					zap.Float64("page_file_total_bytes", *result.PageFileTotalBytes))

				if err := s.processDatabasePageFileTotalMetrics(result); err != nil {
					s.logger.Error("Failed to process database page file total metrics",
						zap.String("database_name", result.DatabaseName),
						zap.Error(err))
					continue
				}
			}
		}
	}

	return nil
}

// processDatabasePageFileMetrics processes individual database page file metrics using reflection
func (s *DatabaseScraper) processDatabasePageFileMetrics(result models.DatabasePageFileMetrics) error {
	if result.PageFileAvailableBytes != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordSqlserverDatabasePageFileAvailableDataPoint(now, *result.PageFileAvailableBytes, result.DatabaseName, "sys.partitions_sys.allocation_units", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

		s.logger.Debug("Recorded database page file available metric",
			zap.String("database_name", result.DatabaseName),
			zap.Float64("page_file_available_bytes", *result.PageFileAvailableBytes))
	}

	return nil
}

// processDatabasePageFileTotalMetrics processes individual database page file total metrics using reflection
func (s *DatabaseScraper) processDatabasePageFileTotalMetrics(result models.DatabasePageFileTotalMetrics) error {
	if result.PageFileTotalBytes != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordSqlserverDatabasePageFileTotalDataPoint(now, *result.PageFileTotalBytes, result.DatabaseName, "sys.partitions_sys.allocation_units", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

		s.logger.Debug("Recorded database page file total metric",
			zap.String("database_name", result.DatabaseName),
			zap.Float64("page_file_total_bytes", *result.PageFileTotalBytes))
	}

	return nil
}

// ScrapeDatabaseMemoryMetrics scrapes available physical memory metrics
// This is an instance-level metric that provides system memory information
// Note: Memory metrics are only available for Azure SQL Database (engine edition 5)
func (s *DatabaseScraper) ScrapeDatabaseMemoryMetrics(ctx context.Context) error {
	s.logger.Debug("Starting ScrapeDatabaseMemoryMetrics")

	// Memory metrics are only supported for Azure SQL Database (engine edition 5)
	if s.engineEdition != queries.AzureSQLDatabaseEngineEdition {
		s.logger.Debug("Memory metrics are only available for Azure SQL Database, skipping",
			zap.String("current_engine", queries.GetEngineTypeName(s.engineEdition)),
			zap.Int("engine_edition", s.engineEdition))
		return nil
	}

	// Get the appropriate query based on the engine edition
	query, found := s.getQueryForMetric("sqlserver.database_memory_metrics")
	if !found {
		return fmt.Errorf("memory query not found for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing memory query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Query the database for memory metrics
	var results []models.DatabaseMemoryMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute memory query", zap.Error(err))
		return fmt.Errorf("failed to execute memory query: %w", err)
	}

	s.logger.Debug("Memory query completed", zap.Int("result_count", len(results)))

	// Process each result using reflection
	return s.processDatabaseMemoryResults(results)
}

// processDatabaseMemoryResults processes the memory query results and creates OpenTelemetry metrics
func (s *DatabaseScraper) processDatabaseMemoryResults(results []models.DatabaseMemoryMetrics) error {
	if len(results) == 0 {
		s.logger.Debug("No memory metrics results to process")
		return nil
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	for _, result := range results {
		if result.AvailablePhysicalMemoryBytes != nil {
			s.mb.RecordSqlserverInstanceMemoryAvailableDataPoint(now, *result.AvailablePhysicalMemoryBytes)
			s.logger.Debug("Recorded instance memory metric",
				zap.Float64("available_physical_memory_bytes", *result.AvailablePhysicalMemoryBytes))
		}
	}

	return nil
}

// ScrapeDatabaseSizeMetrics scrapes basic database size metrics for SQL Server databases
func (s *DatabaseScraper) ScrapeDatabaseSizeMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database size metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.size_metrics")
	if !found {
		return fmt.Errorf("no database size metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database size metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabaseSizeMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database size query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database size query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database size query")
		return fmt.Errorf("no results returned from database size query")
	}

	s.logger.Debug("Processing database size metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's size metrics
	for _, result := range results {
		if err := s.processDatabaseSizeMetrics(result); err != nil {
			s.logger.Error("Failed to process database size metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			continue
		}
	}

	s.logger.Debug("Successfully scraped database size metrics",
		zap.Int("metric_count", len(results)))

	return nil
}

// processDatabaseSizeMetrics processes individual database size metrics using reflection
func (s *DatabaseScraper) processDatabaseSizeMetrics(result models.DatabaseSizeMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	if result.TotalSizeMB != nil {
		s.mb.RecordSqlserverDatabaseSizeTotalSizeMBDataPoint(now, *result.TotalSizeMB, result.DatabaseName, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))
		s.logger.Debug("Recorded database total size metric",
			zap.String("database_name", result.DatabaseName),
			zap.Float64("total_size_mb", *result.TotalSizeMB))
	}
	if result.DataSizeMB != nil {
		s.mb.RecordSqlserverDatabaseSizeDataSizeMBDataPoint(now, *result.DataSizeMB, result.DatabaseName, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))
		s.logger.Debug("Recorded database data size metric",
			zap.String("database_name", result.DatabaseName),
			zap.Float64("data_size_mb", *result.DataSizeMB))
	}

	return nil
}

// ScrapeDatabaseTransactionLogMetrics scrapes transaction log performance metrics for SQL Server databases
func (s *DatabaseScraper) ScrapeDatabaseTransactionLogMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database transaction log metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.transaction_log_metrics")
	if !found {
		return fmt.Errorf("no database transaction log metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database transaction log metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabaseTransactionLogMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database transaction log query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database transaction log query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database transaction log query")
		return fmt.Errorf("no results returned from database transaction log query")
	}

	s.logger.Debug("Processing database transaction log metrics results",
		zap.Int("result_count", len(results)))

	// Process each result (should be only one for the aggregated counters)
	for _, result := range results {
		if err := s.processDatabaseTransactionLogMetrics(result); err != nil {
			s.logger.Error("Failed to process database transaction log metrics",
				zap.Error(err))
			continue
		}
	}

	s.logger.Debug("Successfully scraped database transaction log metrics",
		zap.Int("metric_count", len(results)))

	return nil
}

// processDatabaseTransactionLogMetrics processes individual database transaction log metrics using reflection
func (s *DatabaseScraper) processDatabaseTransactionLogMetrics(result models.DatabaseTransactionLogMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	if result.LogFlushesPerSec != nil {
		s.mb.RecordSqlserverDatabaseLogFlushesPerSecDataPoint(now, *result.LogFlushesPerSec, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))
		s.logger.Debug("Recorded log flushes per sec metric",
			zap.Int64("log_flushes_per_sec", *result.LogFlushesPerSec))
	}
	if result.LogBytesFlushesPerSec != nil {
		s.mb.RecordSqlserverDatabaseLogBytesFlushesPerSecDataPoint(now, *result.LogBytesFlushesPerSec, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))
		s.logger.Debug("Recorded log bytes flushes per sec metric",
			zap.Int64("log_bytes_flushes_per_sec", *result.LogBytesFlushesPerSec))
	}
	if result.FlushWaitsPerSec != nil {
		s.mb.RecordSqlserverDatabaseLogFlushWaitsPerSecDataPoint(now, *result.FlushWaitsPerSec, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))
		s.logger.Debug("Recorded log flush waits per sec metric",
			zap.Int64("log_flush_waits_per_sec", *result.FlushWaitsPerSec))
	}
	if result.ActiveTransactions != nil {
		s.mb.RecordSqlserverDatabaseTransactionsActiveDataPoint(now, *result.ActiveTransactions, "sys.dm_os_performance_counters", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))
		s.logger.Debug("Recorded active transactions metric",
			zap.Int64("active_transactions", *result.ActiveTransactions))
	}

	return nil
}

// ScrapeDatabaseLogSpaceUsageMetrics scrapes database log space usage metrics for SQL Server databases
func (s *DatabaseScraper) ScrapeDatabaseLogSpaceUsageMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database log space usage metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.database.log_space_usage_metrics")
	if !found {
		return fmt.Errorf("no database log space usage metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database log space usage metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Check if this is a database-specific query that needs database iteration
	if s.engineEdition == queries.StandardSQLServerEngineEdition {
		// For default SQL Server, we need to iterate through databases
		return s.scrapeDatabaseLogSpaceUsageWithIteration(ctx, query)
	}

	// For Azure SQL Database and Azure SQL Managed Instance, execute directly
	var results []models.DatabaseLogSpaceUsageMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database log space usage query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute database log space usage query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from database log space usage query")
		return fmt.Errorf("no results returned from database log space usage query")
	}

	s.logger.Debug("Processing database log space usage metrics results",
		zap.Int("result_count", len(results)))

	// Process each database's log space usage metrics
	for _, result := range results {
		if err := s.processDatabaseLogSpaceUsageMetrics(result); err != nil {
			s.logger.Error("Failed to process database log space usage metrics",
				zap.Error(err))
			continue
		}
	}

	s.logger.Debug("Successfully scraped database log space usage metrics",
		zap.Int("metric_count", len(results)))

	return nil
}

// scrapeDatabaseLogSpaceUsageWithIteration handles log space usage metrics for engines that require database iteration
func (s *DatabaseScraper) scrapeDatabaseLogSpaceUsageWithIteration(ctx context.Context, queryTemplate string) error {
	// Get the appropriate database list query for this engine edition
	databasesQuery, found := s.getQueryForMetric("sqlserver.database.list")
	if !found {
		return fmt.Errorf("no database list query available for engine edition %d", s.engineEdition)
	}

	var databases []struct {
		Name string `db:"name"`
	}

	if err := s.connection.Query(ctx, &databases, databasesQuery); err != nil {
		return fmt.Errorf("failed to get database list: %w", err)
	}

	s.logger.Debug("Found databases for log space usage metrics",
		zap.Int("database_count", len(databases)))

	// Iterate through each database and collect log space usage metrics
	for _, db := range databases {
		s.logger.Debug("Processing log space usage metrics for database",
			zap.String("database_name", db.Name))

		// Execute query in the context of each database
		// For log space usage, we need to USE the database and then query sys.dm_db_log_space_usage
		contextualQuery := fmt.Sprintf("USE [%s]; %s", db.Name, queryTemplate)

		var results []models.DatabaseLogSpaceUsageMetrics
		if err := s.connection.Query(ctx, &results, contextualQuery); err != nil {
			s.logger.Error("Failed to execute log space usage metrics query for database",
				zap.String("database_name", db.Name),
				zap.Error(err))
			continue
		}

		// Process results for this database
		for _, result := range results {
			// Set the database name since the query doesn't include it
			// We need to create a new struct with the database name
			enrichedResult := models.DatabaseLogSpaceUsageMetrics{
				UsedLogSpaceMB: result.UsedLogSpaceMB,
			}

			if err := s.processDatabaseLogSpaceUsageMetricsWithDBName(enrichedResult, db.Name); err != nil {
				s.logger.Error("Failed to process database log space usage metrics",
					zap.String("database_name", db.Name),
					zap.Error(err))
				continue
			}

			if enrichedResult.UsedLogSpaceMB != nil {
				s.logger.Info("Successfully scraped SQL Server database log space usage metrics",
					zap.String("database_name", db.Name),
					zap.Float64("used_log_space_mb", *enrichedResult.UsedLogSpaceMB))
			}
		}
	}

	return nil
}

// processDatabaseLogSpaceUsageMetrics processes individual database log space usage metrics using reflection
func (s *DatabaseScraper) processDatabaseLogSpaceUsageMetrics(result models.DatabaseLogSpaceUsageMetrics) error {
	// For Azure editions, we need to determine the database name from the context
	databaseName := "current_database" // Default for Azure SQL Database
	return s.processDatabaseLogSpaceUsageMetricsWithDBName(result, databaseName)
}

// processDatabaseLogSpaceUsageMetricsWithDBName processes individual database log space usage metrics with database name
func (s *DatabaseScraper) processDatabaseLogSpaceUsageMetricsWithDBName(result models.DatabaseLogSpaceUsageMetrics, databaseName string) error {
	if result.UsedLogSpaceMB != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		s.mb.RecordSqlserverDatabaseLogUsedSpaceMBDataPoint(now, *result.UsedLogSpaceMB, databaseName, "sys.dm_os_sys_memory", queries.GetEngineTypeName(s.engineEdition), int64(s.engineEdition))

		s.logger.Debug("Recorded database log space usage metric",
			zap.String("database_name", databaseName),
			zap.Float64("used_log_space_mb", *result.UsedLogSpaceMB))
	}

	return nil
}

/* Duplicate ScrapeDatabaseDiskMetrics removed to resolve redeclaration error. */

/* Duplicate processDatabaseDiskMetrics removed to resolve redeclaration error. */
