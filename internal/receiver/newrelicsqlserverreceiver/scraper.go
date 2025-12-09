// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newrelicsqlserverreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/helpers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/queries"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/scrapers"
)

// sqlServerScraper handles SQL Server metrics collection
type sqlServerScraper struct {
	connection              *SQLConnection
	config                  *Config
	logger                  *zap.Logger
	startTime               pcommon.Timestamp
	settings                receiver.Settings
	mb                      *metadata.MetricsBuilder // Shared MetricsBuilder for all scrapers (Oracle pattern)
	instanceScraper         *scrapers.InstanceScraper
	queryPerformanceScraper *scrapers.QueryPerformanceScraper
	// slowQueryScraper  *scrapers.SlowQueryScraper
	databaseScraper               *scrapers.DatabaseScraper
	userConnectionScraper         *scrapers.UserConnectionScraper
	failoverClusterScraper        *scrapers.FailoverClusterScraper
	databasePrincipalsScraper     *scrapers.DatabasePrincipalsScraper
	databaseRoleMembershipScraper *scrapers.DatabaseRoleMembershipScraper
	waitTimeScraper               *scrapers.WaitTimeScraper         // Add this line
	securityScraper               *scrapers.SecurityScraper         // Security metrics scraper
	lockScraper                   *scrapers.LockScraper             // Lock analysis metrics scraper
	threadPoolHealthScraper       *scrapers.ThreadPoolHealthScraper // Thread pool health monitoring
	tempdbContentionScraper       *scrapers.TempDBContentionScraper // TempDB contention monitoring
	metadataCache                 *helpers.MetadataCache            // Metadata cache for wait resource enrichment
	engineEdition                 int                               // SQL Server engine edition (0=Unknown, 5=Azure DB, 8=Azure MI)
}

// newSqlServerScraper creates a new SQL Server scraper with structured approach
func newSqlServerScraper(settings receiver.Settings, cfg *Config) *sqlServerScraper {
	return &sqlServerScraper{
		config:   cfg,
		logger:   settings.Logger,
		settings: settings,
	}
}

// Start initializes the scraper and establishes database connection
func (s *sqlServerScraper) Start(ctx context.Context, _ component.Host) error {
	s.logger.Info("Starting SQL Server receiver")

	connection, err := NewSQLConnection(ctx, s.config, s.logger)
	if err != nil {
		s.logger.Error("Failed to connect to SQL Server", zap.Error(err))
		return err
	}
	s.connection = connection
	s.startTime = pcommon.NewTimestampFromTime(time.Now())

	if err := s.connection.Ping(ctx); err != nil {
		s.logger.Error("Failed to ping SQL Server", zap.Error(err))
		return err
	}

	// Get EngineEdition
	s.engineEdition = 0 // Default to 0 (Unknown)
	s.engineEdition, err = s.detectEngineEdition(ctx)
	if err != nil {
		s.logger.Debug("Failed to get engine edition, using default", zap.Error(err))
		s.engineEdition = queries.StandardSQLServerEngineEdition
	} else {
		s.logger.Info("Detected SQL Server engine edition",
			zap.Int("engine_edition", s.engineEdition),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
	}

	// Create ONE MetricsBuilder that will be shared across all scrapers (Oracle pattern)
	s.mb = metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), s.settings)

	// Initialize metadata cache for wait resource enrichment if enabled
	if s.config.EnableWaitResourceEnrichment {
		refreshInterval := time.Duration(s.config.WaitResourceMetadataRefreshMinutes) * time.Minute
		s.metadataCache = helpers.NewMetadataCache(s.connection.Connection.DB, refreshInterval)

		// Perform initial cache refresh
		s.logger.Info("Initializing metadata cache for wait resource enrichment",
			zap.Int("refresh_interval_minutes", s.config.WaitResourceMetadataRefreshMinutes))

		if err := s.metadataCache.Refresh(ctx); err != nil {
			s.logger.Warn("Failed to perform initial metadata cache refresh",
				zap.Error(err))
			// Continue - cache will retry on next scrape
		} else {
			stats := s.metadataCache.GetCacheStats()
			s.logger.Info("Metadata cache initialized successfully",
				zap.Int("databases", stats["databases"]),
				zap.Int("objects", stats["objects"]),
				zap.Int("hobts", stats["hobts"]),
				zap.Int("partitions", stats["partitions"]))
		}
	} else {
		s.logger.Info("Wait resource enrichment disabled, skipping metadata cache initialization")
	}

	// Initialize instance scraper with engine edition for engine-specific queries
	// Create instance scraper for instance-level metrics
	s.instanceScraper = scrapers.NewInstanceScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Create database scraper for database-level metrics
	s.databaseScraper = scrapers.NewDatabaseScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Create failover cluster scraper for Always On Availability Group metrics
	s.failoverClusterScraper = scrapers.NewFailoverClusterScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Create database principals scraper for database security metrics
	s.databasePrincipalsScraper = scrapers.NewDatabasePrincipalsScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Create database role membership scraper for database role and membership metrics
	s.databaseRoleMembershipScraper = scrapers.NewDatabaseRoleMembershipScraper(s.logger, s.connection, s.mb, s.engineEdition)

	// Initialize query performance scraper for blocking sessions and performance monitoring
	// Pass smoothing and simplified interval calculator configuration parameters from config
	s.queryPerformanceScraper = scrapers.NewQueryPerformanceScraper(
		s.connection,
		s.logger,
		s.mb,
		s.engineEdition,
		s.config.EnableSlowQuerySmoothing,
		s.config.SlowQuerySmoothingFactor,
		s.config.SlowQuerySmoothingDecayThreshold,
		s.config.SlowQuerySmoothingMaxAgeMinutes,
		s.config.EnableIntervalBasedAveraging,
		s.config.IntervalCalculatorCacheTTLMinutes,
		s.metadataCache,
	)
	// s.slowQueryScraper = scrapers.NewSlowQueryScraper(s.logger, s.connection)

	// Initialize user connection scraper for user connection and authentication metrics
	s.userConnectionScraper = scrapers.NewUserConnectionScraper(s.connection, s.logger, s.engineEdition, s.mb)

	// Initialize wait time scraper for wait time metrics
	s.waitTimeScraper = scrapers.NewWaitTimeScraper(s.connection, s.logger, s.engineEdition, s.mb)

	// Initialize security scraper for server-level security metrics
	s.securityScraper = scrapers.NewSecurityScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Initialize lock scraper for lock analysis metrics
	s.lockScraper = scrapers.NewLockScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Initialize thread pool health scraper for thread pool monitoring
	s.threadPoolHealthScraper = scrapers.NewThreadPoolHealthScraper(s.connection, s.logger, s.mb)

	// Initialize TempDB contention scraper for TempDB monitoring
	s.tempdbContentionScraper = scrapers.NewTempDBContentionScraper(s.connection, s.logger, s.mb)

	s.logger.Info("Successfully connected to SQL Server",
		zap.String("hostname", s.config.Hostname),
		zap.String("port", s.config.Port),
		zap.Int("engine_edition", s.engineEdition),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	return nil
}

// Shutdown closes the database connection
func (s *sqlServerScraper) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down SQL Server receiver")
	if s.connection != nil {
		s.connection.Close()
	}
	return nil
}

// ScrapeLogs collects execution plan logs from SQL Server and emits them as OTLP logs
// NOW COLLECTS EXECUTION PLANS FOR ACTIVE RUNNING QUERIES ONLY (not slow queries from dm_exec_query_stats)
func (s *sqlServerScraper) ScrapeLogs(ctx context.Context) (plog.Logs, error) {
	s.logger.Info("=== scrapeLogs: Starting SQL Server logs collection for ACTIVE QUERY execution plans ===")

	// Create logs collection
	logs := plog.NewLogs()

	// Only collect execution plan logs if query monitoring is enabled
	if !s.config.EnableQueryMonitoring {
		s.logger.Warn("Query monitoring disabled, skipping execution plan logs collection")
		return logs, nil
	}

	if s.queryPerformanceScraper == nil {
		s.logger.Warn("Query performance scraper not initialized, skipping execution plan logs collection")
		return logs, nil
	}

	s.logger.Info("Query monitoring is ENABLED, collecting execution plans for ACTIVE running queries")

	// First, fetch slow query IDs for correlation with active queries
	// This ensures we only fetch execution plans for slow queries selected by the user
	intervalSeconds := s.config.QueryMonitoringFetchInterval
	topN := s.config.QueryMonitoringCountThreshold
	elapsedTimeThreshold := s.config.QueryMonitoringResponseTimeThreshold
	textTruncateLimit := s.config.QueryMonitoringTextTruncateLimit

	s.logger.Info("Fetching slow queries for active query correlation",
		zap.Int("interval_seconds", intervalSeconds),
		zap.Int("top_n", topN),
		zap.Int("elapsed_time_threshold", elapsedTimeThreshold))

	slowQueries, err := s.queryPerformanceScraper.ScrapeSlowQueryMetrics(ctx, intervalSeconds, topN, elapsedTimeThreshold, textTruncateLimit, false)
	if err != nil {
		s.logger.Warn("Failed to fetch slow queries for correlation, will skip execution plan collection",
			zap.Error(err))
		return logs, nil // Return empty logs, don't fail
	}

	// Extract query IDs from slow queries
	var slowQueryIDs []string
	if len(slowQueries) > 0 {
		slowQueryIDs = s.queryPerformanceScraper.ExtractQueryIDsFromSlowQueries(slowQueries)
		s.logger.Info("Extracted query IDs from slow queries for active query filtering",
			zap.Int("slow_query_count", len(slowQueries)),
			zap.Int("unique_query_id_count", len(slowQueryIDs)))
	} else {
		s.logger.Info("No slow queries found, skipping execution plan collection")
		return logs, nil // No slow queries, nothing to collect
	}

	// Scrape active running queries with execution plans
	// This will fetch, parse, and emit execution plan operators as OTLP logs
	limit := s.config.QueryMonitoringCountThreshold
	activeElapsedTimeThreshold := s.config.ActiveRunningQueriesElapsedTimeThreshold // Minimum elapsed time in ms

	s.logger.Info("Collecting active running query execution plans",
		zap.Int("limit", limit),
		zap.Int("text_truncate_limit", textTruncateLimit),
		zap.Int("elapsed_time_threshold_ms", activeElapsedTimeThreshold),
		zap.Int("slow_query_id_filter_count", len(slowQueryIDs)))

	// Step 1: Fetch active queries from database
	activeQueries, err := s.queryPerformanceScraper.ScrapeActiveRunningQueriesMetrics(ctx, limit, textTruncateLimit, activeElapsedTimeThreshold, slowQueryIDs)
	if err != nil {
		s.logger.Error("Failed to fetch active running queries", zap.Error(err))
		return logs, err
	}

	if len(activeQueries) == 0 {
		s.logger.Info("No active queries found matching slow query IDs")
		return logs, nil
	}

	s.logger.Info("Active queries fetched, emitting metrics",
		zap.Int("active_query_count", len(activeQueries)))

	// Step 3: Fetch execution plans and emit as logs
	if err := s.queryPerformanceScraper.EmitActiveRunningExecutionPlansAsLogs(ctx, logs, activeQueries); err != nil {
		s.logger.Error("Failed to emit execution plans as logs", zap.Error(err))
		return logs, err
	}

	logCount := logs.LogRecordCount()
	s.logger.Info("=== scrapeLogs: Completed SQL Server logs collection for ACTIVE queries ===",
		zap.Int("log_records", logCount),
		zap.Int("resource_logs", logs.ResourceLogs().Len()))

	return logs, nil
}

// formatPlanHandlesForSQL formats plan handles for use in SQL IN clause
func (s *sqlServerScraper) formatPlanHandlesForSQL(planHandles []string) string {
	if len(planHandles) == 0 {
		return "0x0"
	}

	// Join plan handles with commas for SQL query
	planHandlesString := ""
	for i, planHandle := range planHandles {
		if i > 0 {
			planHandlesString += ","
		}
		planHandlesString += planHandle
	}

	return planHandlesString
}

// convertExecutionPlansToLogs converts execution plan analysis to OTLP log records
func (s *sqlServerScraper) convertExecutionPlansToLogs(executionPlans []*models.ExecutionPlanAnalysis, logs plog.Logs) {
	if len(executionPlans) == 0 {
		return
	}

	resourceLogs := logs.ResourceLogs().AppendEmpty()

	// Set resource attributes following OpenTelemetry semantic conventions
	resourceAttrs := resourceLogs.Resource().Attributes()
	resourceAttrs.PutStr("db.system", "mssql")
	resourceAttrs.PutStr("server.address", s.config.Hostname)
	resourceAttrs.PutStr("server.port", s.config.Port)

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver")

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, analysis := range executionPlans {
		if analysis == nil {
			continue
		}

		// Create log record for execution plan summary
		s.createExecutionPlanSummaryLog(analysis, scopeLogs, now)

		// Create log records for each execution plan node/operator
		for _, node := range analysis.Nodes {
			s.createExecutionPlanNodeLog(analysis, &node, scopeLogs, now)
		}
	}
}

// createExecutionPlanSummaryLog creates a log record for execution plan summary
func (s *sqlServerScraper) createExecutionPlanSummaryLog(analysis *models.ExecutionPlanAnalysis, scopeLogs plog.ScopeLogs, timestamp pcommon.Timestamp) {
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.SetTimestamp(timestamp)
	logRecord.SetObservedTimestamp(timestamp)
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
	logRecord.SetSeverityText("INFO")

	// Set event name - this is the key for proper log event emission
	logRecord.SetEventName("sqlserver.execution_plan")

	// Set log body
	logRecord.Body().SetStr("SQL Server Execution Plan Summary")

	// Set attributes
	attrs := logRecord.Attributes()
	attrs.PutStr("query_id", analysis.QueryID)
	attrs.PutStr("plan_handle", analysis.PlanHandle)
	attrs.PutStr("sql_text", helpers.AnonymizeQueryText(analysis.SQLText))
	attrs.PutDouble("total_cost", analysis.TotalCost)
	attrs.PutStr("compile_time", analysis.CompileTime)
	attrs.PutInt("compile_cpu", analysis.CompileCPU)
	attrs.PutInt("compile_memory", analysis.CompileMemory)
	attrs.PutStr("collection_time", analysis.CollectionTime)
	attrs.PutInt("total_operators", int64(len(analysis.Nodes)))
}

// createExecutionPlanNodeLog creates a log record for an execution plan node/operator
func (s *sqlServerScraper) createExecutionPlanNodeLog(analysis *models.ExecutionPlanAnalysis, node *models.ExecutionPlanNode, scopeLogs plog.ScopeLogs, timestamp pcommon.Timestamp) {
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.SetTimestamp(timestamp)
	logRecord.SetObservedTimestamp(timestamp)
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
	logRecord.SetSeverityText("INFO")

	// Set event name - this is the key for proper log event emission
	logRecord.SetEventName("sqlserver.execution_plan_operator")

	// Set log body
	logRecord.Body().SetStr(fmt.Sprintf("SQL Server Execution Plan Operator: %s", node.PhysicalOp))

	// Set attributes
	attrs := logRecord.Attributes()
	attrs.PutStr("query_id", node.QueryID)
	attrs.PutStr("plan_handle", node.PlanHandle)
	attrs.PutInt("node_id", int64(node.NodeID))
	attrs.PutInt("parent_node_id", int64(node.ParentNodeID))
	attrs.PutStr("sql_text", helpers.AnonymizeQueryText(node.SQLText))
	attrs.PutStr("physical_op", node.PhysicalOp)
	attrs.PutStr("logical_op", node.LogicalOp)
	attrs.PutStr("input_type", node.InputType)
	attrs.PutDouble("estimate_rows", node.EstimateRows)
	attrs.PutDouble("estimate_io", node.EstimateIO)
	attrs.PutDouble("estimate_cpu", node.EstimateCPU)
	attrs.PutDouble("avg_row_size", node.AvgRowSize)
	attrs.PutDouble("total_subtree_cost", node.TotalSubtreeCost)
	attrs.PutDouble("estimated_operator_cost", node.EstimatedOperatorCost)
	attrs.PutStr("estimated_execution_mode", node.EstimatedExecutionMode)
	attrs.PutInt("granted_memory_kb", node.GrantedMemoryKb)
	attrs.PutBool("spill_occurred", node.SpillOccurred)
	attrs.PutBool("no_join_predicate", node.NoJoinPredicate)
	attrs.PutDouble("total_worker_time", node.TotalWorkerTime)
	attrs.PutDouble("total_elapsed_time", node.TotalElapsedTime)
	attrs.PutInt("total_logical_reads", node.TotalLogicalReads)
	attrs.PutInt("total_logical_writes", node.TotalLogicalWrites)
	attrs.PutInt("execution_count", int64(node.ExecutionCount))
	attrs.PutDouble("avg_elapsed_time_ms", node.AvgElapsedTimeMs)
	attrs.PutStr("collection_timestamp", node.CollectionTimestamp)
	attrs.PutStr("last_execution_time", node.LastExecutionTime)
}

// detectEngineEdition detects the SQL Server engine edition following nri-mssql pattern
// detectEngineEdition detects the SQL Server engine edition
func (s *sqlServerScraper) detectEngineEdition(ctx context.Context) (int, error) {
	queryFunc := func(query string) (int, error) {
		var results []struct {
			EngineEdition int `db:"EngineEdition"`
		}

		err := s.connection.Query(ctx, &results, query)
		if err != nil {
			return 0, err
		}

		if len(results) == 0 {
			s.logger.Debug("EngineEdition query returned empty output.")
			return 0, nil
		}

		s.logger.Debug("Detected EngineEdition", zap.Int("engine_edition", results[0].EngineEdition))
		return results[0].EngineEdition, nil
	}

	return queries.DetectEngineEdition(queryFunc)
}

// scrape collects SQL Server instance metrics using structured approach
func (s *sqlServerScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	s.logger.Debug("Starting SQL Server metrics collection",
		zap.String("hostname", s.config.Hostname),
		zap.String("port", s.config.Port))

	// Track scraping errors but continue with partial results
	var scrapeErrors []error

	// Check if connection is still valid before scraping
	if s.connection != nil {
		if err := s.connection.Ping(ctx); err != nil {
			s.logger.Error("Connection health check failed before scraping", zap.Error(err))
			scrapeErrors = append(scrapeErrors, fmt.Errorf("connection health check failed: %w", err))
			// Continue with scraping attempt - connection might recover
		}
	} else {
		s.logger.Error("No database connection available for scraping")
		return s.buildMetrics(ctx), fmt.Errorf("no database connection available")
	}

	// Refresh metadata cache if enabled and needed (respects refresh interval)
	if s.config.EnableWaitResourceEnrichment && s.metadataCache != nil {
		if err := s.metadataCache.Refresh(ctx); err != nil {
			s.logger.Warn("Failed to refresh metadata cache",
				zap.Error(err))
			// Continue scraping - stale cache is better than no data
		}
	}

	// Scrape database-level buffer pool metrics (bufferpool.sizePerDatabaseInBytes)
	scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseBufferMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database buffer metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database buffer metrics")
	}

	// Scrape database-level IO metrics (io.stallInMilliseconds)
	s.logger.Debug("Starting database IO metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseIOMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database IO metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database IO metrics")
	}

	// Scrape database-level log growth metrics (log.transactionGrowth)
	s.logger.Debug("Starting database log growth metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseLogGrowthMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database log growth metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database log growth metrics")
	}

	// Scrape database-level page file metrics (pageFileAvailable)
	s.logger.Debug("Starting database page file metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabasePageFileMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database page file metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database page file metrics")
	}

	// Scrape database-level page file total metrics (pageFileTotal)
	s.logger.Debug("Starting database page file total metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabasePageFileTotalMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database page file total metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database page file total metrics")
	}

	// Scrape instance-level memory metrics (memoryTotal, memoryAvailable, memoryUtilization)
	s.logger.Debug("Starting database memory metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseMemoryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database memory metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database memory metrics")
	}

	// Scrape database size metrics (total size and data size in MB)
	s.logger.Debug("Starting database size metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseSizeMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database size metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database size metrics")
	}

	// Scrape database disk metrics (max disk size for Azure SQL Database)
	s.logger.Debug("Starting database disk metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseDiskMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database disk metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database disk metrics")
	}

	// Scrape database transaction log metrics (flushes, bytes flushed, flush waits, active transactions)
	s.logger.Debug("Starting database transaction log metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseTransactionLogMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database transaction log metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database transaction log metrics")
	}

	// Scrape database log space usage metrics (used log space in MB)
	s.logger.Debug("Starting database log space usage metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseLogSpaceUsageMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database log space usage metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database log space usage metrics")
	}

	// // Scrape blocking session metrics if query monitoring is enabled
	// if s.config.EnableQueryMonitoring {
	// 	scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	// 	defer cancel()

	// 	// Use config values for blocking session parameters
	// 	limit := s.config.QueryMonitoringCountThreshold
	// 	textTruncateLimit := s.config.QueryMonitoringTextTruncateLimit // Use config value

	// 	if err := s.queryPerformanceScraper.ScrapeBlockingSessionMetrics(scrapeCtx, scopeMetrics, limit, textTruncateLimit); err != nil {
	// 		s.logger.Warn("Failed to scrape blocking session metrics - continuing with other metrics",
	// 			zap.Error(err),
	// 			zap.Duration("timeout", s.config.Timeout),
	// 			zap.Int("limit", limit),
	// 			zap.Int("text_truncate_limit", textTruncateLimit))
	// 		// Don't add to scrapeErrors - just warn and continue
	// 	} else {
	// 		s.logger.Debug("Successfully scraped blocking session metrics",
	// 			zap.Int("limit", limit),
	// 			zap.Int("text_truncate_limit", textTruncateLimit))
	// 	}
	// }

	// Scrape slow query metrics if query monitoring is enabled
	// Store query IDs for correlation with active queries
	var slowQueryIDs []string
	if s.config.EnableQueryMonitoring {
		scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		// Use config values for slow query parameters
		intervalSeconds := s.config.QueryMonitoringFetchInterval
		topN := s.config.QueryMonitoringCountThreshold
		elapsedTimeThreshold := s.config.QueryMonitoringResponseTimeThreshold
		textTruncateLimit := s.config.QueryMonitoringTextTruncateLimit

		s.logger.Info("Attempting to scrape slow query metrics",
			zap.Int("interval_seconds", intervalSeconds),
			zap.Int("top_n", topN),
			zap.Int("elapsed_time_threshold", elapsedTimeThreshold))

		slowQueries, err := s.queryPerformanceScraper.ScrapeSlowQueryMetrics(scrapeCtx, intervalSeconds, topN, elapsedTimeThreshold, textTruncateLimit, true)
		if err != nil {
			s.logger.Warn("Failed to scrape slow query metrics - continuing with other metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout),
				zap.Int("interval_seconds", intervalSeconds),
				zap.Int("top_n", topN),
				zap.Int("elapsed_time_threshold", elapsedTimeThreshold),
				zap.Int("text_truncate_limit", textTruncateLimit))
			// Don't add to scrapeErrors - just warn and continue
		} else {
			s.logger.Info("Successfully scraped slow query metrics",
				zap.Int("interval_seconds", intervalSeconds),
				zap.Int("top_n", topN),
				zap.Int("elapsed_time_threshold", elapsedTimeThreshold),
				zap.Int("text_truncate_limit", textTruncateLimit),
				zap.Int("slow_query_count", len(slowQueries)))

			// Extract query IDs for active query correlation
			slowQueryIDs = s.queryPerformanceScraper.ExtractQueryIDsFromSlowQueries(slowQueries)
			s.logger.Info("Extracted query IDs from slow queries for active query filtering",
				zap.Int("unique_query_id_count", len(slowQueryIDs)))

		}
	} else {
		s.logger.Info("Slow query scraping SKIPPED - EnableQueryMonitoring is false")
	}

	// Scrape active running queries metrics if enabled
	if s.config.EnableActiveRunningQueries {
		scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		// Use config values for active running queries parameters
		limit := s.config.QueryMonitoringCountThreshold // Reuse count threshold for active queries limit
		textTruncateLimit := s.config.QueryMonitoringTextTruncateLimit
		elapsedTimeThreshold := s.config.ActiveRunningQueriesElapsedTimeThreshold // Minimum elapsed time in ms

		s.logger.Info("Attempting to scrape active running queries metrics",
			zap.Int("limit", limit),
			zap.Int("text_truncate_limit", textTruncateLimit),
			zap.Int("elapsed_time_threshold_ms", elapsedTimeThreshold))

		// Step 1: Fetch active queries from database
		activeQueries, err := s.queryPerformanceScraper.ScrapeActiveRunningQueriesMetrics(scrapeCtx, limit, textTruncateLimit, elapsedTimeThreshold, slowQueryIDs)
		if err != nil {
			s.logger.Warn("Failed to fetch active running queries - continuing with other metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			// Don't add to scrapeErrors - just warn and continue
		} else if len(activeQueries) == 0 {
			s.logger.Info("No active queries found matching slow query IDs")
		} else {
			s.logger.Info("Active queries fetched, emitting metrics",
				zap.Int("active_query_count", len(activeQueries)))

			// Step 2: Emit metrics for active queries
			if err := s.queryPerformanceScraper.EmitActiveRunningQueriesMetrics(scrapeCtx, activeQueries); err != nil {
				s.logger.Warn("Failed to emit active running queries metrics",
					zap.Error(err))
			} else {
				s.logger.Info("Successfully emitted active running queries metrics",
					zap.Int("active_query_count", len(activeQueries)))
			}
		}
	} else {
		s.logger.Info("Active running queries scraping SKIPPED - EnableActiveRunningQueries is false")
	}
	s.logger.Debug("Starting instance buffer pool hit percent metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceComprehensiveStats(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance comprehensive statistics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Instance comprehensive statistics collection is disabled")
	}

	// Scrape instance-level memory metrics
	s.logger.Debug("Starting instance memory metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceMemoryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance memory metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance memory metrics")
	}

	// Scrape instance-level process counts metrics
	s.logger.Debug("Starting instance process counts metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceProcessCounts(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance process counts metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance process counts metrics")
	}

	// Scrape instance-level runnable tasks metrics
	s.logger.Debug("Starting instance runnable tasks metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceRunnableTasks(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance runnable tasks metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance runnable tasks metrics")
	}

	// Scrape instance-level active connections metrics
	s.logger.Debug("Starting instance active connections metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceActiveConnections(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance active connections metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance active connections metrics")
	}

	// Scrape instance-level buffer pool hit percent metrics
	s.logger.Debug("Starting instance buffer pool hit percent metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceBufferPoolHitPercent(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance buffer pool hit percent metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance buffer pool hit percent metrics")
	}

	// Scrape instance-level disk metrics
	s.logger.Debug("Starting instance disk metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceDiskMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance disk metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance disk metrics")
	}

	// Scrape instance-level buffer pool size metrics
	s.logger.Debug("Starting instance buffer pool size metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceBufferPoolSize(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance buffer pool size metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance buffer pool size metrics")
	}

	// Scrape instance-level comprehensive statistics
	s.logger.Debug("Starting instance comprehensive statistics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceComprehensiveStats(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance comprehensive statistics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance comprehensive statistics")
	}

	// Scrape enhanced instance metrics (new performance monitoring capabilities)
	s.logger.Debug("Starting enhanced instance metrics scraping")

	// Scrape instance target memory metrics
	s.logger.Debug("Starting instance target memory metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceTargetMemoryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance target memory metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance target memory metrics")
	}

	// Scrape instance performance ratios metrics
	s.logger.Debug("Starting instance performance ratios metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstancePerformanceRatiosMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance performance ratios metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance performance ratios metrics")
	}

	// Scrape instance index metrics
	s.logger.Debug("Starting instance index metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceIndexMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance index metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance index metrics")
	}

	// Scrape instance lock metrics
	s.logger.Debug("Starting instance lock metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.instanceScraper.ScrapeInstanceLockMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape instance lock metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped instance lock metrics")
	}

	// Scrape user connection metrics

	// Scrape user connection status metrics
	s.logger.Debug("Starting user connection status metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeUserConnectionStatusMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape user connection status metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped user connection status metrics")
	}

	// Scrape user connection summary metrics
	s.logger.Debug("Starting user connection summary metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeUserConnectionSummaryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape user connection summary metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped user connection summary metrics")
	}

	// Scrape user connection utilization metrics
	s.logger.Debug("Starting user connection utilization metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeUserConnectionUtilizationMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape user connection utilization metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped user connection utilization metrics")
	}

	// Scrape user connection by client metrics
	s.logger.Debug("Starting user connection by client metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeUserConnectionByClientMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape user connection by client metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped user connection by client metrics")
	}

	// Scrape user connection client summary metrics
	s.logger.Debug("Starting user connection client summary metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeUserConnectionClientSummaryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape user connection client summary metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped user connection client summary metrics")
	}

	// Scrape user connection stats metrics
	s.logger.Debug("Starting user connection stats metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeUserConnectionStatsMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape user connection stats metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped user connection stats metrics")
	}

	// Scrape authentication metrics

	// Scrape login/logout rate metrics
	s.logger.Debug("Starting login/logout rate metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeLoginLogoutMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape login/logout rate metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped login/logout rate metrics")
	}

	// Scrape login/logout summary metrics
	s.logger.Debug("Starting login/logout summary metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeLoginLogoutSummaryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape login/logout summary metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped login/logout summary metrics")
	}

	// Scrape failed login metrics
	s.logger.Debug("Starting failed login metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeFailedLoginMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape failed login metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped failed login metrics")
	}

	// Scrape failed login summary metrics
	s.logger.Debug("Starting failed login summary metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	if err := s.userConnectionScraper.ScrapeFailedLoginSummaryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape failed login summary metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped failed login summary metrics")
	}

	// Scrape failover cluster metrics

	// Scrape failover cluster replica metrics
	s.logger.Debug("Starting failover cluster replica metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.failoverClusterScraper.ScrapeFailoverClusterMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape failover cluster replica metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped failover cluster replica metrics")
	}

	// Scrape failover cluster replica state metrics
	s.logger.Debug("Starting failover cluster replica state metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.failoverClusterScraper.ScrapeFailoverClusterReplicaStateMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape failover cluster replica state metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped failover cluster replica state metrics")
	}

	// Scrape availability group health metrics
	s.logger.Debug("Starting availability group health metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.failoverClusterScraper.ScrapeFailoverClusterAvailabilityGroupHealthMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape availability group health metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped availability group health metrics")
	}

	// Scrape availability group configuration metrics
	s.logger.Debug("Starting availability group configuration metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.failoverClusterScraper.ScrapeFailoverClusterAvailabilityGroupMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape availability group configuration metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped availability group configuration metrics")
	}

	// Scrape failover cluster redo queue metrics (Azure SQL Managed Instance only)
	s.logger.Debug("Starting failover cluster redo queue metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.failoverClusterScraper.ScrapeFailoverClusterRedoQueueMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape failover cluster redo queue metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped failover cluster redo queue metrics")
	}

	// Scrape database principals metrics

	// Scrape database principals details metrics
	s.logger.Debug("Starting database principals details metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databasePrincipalsScraper.ScrapeDatabasePrincipalsMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database principals details metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database principals details metrics")
	}

	// Scrape database principals summary metrics
	s.logger.Debug("Starting database principals summary metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databasePrincipalsScraper.ScrapeDatabasePrincipalsSummaryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database principals summary metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database principals summary metrics")
	}

	// Scrape database principals activity metrics
	s.logger.Debug("Starting database principals activity metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databasePrincipalsScraper.ScrapeDatabasePrincipalActivityMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database principals activity metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database principals activity metrics")
	}

	// Scrape database role membership metrics

	// Scrape database role membership details metrics
	s.logger.Debug("Starting database role membership details metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databaseRoleMembershipScraper.ScrapeDatabaseRoleMembershipMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database role membership details metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database role membership details metrics")
	}

	// Scrape database role membership summary metrics
	s.logger.Debug("Starting database role membership summary metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databaseRoleMembershipScraper.ScrapeDatabaseRoleMembershipSummaryMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database role membership summary metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database role membership summary metrics")
	}

	// Scrape database role hierarchy metrics
	s.logger.Debug("Starting database role hierarchy metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databaseRoleMembershipScraper.ScrapeDatabaseRoleHierarchyMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database role hierarchy metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database role hierarchy metrics")
	}

	// Scrape database role activity metrics
	s.logger.Debug("Starting database role activity metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databaseRoleMembershipScraper.ScrapeDatabaseRoleActivityMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database role activity metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database role activity metrics")
	}

	// Scrape database role permission matrix metrics
	s.logger.Debug("Starting database role permission matrix metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.databaseRoleMembershipScraper.ScrapeDatabaseRolePermissionMatrixMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database role permission matrix metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database role permission matrix metrics")
	}

	// Scrape wait time metrics
	s.logger.Debug("Starting wait time metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.waitTimeScraper.ScrapeWaitTimeMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape wait time metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped wait time metrics")
	}

	// Scrape latch wait time metrics
	s.logger.Debug("Starting latch wait time metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.waitTimeScraper.ScrapeLatchWaitTimeMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape latch wait time metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped latch wait time metrics")
	}

	// Scrape security metrics
	s.logger.Debug("Starting security metrics scraping")

	// Scrape security principals metrics
	s.logger.Debug("Starting security principals metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.securityScraper.ScrapeSecurityPrincipalsMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape security principals metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped security principals metrics")
	}

	// Scrape security role members metrics
	s.logger.Debug("Starting security role members metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.securityScraper.ScrapeSecurityRoleMembersMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape security role members metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped security role members metrics")
	}

	// Scrape lock resource metrics
	s.logger.Debug("Starting lock resource metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.lockScraper.ScrapeLockResourceMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape lock resource metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped lock resource metrics")
	}

	// Scrape lock mode metrics
	s.logger.Debug("Starting lock mode metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.lockScraper.ScrapeLockModeMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape lock mode metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped lock mode metrics")
	}

	// Scrape thread pool health metrics
	s.logger.Debug("Starting thread pool health metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.threadPoolHealthScraper.ScrapeThreadPoolHealthMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape thread pool health metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped thread pool health metrics")
	}

	// Scrape TempDB contention metrics
	s.logger.Debug("Starting TempDB contention metrics scraping")
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	if err := s.tempdbContentionScraper.ScrapeTempDBContentionMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape TempDB contention metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped TempDB contention metrics")
	}

	// Build final metrics using MetricsBuilder (Oracle pattern)
	metrics := s.buildMetrics(ctx)

	// Log summary of scraping results
	if len(scrapeErrors) > 0 {
		s.logger.Warn("Completed scraping with errors",
			zap.Int("error_count", len(scrapeErrors)),
			zap.Int("metrics_collected", metrics.MetricCount()))

		// Return the first error but with partial metrics
		return metrics, scrapeErrors[0]
	}

	s.logger.Debug("Successfully completed SQL Server metrics collection",
		zap.Int("metrics_collected", metrics.MetricCount()))

	return metrics, nil
}

// buildMetrics constructs the final metrics output with resource attributes (Oracle pattern)
func (s *sqlServerScraper) buildMetrics(ctx context.Context) pmetric.Metrics {
	// Emit metrics with default resource
	// Resource attributes will be added by the collector pipeline
	return s.mb.Emit()
}

// addSystemInformationAsResourceAttributes collects system/host information and adds it as resource attributes
// This ensures that all metrics sent by the scraper include comprehensive host context
func (s *sqlServerScraper) addSystemInformationAsResourceAttributes(ctx context.Context, attrs pcommon.Map) error {
	// Collect system information using the main scraper
	systemInfo, err := s.CollectSystemInformation(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect system information: %w", err)
	}

	// Add SQL Server instance information
	if systemInfo.ServerName != nil && *systemInfo.ServerName != "" {
		attrs.PutStr("sql.instance_name", *systemInfo.ServerName)
	}
	if systemInfo.ComputerName != nil && *systemInfo.ComputerName != "" {
		attrs.PutStr("host.name", *systemInfo.ComputerName)
	}
	if systemInfo.ServiceName != nil && *systemInfo.ServiceName != "" {
		attrs.PutStr("sql.service_name", *systemInfo.ServiceName)
	}

	// Add SQL Server edition and version information
	if systemInfo.Edition != nil && *systemInfo.Edition != "" {
		attrs.PutStr("sql.edition", *systemInfo.Edition)
	}
	if systemInfo.EngineEdition != nil {
		attrs.PutInt("sql.engine_edition", int64(*systemInfo.EngineEdition))
	}
	if systemInfo.ProductVersion != nil && *systemInfo.ProductVersion != "" {
		attrs.PutStr("sql.version", *systemInfo.ProductVersion)
	}
	if systemInfo.VersionDesc != nil && *systemInfo.VersionDesc != "" {
		attrs.PutStr("sql.version_description", *systemInfo.VersionDesc)
	}

	// Add hardware information
	// if systemInfo.CPUCount != nil {
	// 	attrs.PutInt("host.cpu.count", int64(*systemInfo.CPUCount))
	// }
	// if systemInfo.ServerMemoryKB != nil {
	// 	attrs.PutInt("host.memory.total_kb", *systemInfo.ServerMemoryKB)
	// }
	// if systemInfo.AvailableMemoryKB != nil {
	// 	attrs.PutInt("host.memory.available_kb", *systemInfo.AvailableMemoryKB)
	// }

	// // Add instance configuration
	// if systemInfo.IsClustered != nil {
	// 	attrs.PutBool("sql.is_clustered", *systemInfo.IsClustered)
	// }
	// if systemInfo.IsHadrEnabled != nil {
	// 	attrs.PutBool("sql.is_hadr_enabled", *systemInfo.IsHadrEnabled)
	// }
	// if systemInfo.Uptime != nil {
	// 	attrs.PutInt("sql.uptime_minutes", int64(*systemInfo.Uptime))
	// }
	// if systemInfo.ComputerUptime != nil {
	// 	attrs.PutInt("host.uptime_seconds", int64(*systemInfo.ComputerUptime))
	// }

	// Add network configuration
	if systemInfo.Port != nil && *systemInfo.Port != "" {
		attrs.PutStr("sql.port", *systemInfo.Port)
	}
	if systemInfo.PortType != nil && *systemInfo.PortType != "" {
		attrs.PutStr("sql.port_type", *systemInfo.PortType)
	}
	if systemInfo.ForceEncryption != nil {
		attrs.PutBool("sql.force_encryption", *systemInfo.ForceEncryption != 0)
	}

	s.logger.Debug("Successfully added system information as resource attributes",
		zap.String("host_name", getStringValueFromMap(systemInfo.ComputerName)),
		zap.String("sql_instance", getStringValueFromMap(systemInfo.ServerName)),
		zap.String("sql_edition", getStringValueFromMap(systemInfo.Edition)),
		zap.Int("cpu_count", getIntValueFromMap(systemInfo.CPUCount)),
		zap.Bool("is_clustered", getBoolValueFromMap(systemInfo.IsClustered)))

	return nil
}

// Helper functions to safely extract values from pointers for logging
func getStringValueFromMap(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntValueFromMap(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getInt64ValueFromMap(ptr *int64) int64 {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getBoolValueFromMap(ptr *bool) bool {
	if ptr != nil {
		return *ptr
	}
	return false
}

// CollectSystemInformation retrieves comprehensive system and host information
// This information should be included as resource attributes with all metrics
func (s *sqlServerScraper) CollectSystemInformation(ctx context.Context) (*models.SystemInformation, error) {
	s.logger.Debug("Collecting SQL Server system and host information")

	var results []models.SystemInformation
	if err := s.connection.Query(ctx, &results, queries.SystemInformationQuery); err != nil {
		s.logger.Error("Failed to execute system information query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(queries.SystemInformationQuery, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return nil, fmt.Errorf("failed to execute system information query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from system information query - SQL Server may not be ready")
		return nil, fmt.Errorf("no results returned from system information query")
	}

	if len(results) > 1 {
		s.logger.Warn("Multiple results returned from system information query",
			zap.Int("result_count", len(results)))
	}

	result := results[0]

	// Log collected system information for debugging
	s.logger.Info("Successfully collected system information",
		zap.String("server_name", getStringValueFromMap(result.ServerName)),
		zap.String("computer_name", getStringValueFromMap(result.ComputerName)),
		zap.String("edition", getStringValueFromMap(result.Edition)),
		zap.Int("engine_edition", getIntValueFromMap(result.EngineEdition)),
		zap.String("product_version", getStringValueFromMap(result.ProductVersion)),
		zap.Int("cpu_count", getIntValueFromMap(result.CPUCount)),
		zap.Int64("server_memory_kb", getInt64ValueFromMap(result.ServerMemoryKB)),
		zap.Bool("is_clustered", getBoolValueFromMap(result.IsClustered)),
		zap.Bool("is_hadr_enabled", getBoolValueFromMap(result.IsHadrEnabled)))

	return &result, nil
}
