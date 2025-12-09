// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/helpers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/queries"
)

// QueryPerformanceScraper handles SQL Server query performance monitoring metrics collection
type QueryPerformanceScraper struct {
	connection          SQLConnectionInterface
	logger              *zap.Logger
	mb                  *metadata.MetricsBuilder
	startTime           pcommon.Timestamp // Used by slow query metrics (not yet migrated)
	engineEdition       int
	executionPlanLogger *models.ExecutionPlanLogger
	logConsumer         plog.Logs                     // For emitting execution plan logs
	slowQuerySmoother   *SlowQuerySmoother            // EWMA-based smoothing algorithm for slow queries
	intervalCalculator  *SimplifiedIntervalCalculator // Simplified delta-based interval calculator
	metadataCache       *helpers.MetadataCache        // Metadata cache for wait resource enrichment
}

// NewQueryPerformanceScraper creates a new query performance scraper with smoothing and interval calculator configuration
func NewQueryPerformanceScraper(
	conn SQLConnectionInterface,
	logger *zap.Logger,
	mb *metadata.MetricsBuilder,
	engineEdition int,
	smoothingEnabled bool,
	smoothingFactor float64,
	decayThreshold int,
	maxAgeMinutes int,
	intervalCalcEnabled bool,
	intervalCalcCacheTTLMinutes int,
	metadataCache *helpers.MetadataCache,
) *QueryPerformanceScraper {
	var smoother *SlowQuerySmoother
	var intervalCalc *SimplifiedIntervalCalculator

	// Initialize EWMA-based smoother if enabled
	if smoothingEnabled {
		// Initialize smoother with configured parameters
		maxAge := time.Duration(maxAgeMinutes) * time.Minute
		smoother = NewSlowQuerySmoother(logger, smoothingFactor, decayThreshold, maxAge)
		logger.Info("Slow query EWMA smoothing enabled",
			zap.Float64("smoothing_factor", smoothingFactor),
			zap.Int("decay_threshold", decayThreshold),
			zap.Int("max_age_minutes", maxAgeMinutes))
	} else {
		logger.Info("Slow query EWMA smoothing disabled")
	}

	// Initialize simplified interval-based calculator if enabled
	if intervalCalcEnabled {
		cacheTTL := time.Duration(intervalCalcCacheTTLMinutes) * time.Minute
		intervalCalc = NewSimplifiedIntervalCalculator(logger, cacheTTL)
		logger.Info("Simplified interval-based delta calculator enabled",
			zap.Int("cache_ttl_minutes", intervalCalcCacheTTLMinutes))
	} else {
		logger.Info("Simplified interval-based delta calculator disabled")
	}

	return &QueryPerformanceScraper{
		connection:          conn,
		logger:              logger,
		mb:                  mb,
		startTime:           pcommon.NewTimestampFromTime(time.Now()),
		engineEdition:       engineEdition,
		executionPlanLogger: models.NewExecutionPlanLogger(),
		slowQuerySmoother:   smoother,
		intervalCalculator:  intervalCalc,
		metadataCache:       metadataCache,
	}
}

// SetMetricsBuilder sets the metrics builder for this scraper
// This is called before each scrape operation to provide the current metrics builder
func (s *QueryPerformanceScraper) SetMetricsBuilder(mb *metadata.MetricsBuilder) {
	s.mb = mb
}

// ScrapeSlowQueryMetrics collects slow query performance monitoring metrics with interval-based averaging and/or EWMA smoothing
// Returns the processed slow queries for downstream correlation (e.g., filtering active queries by slow query IDs)
// emitMetrics: if true, emit metrics to MetricsBuilder; if false, only fetch and process data without metric emission
func (s *QueryPerformanceScraper) ScrapeSlowQueryMetrics(ctx context.Context, intervalSeconds, topN, elapsedTimeThreshold, textTruncateLimit int, emitMetrics bool) ([]models.SlowQuery, error) {

	query := fmt.Sprintf(queries.SlowQuery, intervalSeconds, topN, elapsedTimeThreshold, textTruncateLimit)

	s.logger.Debug("Executing slow query metrics collection",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.Int("interval_seconds", intervalSeconds),
		zap.Int("top_n", topN),
		zap.Int("elapsed_time_threshold", elapsedTimeThreshold),
		zap.Bool("interval_calc_enabled", s.intervalCalculator != nil),
		zap.Bool("ewma_smoothing_enabled", s.slowQuerySmoother != nil))

	var rawResults []models.SlowQuery
	if err := s.connection.Query(ctx, &rawResults, query); err != nil {
		return nil, fmt.Errorf("failed to execute slow query metrics query: %w", err)
	}

	s.logger.Debug("Raw slow query metrics fetched", zap.Int("raw_result_count", len(rawResults)))

	// Apply simplified interval-based delta calculation if enabled
	var resultsWithIntervalMetrics []models.SlowQuery
	if s.intervalCalculator != nil {
		now := time.Now()

		// Pre-allocate slice capacity to avoid reallocations (performance optimization)
		// Note: Length is still 0, capacity reserves space for worst case (all queries pass filters)
		resultsWithIntervalMetrics = make([]models.SlowQuery, 0, len(rawResults))

		// Calculate interval metrics for each query
		for _, rawQuery := range rawResults {
			metrics := s.intervalCalculator.CalculateMetrics(&rawQuery, now)

			if metrics == nil {
				s.logger.Debug("Skipping query with nil metrics")
				continue
			}

			// Skip queries with no new executions
			if !metrics.HasNewExecutions {
				s.logger.Debug("Skipping query with no new executions",
					zap.String("query_id", rawQuery.QueryID.String()),
					zap.Float64("time_since_last_exec_sec", metrics.TimeSinceLastExecSec))
				continue
			}

			// Apply interval-based threshold filtering
			// First scrape: interval = historical (no baseline), filter still applies
			// Subsequent scrapes: interval = delta, filter on delta performance
			if metrics.IntervalAvgElapsedTimeMs < float64(elapsedTimeThreshold) {
				s.logger.Debug("Skipping query below interval threshold",
					zap.String("query_id", rawQuery.QueryID.String()),
					zap.Float64("interval_avg_ms", metrics.IntervalAvgElapsedTimeMs),
					zap.Float64("historical_avg_ms", metrics.HistoricalAvgElapsedTimeMs),
					zap.Int("threshold_ms", elapsedTimeThreshold),
					zap.Bool("is_first_scrape", metrics.IsFirstScrape))
				continue
			}

			// Store interval metrics in new fields (preserve historical values)
			// Historical values (AvgElapsedTimeMS, AvgCPUTimeMS, ExecutionCount) remain unchanged from DB
			// NOTE: Only elapsed time has delta calculation, CPU uses historical average
			rawQuery.IntervalAvgElapsedTimeMS = &metrics.IntervalAvgElapsedTimeMs
			rawQuery.IntervalExecutionCount = &metrics.IntervalExecutionCount

			resultsWithIntervalMetrics = append(resultsWithIntervalMetrics, rawQuery)
		}

		// Cleanup stale entries periodically (TTL-based only)
		s.intervalCalculator.CleanupStaleEntries(now)

		s.logger.Debug("Simplified interval-based delta calculation applied",
			zap.Int("raw_count", len(rawResults)),
			zap.Int("processed_count", len(resultsWithIntervalMetrics)))

		// Log calculator statistics
		stats := s.intervalCalculator.GetCacheStats()
		s.logger.Debug("Interval calculator statistics", zap.Any("stats", stats))

		// Use interval-calculated results as input for next step
		rawResults = resultsWithIntervalMetrics
	}

	// Apply Go-level sorting and top-N selection AFTER delta calculation and filtering
	// This ensures we get the slowest queries based on interval (delta) averages, not historical
	if s.intervalCalculator != nil && len(rawResults) > 0 {
		// Sort by interval average elapsed time (delta) - slowest first
		sort.Slice(rawResults, func(i, j int) bool {
			// Handle nil cases - queries without interval metrics go to the end
			if rawResults[i].IntervalAvgElapsedTimeMS == nil {
				return false
			}
			if rawResults[j].IntervalAvgElapsedTimeMS == nil {
				return true
			}
			// Sort descending (highest delta average first)
			return *rawResults[i].IntervalAvgElapsedTimeMS > *rawResults[j].IntervalAvgElapsedTimeMS
		})

		s.logger.Debug("Sorted queries by interval average elapsed time (delta)",
			zap.Int("total_count", len(rawResults)))

		// Take top N queries after sorting
		if len(rawResults) > topN {
			s.logger.Debug("Applying top N selection",
				zap.Int("before_count", len(rawResults)),
				zap.Int("top_n", topN))
			rawResults = rawResults[:topN]
		}
	}

	// Apply EWMA smoothing algorithm if enabled (can work on top of interval metrics or raw)
	var resultsToProcess []models.SlowQuery
	if s.slowQuerySmoother != nil {
		// Smoothing is enabled - apply the algorithm
		resultsToProcess = s.slowQuerySmoother.Smooth(rawResults)

		s.logger.Debug("EWMA smoothing applied",
			zap.Int("input_count", len(rawResults)),
			zap.Int("smoothed_count", len(resultsToProcess)))

		// Log smoother statistics
		stats := s.slowQuerySmoother.GetHistoryStats()
		s.logger.Debug("EWMA smoother statistics", zap.Any("stats", stats))
	} else {
		// Smoothing is disabled - use results from interval calculator or raw
		resultsToProcess = rawResults
		s.logger.Debug("EWMA smoothing disabled, using interval-calculated or raw results")
	}

	// Process results and emit metrics if requested
	if emitMetrics {
		for i, result := range resultsToProcess {
			if err := s.processSlowQueryMetrics(result, i); err != nil {
				s.logger.Error("Failed to process slow query metric", zap.Error(err), zap.Int("index", i))
			}
		}
		s.logger.Debug("Slow query metrics emitted", zap.Int("count", len(resultsToProcess)))
	} else {
		s.logger.Debug("Skipping metric emission (emitMetrics=false)", zap.Int("count", len(resultsToProcess)))
	}

	// Return processed results for downstream correlation (e.g., filtering active queries)
	return resultsToProcess, nil
}

func (s *QueryPerformanceScraper) ScrapeActiveQueryExecutionPlans(ctx context.Context, activeQueries []models.ActiveRunningQuery) error {
	if len(activeQueries) == 0 {
		s.logger.Debug("No active queries to fetch execution statistics for")
		return nil
	}

	timestamp := pcommon.NewTimestampFromTime(time.Now())
	totalStatsFound := 0

	// Collect unique plan_handles from active queries to avoid duplicate fetches
	uniquePlanHandles := make(map[string]models.ActiveRunningQuery)
	for _, activeQuery := range activeQueries {
		if activeQuery.PlanHandle == nil || activeQuery.PlanHandle.IsEmpty() {
			continue
		}
		planHandleStr := activeQuery.PlanHandle.String()
		// Keep first occurrence for correlation attributes
		if _, exists := uniquePlanHandles[planHandleStr]; !exists {
			uniquePlanHandles[planHandleStr] = activeQuery
		}
	}

	s.logger.Info("Fetching execution statistics for active running query plan_handles",
		zap.Int("active_query_count", len(activeQueries)),
		zap.Int("unique_plan_handle_count", len(uniquePlanHandles)))

	// Fetch execution statistics for each unique plan_handle
	for _, activeQuery := range uniquePlanHandles {
		// Extract correlation attributes for logging
		sessionID := int64(0)
		if activeQuery.CurrentSessionID != nil {
			sessionID = *activeQuery.CurrentSessionID
		}
		requestID := int64(0)
		if activeQuery.RequestID != nil {
			requestID = *activeQuery.RequestID
		}

		planHandleHex := activeQuery.PlanHandle.String()
		query := fmt.Sprintf(queries.ExecutionStatsForActivePlanHandleQuery, "'"+planHandleHex+"'")

		s.logger.Debug("Fetching execution statistics for active query plan_handle",
			zap.String("plan_handle", planHandleHex),
			zap.Int64("session_id", sessionID),
			zap.Int64("request_id", requestID))

		var planResults []models.PlanHandleResult
		if err := s.connection.Query(ctx, &planResults, query); err != nil {
			s.logger.Warn("Failed to fetch execution statistics for active query plan_handle - continuing with others",
				zap.Error(err),
				zap.String("plan_handle", planHandleHex),
				zap.Int64("session_id", sessionID))
			continue
		}

		if len(planResults) == 0 {
			s.logger.Debug("No execution statistics found for active query plan_handle",
				zap.String("plan_handle", planHandleHex),
				zap.Int64("session_id", sessionID))
			continue
		}

		// Should only be 1 result (query by plan_handle, not query_hash)
		if len(planResults) > 1 {
			s.logger.Warn("Expected 1 result for plan_handle query, got multiple - using first",
				zap.String("plan_handle", planHandleHex),
				zap.Int("result_count", len(planResults)))
		}

		// Emit metrics for this plan_handle WITH active query correlation
		planResult := planResults[0]
		s.emitActiveQueryPlanMetrics(planResult, activeQuery, timestamp)

		totalStatsFound++
		s.logger.Debug("Fetched and emitted execution statistics for active query plan_handle",
			zap.String("plan_handle", planHandleHex),
			zap.Int64("session_id", sessionID))
	}

	s.logger.Info("Successfully fetched execution statistics for active running query plan_handles",
		zap.Int("active_query_count", len(activeQueries)),
		zap.Int("unique_plan_handles", len(uniquePlanHandles)),
		zap.Int("stats_found", totalStatsFound))

	return nil
}

// emitActiveQueryPlanMetrics emits execution plan metrics for an active running query
// These metrics represent HISTORICAL statistics from dm_exec_query_stats (same as slow query plans)
// Uses namespace: sqlserver.plan.* (SAME as slow query plans)
// Context: Active query drill-down (WITH session_id/request_id/request_start_time for correlation)
// The key difference from slow query plans is the presence of session correlation attributes
func (s *QueryPerformanceScraper) emitActiveQueryPlanMetrics(planResult models.PlanHandleResult, activeQuery models.ActiveRunningQuery, timestamp pcommon.Timestamp) {
	// Helper functions to safely get attribute values
	getQueryID := func() string {
		if planResult.QueryID != nil {
			return planResult.QueryID.String()
		}
		return ""
	}

	getPlanHandle := func() string {
		if planResult.PlanHandle != nil {
			return planResult.PlanHandle.String()
		}
		return ""
	}

	getQueryPlanHash := func() string {
		if planResult.QueryPlanHash != nil {
			return planResult.QueryPlanHash.String()
		}
		return ""
	}

	getSessionID := func() int64 {
		if activeQuery.CurrentSessionID != nil {
			return *activeQuery.CurrentSessionID
		}
		return 0
	}

	getRequestID := func() int64 {
		if activeQuery.RequestID != nil {
			return *activeQuery.RequestID
		}
		return 0
	}

	getRequestStartTime := func() string {
		if activeQuery.RequestStartTime != nil {
			return *activeQuery.RequestStartTime
		}
		return ""
	}

	getLastExecutionTime := func() string {
		if planResult.LastExecutionTime != nil {
			return *planResult.LastExecutionTime
		}
		return ""
	}

	getCreationTime := func() string {
		if planResult.CreationTime != nil {
			return *planResult.CreationTime
		}
		return ""
	}

	getDatabaseName := func() string {
		if activeQuery.DatabaseName != nil {
			return *activeQuery.DatabaseName
		}
		return ""
	}

	getSchemaName := func() string {
		if activeQuery.SchemaName != nil {
			return *activeQuery.SchemaName
		}
		return ""
	}

	// Metric 1: Execution count
	if planResult.ExecutionCount != nil {
		s.mb.RecordSqlserverPlanExecutionCountDataPoint(
			timestamp,
			*planResult.ExecutionCount,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 2: Average elapsed time
	if planResult.AvgElapsedTimeMs != nil {
		s.mb.RecordSqlserverPlanAvgElapsedTimeMsDataPoint(
			timestamp,
			*planResult.AvgElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 3: Total elapsed time
	if planResult.TotalElapsedTimeMs != nil {
		s.mb.RecordSqlserverPlanTotalElapsedTimeMsDataPoint(
			timestamp,
			*planResult.TotalElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 4: Min/Max elapsed time
	if planResult.MinElapsedTimeMs != nil {
		s.mb.RecordSqlserverPlanMinElapsedTimeMsDataPoint(
			timestamp,
			*planResult.MinElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	if planResult.MaxElapsedTimeMs != nil {
		s.mb.RecordSqlserverPlanMaxElapsedTimeMsDataPoint(
			timestamp,
			*planResult.MaxElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 5: Average worker time (CPU)
	if planResult.AvgWorkerTimeMs != nil {
		s.mb.RecordSqlserverPlanAvgWorkerTimeMsDataPoint(
			timestamp,
			*planResult.AvgWorkerTimeMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 6: Average logical reads
	if planResult.AvgLogicalReads != nil {
		s.mb.RecordSqlserverPlanAvgLogicalReadsDataPoint(
			timestamp,
			*planResult.AvgLogicalReads,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 7: Average logical writes
	if planResult.AvgLogicalWrites != nil {
		s.mb.RecordSqlserverPlanAvgLogicalWritesDataPoint(
			timestamp,
			*planResult.AvgLogicalWrites,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 8: Last DOP (degree of parallelism)
	if planResult.LastDOP != nil {
		s.mb.RecordSqlserverPlanLastDopDataPoint(
			timestamp,
			int64(*planResult.LastDOP),
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 9: Last grant KB (memory grant)
	if planResult.LastGrantKB != nil {
		s.mb.RecordSqlserverPlanLastGrantKbDataPoint(
			timestamp,
			*planResult.LastGrantKB,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}

	// Metric 10: Last spills (tempdb spills)
	if planResult.LastSpills != nil {
		s.mb.RecordSqlserverPlanLastSpillsDataPoint(
			timestamp,
			int64(*planResult.LastSpills),
			getQueryID(),
			getPlanHandle(),
			getQueryPlanHash(),
			getSessionID(),
			getRequestID(),
			getRequestStartTime(),
			getLastExecutionTime(),
			getCreationTime(),
			getDatabaseName(),
			getSchemaName(),
		)
	}
}
func (s *QueryPerformanceScraper) processSlowQueryMetrics(result models.SlowQuery, index int) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Helper functions to safely get attribute values
	getQueryID := func() string {
		if result.QueryID != nil {
			return result.QueryID.String()
		}
		return ""
	}

	getPlanHandle := func() string {
		if result.PlanHandle != nil {
			return result.PlanHandle.String()
		}
		return ""
	}

	getDatabaseName := func() string {
		if result.DatabaseName != nil {
			return *result.DatabaseName
		}
		return ""
	}

	getSchemaName := func() string {
		if result.SchemaName != nil {
			return *result.SchemaName
		}
		return ""
	}

	getStatementType := func() string {
		if result.StatementType != nil {
			return *result.StatementType
		}
		return ""
	}

	getQueryText := func() string {
		if result.QueryText != nil {
			return *result.QueryText
		}
		return ""
	}

	getCollectionTimestamp := func() string {
		if result.CollectionTimestamp != nil {
			return *result.CollectionTimestamp
		}
		return ""
	}

	getLastExecutionTimestamp := func() string {
		if result.LastExecutionTimestamp != nil {
			return *result.LastExecutionTimestamp
		}
		return ""
	}

	getQuerySignature := func() string {
		if result.QueryText != nil {
			return helpers.ComputeQueryHash(*result.QueryText)
		}
		return ""
	}

	// Create detailed attributes for logging/debugging (not used in metrics)
	logAttributes := func() []zap.Field {
		var fields []zap.Field
		if result.QueryID != nil {
			fields = append(fields, zap.String("query_id", result.QueryID.String()))
		}
		if result.PlanHandle != nil {
			fields = append(fields, zap.String("plan_handle", result.PlanHandle.String()))
		}
		if result.DatabaseName != nil {
			fields = append(fields, zap.String("database_name", *result.DatabaseName))
		}
		if result.QueryText != nil {
			// Anonymize and truncate query text for logging
			anonymizedSQL := helpers.SafeAnonymizeQueryText(result.QueryText)
			if len(anonymizedSQL) > 100 {
				anonymizedSQL = anonymizedSQL[:100] + "..."
			}
			fields = append(fields, zap.String("query_text_preview", anonymizedSQL))
		}
		if result.CollectionTimestamp != nil {
			fields = append(fields, zap.String("collection_timestamp", *result.CollectionTimestamp))
		}
		if result.LastExecutionTimestamp != nil {
			fields = append(fields, zap.String("last_execution_timestamp", *result.LastExecutionTimestamp))
		}
		return fields
	}

	// Create avg_cpu_time_ms metric (historical/cumulative only, no delta) - CARDINALITY SAFE
	if result.AvgCPUTimeMS != nil {
		s.mb.RecordSqlserverSlowqueryHistoricalAvgCPUTimeMsDataPoint(
			timestamp,
			*result.AvgCPUTimeMS,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Create avg_disk_reads metric - CARDINALITY SAFE
	if result.AvgDiskReads != nil {
		s.mb.RecordSqlserverSlowqueryAvgDiskReadsDataPoint(
			timestamp,
			*result.AvgDiskReads,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Create avg_disk_writes metric - CARDINALITY SAFE
	if result.AvgDiskWrites != nil {
		s.mb.RecordSqlserverSlowqueryAvgDiskWritesDataPoint(
			timestamp,
			*result.AvgDiskWrites,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Create avg_rows_processed metric - CARDINALITY SAFE
	if result.AvgRowsProcessed != nil {
		s.mb.RecordSqlserverSlowqueryAvgRowsProcessedDataPoint(
			timestamp,
			*result.AvgRowsProcessed,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	} else {
		s.logger.Debug("AvgRowsProcessed is nil for slow query", zap.Int("index", index))
	}

	// Create avg_elapsed_time_ms metric (historical/cumulative) - CARDINALITY SAFE
	if result.AvgElapsedTimeMS != nil {
		s.mb.RecordSqlserverSlowqueryHistoricalAvgElapsedTimeMsDataPoint(
			timestamp,
			*result.AvgElapsedTimeMS,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
			getCollectionTimestamp(),
			getLastExecutionTimestamp(),
		)
	}

	// Create interval_avg_elapsed_time_ms metric (delta) - CARDINALITY SAFE
	if result.IntervalAvgElapsedTimeMS != nil {
		s.mb.RecordSqlserverSlowqueryIntervalAvgElapsedTimeMsDataPoint(
			timestamp,
			*result.IntervalAvgElapsedTimeMS,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Create execution_count metric (historical/cumulative) - CARDINALITY SAFE
	if result.ExecutionCount != nil {
		s.mb.RecordSqlserverSlowqueryHistoricalExecutionCountDataPoint(
			timestamp,
			*result.ExecutionCount,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Create interval_execution_count metric (delta) - CARDINALITY SAFE
	if result.IntervalExecutionCount != nil {
		s.mb.RecordSqlserverSlowqueryIntervalExecutionCountDataPoint(
			timestamp,
			*result.IntervalExecutionCount,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// NOTE: Removed separate query_id and plan_handle metrics as these attributes
	// are now included in ALL slow query metrics via createSafeAttributes()
	// This eliminates redundant metric emission

	// Create query_text metric with cardinality control
	if result.QueryText != nil {
		s.mb.RecordSqlserverSlowqueryQueryTextDataPoint(
			timestamp,
			1,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
			getQuerySignature(),
		)
	}

	// ========================================
	// RCA ENHANCEMENT METRICS
	// ========================================

	// Min/Max/Last Elapsed Time Metrics
	if result.MinElapsedTimeMs != nil {
		s.mb.RecordSqlserverSlowqueryMinElapsedTimeMsDataPoint(
			timestamp,
			*result.MinElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	if result.MaxElapsedTimeMs != nil {
		s.mb.RecordSqlserverSlowqueryMaxElapsedTimeMsDataPoint(
			timestamp,
			*result.MaxElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	if result.LastElapsedTimeMs != nil {
		s.mb.RecordSqlserverSlowqueryLastElapsedTimeMsDataPoint(
			timestamp,
			*result.LastElapsedTimeMs,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Memory Grant Metrics
	if result.LastGrantKB != nil {
		s.mb.RecordSqlserverSlowqueryLastGrantKbDataPoint(
			timestamp,
			*result.LastGrantKB,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	if result.LastUsedGrantKB != nil {
		s.mb.RecordSqlserverSlowqueryLastUsedGrantKbDataPoint(
			timestamp,
			*result.LastUsedGrantKB,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// TempDB Spill Metrics
	if result.LastSpills != nil {
		s.mb.RecordSqlserverSlowqueryLastSpillsDataPoint(
			timestamp,
			*result.LastSpills,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	if result.MaxSpills != nil {
		s.mb.RecordSqlserverSlowqueryMaxSpillsDataPoint(
			timestamp,
			*result.MaxSpills,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Parallelism Metrics
	if result.LastDOP != nil {
		s.mb.RecordSqlserverSlowqueryLastDopDataPoint(
			timestamp,
			*result.LastDOP,
			getQueryID(),
			getPlanHandle(),
			getDatabaseName(),
			getSchemaName(),
			getStatementType(),
			getQueryText(),
		)
	}

	// Use dedicated logging function with cardinality-safe approach
	s.logger.Debug("Processed slow query metrics with cardinality safety", logAttributes()...)

	return nil
}

// processQueryExecutionPlanMetrics processes query execution plan metrics with cardinality safety
func (s *QueryPerformanceScraper) processQueryExecutionPlanMetrics(result models.QueryExecutionPlan, index int) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Helper functions to safely get attribute values
	getQueryID := func() string {
		if result.QueryID != nil {
			return result.QueryID.String()
		}
		return ""
	}

	getPlanHandle := func() string {
		if result.PlanHandle != nil {
			return result.PlanHandle.String()
		}
		return ""
	}

	getQueryPlanID := func() string {
		if result.QueryPlanID != nil {
			return result.QueryPlanID.String()
		}
		return ""
	}

	getQueryText := func() string {
		if result.SQLText != nil {
			return helpers.AnonymizeQueryText(*result.SQLText)
		}
		return ""
	}

	getCreationTime := func() string {
		if result.CreationTime != nil {
			return *result.CreationTime
		}
		return ""
	}

	getLastExecutionTime := func() string {
		if result.LastExecutionTime != nil {
			return *result.LastExecutionTime
		}
		return ""
	}

	// Create detailed attributes for logging/debugging (not used in metrics)
	logAttributes := func() []zap.Field {
		var fields []zap.Field
		if result.QueryID != nil {
			fields = append(fields, zap.String("query_id", result.QueryID.String()))
		}
		if result.PlanHandle != nil {
			fields = append(fields, zap.String("plan_handle", result.PlanHandle.String()))
		}
		if result.QueryPlanID != nil {
			fields = append(fields, zap.String("query_plan_id", result.QueryPlanID.String()))
		}
		if result.SQLText != nil {
			// Anonymize and truncate SQL text for logging
			anonymizedSQL := helpers.AnonymizeQueryText(*result.SQLText)
			if len(anonymizedSQL) > 100 {
				anonymizedSQL = anonymizedSQL[:100] + "..."
			}
			fields = append(fields, zap.String("sql_text_preview", anonymizedSQL))
		}
		return fields
	}

	// Create TotalCPUMs metric - CARDINALITY SAFE
	if result.TotalCPUMs != nil {
		s.mb.RecordSqlserverIndividualQueryTotalCPUMsDataPoint(
			timestamp,
			*result.TotalCPUMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanID(),
			getQueryText(),
			getCreationTime(),
			getLastExecutionTime(),
		)
	}

	// Create TotalElapsedMs metric - CARDINALITY SAFE
	// This metric includes all attributes: execution plan XML, timestamps, query details
	if result.TotalElapsedMs != nil {
		s.mb.RecordSqlserverIndividualQueryTotalElapsedMsDataPoint(
			timestamp,
			*result.TotalElapsedMs,
			getQueryID(),
			getPlanHandle(),
			getQueryPlanID(),
			getQueryText(),
			getCreationTime(),
			getLastExecutionTime(),
		)
	}

	// Log detailed information for debugging/analysis
	s.logger.Debug("Processed query execution plan metrics with execution plan XML and timestamps included",
		logAttributes()...)

	return nil
}

// getSlowQueryResults fetches slow query results to extract QueryIDs for execution plan analysis
func (s *QueryPerformanceScraper) getSlowQueryResults(ctx context.Context, intervalSeconds, topN, elapsedTimeThreshold,
	textTruncateLimit int,
) ([]models.SlowQuery, error) {
	// Format the slow query with parameters
	formattedQuery := fmt.Sprintf(queries.SlowQuery, intervalSeconds, topN, elapsedTimeThreshold, textTruncateLimit)

	s.logger.Debug("Executing slow query to extract QueryIDs for execution plan analysis",
		zap.String("query", queries.TruncateQuery(formattedQuery, 100)),
		zap.Int("interval_seconds", intervalSeconds),
		zap.Int("top_n", topN),
		zap.Int("elapsed_time_threshold", elapsedTimeThreshold))

	var results []models.SlowQuery
	if err := s.connection.Query(ctx, &results, formattedQuery); err != nil {
		return nil, fmt.Errorf("failed to execute slow query for QueryID extraction: %w", err)
	}

	s.logger.Debug("Successfully fetched slow queries for QueryID extraction",
		zap.Int("result_count", len(results)))

	return results, nil
}

// REMOVED: emitExecutionPlanLogs, categorizeOperator, and assessPerformanceImpact functions
// These functions created SqlServerExecutionPlan and SqlServerExecutionPlanNode custom events
// which have been removed in favor of SqlServerActiveQueryExecutionPlan with aggregated stats

// REMOVED (2025-11-28): logExecutionPlanToNewRelic function
// This function created structured logs for execution plan data
// which has been removed in favor of SqlServerActiveQueryExecutionPlan custom events with aggregated stats

// ExtractQueryIDsFromSlowQueries extracts unique QueryIDs from slow query results
func (s *QueryPerformanceScraper) ExtractQueryIDsFromSlowQueries(slowQueries []models.SlowQuery) []string {
	queryIDMap := make(map[string]bool)
	var queryIDs []string

	for _, slowQuery := range slowQueries {
		if slowQuery.QueryID != nil && !slowQuery.QueryID.IsEmpty() {
			queryIDStr := slowQuery.QueryID.String()
			if !queryIDMap[queryIDStr] {
				queryIDMap[queryIDStr] = true
				queryIDs = append(queryIDs, queryIDStr)
			}
		}
	}

	s.logger.Debug("Extracted unique QueryIDs from slow queries",
		zap.Int("total_slow_queries", len(slowQueries)),
		zap.Int("unique_query_ids", len(queryIDs)))

	return queryIDs
}

// formatQueryIDsForSQL converts QueryID slice to comma-separated string for SQL IN clause
// Follows nri-mssql pattern for QueryID formatting
func (s *QueryPerformanceScraper) formatQueryIDsForSQL(queryIDs []string) string {
	if len(queryIDs) == 0 {
		return "0x0" // Return placeholder if no QueryIDs
	}

	// Join QueryIDs with commas for SQL STRING_SPLIT
	// QueryIDs are already in hex format (0x...), so we can use them directly
	queryIDsString := ""
	for i, queryID := range queryIDs {
		if i > 0 {
			queryIDsString += ","
		}
		queryIDsString += queryID
	}

	s.logger.Debug("Formatted QueryIDs for SQL query",
		zap.Int("query_id_count", len(queryIDs)),
		zap.String("formatted_query_ids", queries.TruncateQuery(queryIDsString, 100)))

	return queryIDsString
}

// extractPlanHandlesFromSlowQueries extracts unique PlanHandles from slow query results
func (s *QueryPerformanceScraper) extractPlanHandlesFromSlowQueries(slowQueries []models.SlowQuery) []string {
	planHandleMap := make(map[string]bool)
	var planHandles []string

	for _, slowQuery := range slowQueries {
		if slowQuery.PlanHandle != nil && !slowQuery.PlanHandle.IsEmpty() {
			planHandleStr := slowQuery.PlanHandle.String()
			if !planHandleMap[planHandleStr] {
				planHandleMap[planHandleStr] = true
				planHandles = append(planHandles, planHandleStr)
			}
		}
	}

	s.logger.Debug("Extracted unique PlanHandles from slow queries",
		zap.Int("total_slow_queries", len(slowQueries)),
		zap.Int("unique_plan_handles", len(planHandles)))

	return planHandles
}

// formatPlanHandlesForSQL converts PlanHandle slice to comma-separated string for SQL IN clause
func (s *QueryPerformanceScraper) formatPlanHandlesForSQL(planHandles []string) string {
	if len(planHandles) == 0 {
		return "0x0" // Return placeholder if no PlanHandles
	}

	// Join PlanHandles with commas for SQL STRING_SPLIT
	// PlanHandles are already in hex format (0x...), so we can use them directly
	planHandlesString := ""
	for i, planHandle := range planHandles {
		if i > 0 {
			planHandlesString += ","
		}
		planHandlesString += planHandle
	}

	s.logger.Debug("Formatted PlanHandles for SQL query",
		zap.Int("plan_handle_count", len(planHandles)),
		zap.String("formatted_plan_handles", queries.TruncateQuery(planHandlesString, 100)))

	return planHandlesString
}
