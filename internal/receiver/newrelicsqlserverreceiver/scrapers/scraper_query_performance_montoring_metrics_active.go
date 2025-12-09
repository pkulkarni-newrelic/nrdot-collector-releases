// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/helpers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/queries"
)

// ScrapeActiveRunningQueriesMetrics fetches active running queries from SQL Server
// Returns the list of active queries for further processing (metrics emission and execution plan fetching)
func (s *QueryPerformanceScraper) ScrapeActiveRunningQueriesMetrics(ctx context.Context, limit, textTruncateLimit, elapsedTimeThreshold int, slowQueryIDs []string) ([]models.ActiveRunningQuery, error) {
	// Skip active query scraping if no slow queries found (nothing to correlate)
	if len(slowQueryIDs) == 0 {
		s.logger.Info("No slow queries found, skipping active query scraping (nothing to correlate)")
		return nil, nil
	}

	// Build query with slow query correlation filter
	queryIDFilter := "AND r_wait.query_hash IN (" + strings.Join(slowQueryIDs, ",") + ")"
	query := fmt.Sprintf(queries.ActiveRunningQueriesQuery, limit, textTruncateLimit, elapsedTimeThreshold, queryIDFilter)

	s.logger.Debug("Executing active running queries fetch",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.Int("limit", limit),
		zap.Int("text_truncate_limit", textTruncateLimit),
		zap.Int("elapsed_time_threshold_ms", elapsedTimeThreshold),
		zap.Int("slow_query_id_count", len(slowQueryIDs)))

	var results []models.ActiveRunningQuery
	if err := s.connection.Query(ctx, &results, query); err != nil {
		return nil, fmt.Errorf("failed to execute active running queries query: %w", err)
	}

	s.logger.Info("Active running queries fetched from database",
		zap.Int("result_count", len(results)))

	return results, nil
}

// EmitActiveRunningQueriesMetrics emits metrics for active running queries
// This processes the active queries and emits metrics (no execution plans)
func (s *QueryPerformanceScraper) EmitActiveRunningQueriesMetrics(ctx context.Context, activeQueries []models.ActiveRunningQuery) error {
	if len(activeQueries) == 0 {
		s.logger.Info("No active queries to emit metrics for")
		return nil
	}

	filteredCount := 0
	processedCount := 0

	for i, result := range activeQueries {
		// Defensive checks for required fields
		if result.WaitType == nil || *result.WaitType == "" {
			filteredCount++
			s.logger.Warn("Active query has NULL/empty wait_type, skipping metric emission",
				zap.Any("session_id", result.CurrentSessionID))
			continue
		}

		if result.PlanHandle == nil || result.PlanHandle.IsEmpty() {
			filteredCount++
			s.logger.Warn("Active query has NULL/empty plan_handle, skipping metric emission",
				zap.Any("session_id", result.CurrentSessionID))
			continue
		}

		if result.QueryID == nil || result.QueryID.IsEmpty() {
			filteredCount++
			s.logger.Warn("Active query has NULL/empty query_id, skipping metric emission",
				zap.Any("session_id", result.CurrentSessionID))
			continue
		}

		processedCount++

		// Emit metrics for this active query (no execution plan XML)
		if err := s.processActiveRunningQueryMetricsWithPlan(result, i, ""); err != nil {
			s.logger.Error("Failed to emit active running query metrics", zap.Error(err), zap.Int("index", i))
		}
	}

	s.logger.Info("Active running queries metrics emission complete",
		zap.Int("total_queries", len(activeQueries)),
		zap.Int("filtered_out", filteredCount),
		zap.Int("metrics_emitted", processedCount))

	return nil
}

// EmitActiveRunningExecutionPlansAsLogs fetches execution plans for active queries and emits as OTLP logs
// This is called from ScrapeLogs to emit execution plans with slow query correlation
func (s *QueryPerformanceScraper) EmitActiveRunningExecutionPlansAsLogs(ctx context.Context, logs plog.Logs, activeQueries []models.ActiveRunningQuery) error {
	if len(activeQueries) == 0 {
		s.logger.Info("No active queries to fetch execution plans for")
		return nil
	}

	fetchedCount := 0
	errorCount := 0

	for i, result := range activeQueries {
		// Skip if no plan handle
		if result.PlanHandle == nil || result.PlanHandle.IsEmpty() {
			continue
		}

		s.logger.Debug("Fetching execution plan XML for active query",
			zap.Any("session_id", result.CurrentSessionID),
			zap.String("plan_handle", result.PlanHandle.String()))

		// Fetch execution plan XML
		executionPlanXML, err := s.fetchExecutionPlanXML(ctx, result.PlanHandle)
		if err != nil {
			errorCount++
			s.logger.Warn("Failed to fetch execution plan XML for active query",
				zap.Error(err),
				zap.Any("session_id", result.CurrentSessionID))
			continue
		}

		if executionPlanXML == "" {
			s.logger.Debug("Empty execution plan XML returned",
				zap.Any("session_id", result.CurrentSessionID))
			continue
		}

		s.logger.Debug("Successfully fetched execution plan XML",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int("xml_length", len(executionPlanXML)))

		// Parse and emit execution plan as logs
		if err := s.parseAndEmitExecutionPlanAsLogs(ctx, result, executionPlanXML, logs); err != nil {
			errorCount++
			s.logger.Error("Failed to parse and emit execution plan as logs",
				zap.Error(err),
				zap.Any("session_id", result.CurrentSessionID),
				zap.Int("index", i))
			continue
		}

		fetchedCount++
	}

	s.logger.Info("Execution plans emission complete",
		zap.Int("total_queries", len(activeQueries)),
		zap.Int("plans_fetched", fetchedCount),
		zap.Int("errors", errorCount))

	return nil
}

func (s *QueryPerformanceScraper) processActiveRunningQueryMetricsWithPlan(result models.ActiveRunningQuery, index int, executionPlanXML string) error {
	if result.CurrentSessionID == nil {
		s.logger.Debug("Skipping active running query with nil session ID", zap.Int("index", index))
		return nil
	}

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Helper functions for attribute extraction
	getSessionID := func() int64 {
		if result.CurrentSessionID != nil {
			return *result.CurrentSessionID
		}
		return 0
	}
	getRequestID := func() int64 {
		if result.RequestID != nil {
			return *result.RequestID
		}
		return 0
	}
	getDatabaseName := func() string {
		if result.DatabaseName != nil {
			return *result.DatabaseName
		}
		return ""
	}
	getLoginName := func() string {
		if result.LoginName != nil {
			return *result.LoginName
		}
		return ""
	}
	getHostName := func() string {
		if result.HostName != nil {
			return *result.HostName
		}
		return ""
	}
	getProgramName := func() string {
		if result.ProgramName != nil {
			return *result.ProgramName
		}
		return ""
	}
	getRequestCommand := func() string {
		if result.RequestCommand != nil {
			return *result.RequestCommand
		}
		return ""
	}
	getRequestStatus := func() string {
		if result.RequestStatus != nil {
			return *result.RequestStatus
		}
		return ""
	}
	getSessionStatus := func() string {
		if result.SessionStatus != nil {
			return *result.SessionStatus
		}
		return ""
	}
	getClientInterfaceName := func() string {
		if result.ClientInterfaceName != nil {
			return *result.ClientInterfaceName
		}
		return ""
	}
	getWaitType := func() string {
		if result.WaitType != nil {
			return *result.WaitType
		}
		return ""
	}
	getWaitResource := func() string {
		if result.WaitResource != nil {
			return *result.WaitResource
		}
		return ""
	}
	getWaitResourceObjectName := func() string {
		if result.WaitResourceObjectName != nil {
			return *result.WaitResourceObjectName
		}
		return ""
	}
	getWaitResourceDatabaseName := func() string {
		if result.WaitResourceDatabaseName != nil {
			return *result.WaitResourceDatabaseName
		}
		return ""
	}
	getLastWaitType := func() string {
		if result.LastWaitType != nil {
			return *result.LastWaitType
		}
		return ""
	}
	getRequestStartTime := func() string {
		if result.RequestStartTime != nil {
			return *result.RequestStartTime
		}
		return ""
	}
	getCollectionTimestamp := func() string {
		if result.CollectionTimestamp != nil {
			return *result.CollectionTimestamp
		}
		return ""
	}
	getTransactionID := func() int64 {
		if result.TransactionID != nil {
			return *result.TransactionID
		}
		return 0
	}
	getOpenTransactionCount := func() int64 {
		if result.OpenTransactionCount != nil {
			return *result.OpenTransactionCount
		}
		return 0
	}
	getTransactionIsolationLevel := func() int64 {
		if result.TransactionIsolationLevel != nil {
			return *result.TransactionIsolationLevel
		}
		return 0
	}
	getDegreeOfParallelism := func() int64 {
		if result.DegreeOfParallelism != nil {
			return *result.DegreeOfParallelism
		}
		return 0
	}
	getParallelWorkerCount := func() int64 {
		if result.ParallelWorkerCount != nil {
			return *result.ParallelWorkerCount
		}
		return 0
	}
	getBlockingSessionID := func() int64 {
		if result.BlockingSessionID != nil {
			return *result.BlockingSessionID
		}
		return 0
	}
	getBlockingLoginName := func() string {
		if result.BlockerLoginName != nil {
			return *result.BlockerLoginName
		}
		return ""
	}
	getBlockingHostName := func() string {
		if result.BlockerHostName != nil {
			return *result.BlockerHostName
		}
		return ""
	}
	getBlockingProgramName := func() string {
		if result.BlockerProgramName != nil {
			return *result.BlockerProgramName
		}
		return ""
	}
	getBlockingStatus := func() string {
		if result.BlockerStatus != nil {
			return *result.BlockerStatus
		}
		return ""
	}
	getBlockingIsolationLevel := func() int64 {
		if result.BlockerIsolationLevel != nil {
			return *result.BlockerIsolationLevel
		}
		return 0
	}
	getBlockingOpenTransactionCount := func() int64 {
		if result.BlockerOpenTransactionCount != nil {
			return *result.BlockerOpenTransactionCount
		}
		return 0
	}
	getQueryText := func() string {
		if result.QueryStatementText != nil {
			return *result.QueryStatementText
		}
		return ""
	}
	getQueryID := func() string {
		if result.QueryID != nil && !result.QueryID.IsEmpty() {
			return result.QueryID.String()
		}
		return ""
	}
	getPlanHandle := func() string {
		if result.PlanHandle != nil && !result.PlanHandle.IsEmpty() {
			return result.PlanHandle.String()
		}
		return ""
	}
	getWaitTypeDescription := func() string {
		// This would be computed using helpers.DecodeWaitType()
		if result.WaitType != nil {
			return helpers.DecodeWaitType(*result.WaitType)
		}
		return ""
	}
	getWaitTypeCategory := func() string {
		// This would be computed using helpers.GetWaitTypeCategory()
		if result.WaitType != nil {
			return helpers.GetWaitTypeCategory(*result.WaitType)
		}
		return ""
	}
	getWaitResourceType := func() string {
		// This would be computed from wait_resource parsing
		if result.WaitResource != nil {
			resourceType, _ := helpers.DecodeWaitResource(*result.WaitResource)
			return resourceType
		}
		return ""
	}
	getWaitResourceDescription := func() string {
		// This would be computed with enhanced logic from addActiveQueryAttributes
		if result.WaitResource != nil {
			_, resourceDesc := helpers.DecodeWaitResource(*result.WaitResource)
			return resourceDesc
		}
		return ""
	}
	getWaitResourceSchemaName := func() string {
		if result.WaitResourceSchemaNameObject != nil {
			return *result.WaitResourceSchemaNameObject
		}
		return ""
	}
	getWaitResourceTableName := func() string {
		if result.WaitResourceObjectName != nil {
			return *result.WaitResourceObjectName
		}
		return ""
	}
	getWaitResourceObjectType := func() string {
		if result.WaitResourceObjectType != nil {
			return *result.WaitResourceObjectType
		}
		return ""
	}
	getWaitResourceIndexName := func() string {
		if result.WaitResourceIndexName != nil {
			return *result.WaitResourceIndexName
		}
		return ""
	}
	getWaitResourceIndexType := func() string {
		if result.WaitResourceIndexType != nil {
			return *result.WaitResourceIndexType
		}
		return ""
	}
	getLastWaitTypeDescription := func() string {
		if result.LastWaitType != nil {
			return helpers.DecodeWaitType(*result.LastWaitType)
		}
		return ""
	}

	// Active query wait time
	if result.WaitTimeS != nil && *result.WaitTimeS > 0 {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.wait_time_seconds",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Float64("value", *result.WaitTimeS),
			zap.Any("wait_type", result.WaitType),
			zap.Any("database_name", result.DatabaseName))

		s.mb.RecordSqlserverActivequeryWaitTimeSecondsDataPoint(
			timestamp,
			*result.WaitTimeS,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Warn("❌ SKIPPED wait_time metric (wait_time_s <= 0 or nil)",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Any("wait_time_s", result.WaitTimeS),
			zap.Any("wait_type", result.WaitType))
	}

	// Active query CPU time
	if result.CPUTimeMs != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.cpu_time_ms",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.CPUTimeMs))

		s.mb.RecordSqlserverActivequeryCPUTimeMsDataPoint(
			timestamp,
			*result.CPUTimeMs,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED cpu_time metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	// Active query elapsed time
	if result.TotalElapsedTimeMs != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.elapsed_time_ms",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.TotalElapsedTimeMs))

		s.mb.RecordSqlserverActivequeryElapsedTimeMsDataPoint(
			timestamp,
			*result.TotalElapsedTimeMs,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED elapsed_time metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	// Active query reads
	if result.Reads != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.reads",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.Reads))

		s.mb.RecordSqlserverActivequeryReadsDataPoint(
			timestamp,
			*result.Reads,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED reads metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	// Active query writes
	if result.Writes != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.writes",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.Writes))

		s.mb.RecordSqlserverActivequeryWritesDataPoint(
			timestamp,
			*result.Writes,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED writes metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	// Active query logical reads
	if result.LogicalReads != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.logical_reads",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.LogicalReads))

		s.mb.RecordSqlserverActivequeryLogicalReadsDataPoint(
			timestamp,
			*result.LogicalReads,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED logical_reads metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	// Active query row count
	if result.RowCount != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.row_count",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.RowCount))

		s.mb.RecordSqlserverActivequeryRowCountDataPoint(
			timestamp,
			*result.RowCount,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED row_count metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	if result.GrantedQueryMemoryPages != nil {
		s.logger.Info("✅ EMITTING METRIC: sqlserver.activequery.granted_query_memory_pages",
			zap.Any("session_id", result.CurrentSessionID),
			zap.Int64("value", *result.GrantedQueryMemoryPages))

		s.mb.RecordSqlserverActivequeryGrantedQueryMemoryPagesDataPoint(
			timestamp,
			*result.GrantedQueryMemoryPages,
			getSessionID(),
			getRequestID(),
			getDatabaseName(),
			getLoginName(),
			getHostName(),
			getProgramName(),
			getRequestCommand(),
			getRequestStatus(),
			getSessionStatus(),
			getClientInterfaceName(),
			getWaitType(),
			getWaitTypeDescription(),
			getWaitTypeCategory(),
			getWaitResource(),
			getWaitResourceObjectName(),
			getWaitResourceDatabaseName(),
			getWaitResourceType(),
			getWaitResourceDescription(),
			getWaitResourceSchemaName(),
			getWaitResourceTableName(),
			getWaitResourceObjectType(),
			getWaitResourceIndexName(),
			getWaitResourceIndexType(),
			getLastWaitType(),
			getLastWaitTypeDescription(),
			getRequestStartTime(),
			getCollectionTimestamp(),
			getTransactionID(),
			getOpenTransactionCount(),
			getTransactionIsolationLevel(),
			getDegreeOfParallelism(),
			getParallelWorkerCount(),
			getBlockingSessionID(),
			getBlockingLoginName(),
			getBlockingHostName(),
			getBlockingProgramName(),
			getBlockingStatus(),
			getBlockingIsolationLevel(),
			getBlockingOpenTransactionCount(),
			getQueryText(),
			getQueryID(),
			getPlanHandle(),
		)
	} else {
		s.logger.Debug("SKIPPED granted_query_memory_pages metric (nil)",
			zap.Any("session_id", result.CurrentSessionID))
	}

	return nil
}

// fetchExecutionPlanXML fetches the execution plan XML for a given plan_handle
// Simple wrapper for use by logs endpoint
func (s *QueryPerformanceScraper) fetchExecutionPlanXML(ctx context.Context, planHandle *models.QueryID) (string, error) {
	if planHandle == nil || planHandle.IsEmpty() {
		return "", nil
	}

	planHandleHex := planHandle.String()
	query := fmt.Sprintf(queries.ActiveQueryExecutionPlanQuery, planHandleHex)

	s.logger.Debug("Fetching execution plan XML",
		zap.String("plan_handle", planHandleHex))

	var results []struct {
		ExecutionPlanXML *string `db:"execution_plan_xml"`
	}

	if err := s.connection.Query(ctx, &results, query); err != nil {
		return "", fmt.Errorf("failed to fetch execution plan: %w", err)
	}

	if len(results) == 0 || results[0].ExecutionPlanXML == nil {
		return "", nil
	}

	return *results[0].ExecutionPlanXML, nil
}

// parseAndEmitExecutionPlanAsLogs parses execution plan XML and emits as OTLP logs
// Converts XML execution plan to custom events using attributes (not JSON body)
func (s *QueryPerformanceScraper) parseAndEmitExecutionPlanAsLogs(
	ctx context.Context,
	activeQuery models.ActiveRunningQuery,
	executionPlanXML string,
	logs plog.Logs,
) error {
	// Build execution plan analysis with metadata
	var queryID string
	if activeQuery.QueryID != nil {
		queryID = activeQuery.QueryID.String()
	}
	var planHandle string
	if activeQuery.PlanHandle != nil {
		planHandle = activeQuery.PlanHandle.String()
	}

	// Parse the XML execution plan
	analysis, err := models.ParseExecutionPlanXML(executionPlanXML, queryID, planHandle)
	if err != nil {
		return fmt.Errorf("failed to parse execution plan XML: %w", err)
	}

	// Emit as OTLP logs
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Emit one log record per operator node using ATTRIBUTES (not JSON body)
	for i := range analysis.Nodes {
		node := &analysis.Nodes[i]
		logRecord := scopeLogs.LogRecords().AppendEmpty()
		logRecord.SetTimestamp(timestamp)
		logRecord.SetObservedTimestamp(timestamp)
		logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
		logRecord.SetSeverityText("INFO")

		// Set body as simple string (not JSON)
		logRecord.Body().SetStr(fmt.Sprintf("Execution Plan Node: %s (NodeID=%d, Parent=%d)",
			node.PhysicalOp, node.NodeID, node.ParentNodeID))

		// Set ALL fields as ATTRIBUTES (this is what makes it a proper custom event)
		attrs := logRecord.Attributes()

		// IMPORTANT: Signal New Relic to ingest this as a Custom Event
		attrs.PutStr("newrelic.event.type", "SqlServerActiveQueryExecutionPlan")

		// Correlation keys
		attrs.PutStr("query_id", node.QueryID)
		attrs.PutStr("plan_handle", node.PlanHandle)
		if activeQuery.CurrentSessionID != nil {
			attrs.PutInt("session_id", *activeQuery.CurrentSessionID)
		}
		if activeQuery.RequestID != nil {
			attrs.PutInt("request_id", *activeQuery.RequestID)
		}
		if activeQuery.DatabaseName != nil {
			attrs.PutStr("database_name", *activeQuery.DatabaseName)
		}
		if activeQuery.RequestStartTime != nil {
			attrs.PutStr("start_time", *activeQuery.RequestStartTime)
		}

		// Node structure
		attrs.PutInt("node_id", int64(node.NodeID))
		attrs.PutInt("parent_node_id", int64(node.ParentNodeID))
		attrs.PutStr("input_type", node.InputType)

		// Operator information
		attrs.PutStr("physical_op", node.PhysicalOp)
		attrs.PutStr("logical_op", node.LogicalOp)
		attrs.PutStr("sql_text", node.SQLText)

		// Object information (for Index Scan/Seek operators)
		if node.SchemaName != "" {
			attrs.PutStr("schema_name", node.SchemaName)
		}
		if node.TableName != "" {
			attrs.PutStr("table_name", node.TableName)
		}
		if node.IndexName != "" {
			attrs.PutStr("index_name", node.IndexName)
		}
		if node.ReferencedColumns != "" {
			attrs.PutStr("referenced_columns", node.ReferencedColumns)
		}

		// Cost estimates
		attrs.PutDouble("estimate_rows", node.EstimateRows)
		attrs.PutDouble("estimate_io", node.EstimateIO)
		attrs.PutDouble("estimate_cpu", node.EstimateCPU)
		attrs.PutDouble("avg_row_size", node.AvgRowSize)
		attrs.PutDouble("total_subtree_cost", node.TotalSubtreeCost)
		attrs.PutDouble("estimated_operator_cost", node.EstimatedOperatorCost)

		// Execution details
		attrs.PutStr("estimated_execution_mode", node.EstimatedExecutionMode)
		attrs.PutInt("granted_memory_kb", node.GrantedMemoryKb)
		attrs.PutBool("spill_occurred", node.SpillOccurred)
		attrs.PutBool("no_join_predicate", node.NoJoinPredicate)

		// Performance metrics
		attrs.PutDouble("total_worker_time", node.TotalWorkerTime)
		attrs.PutDouble("total_elapsed_time", node.TotalElapsedTime)
		attrs.PutInt("total_logical_reads", node.TotalLogicalReads)
		attrs.PutInt("total_logical_writes", node.TotalLogicalWrites)
		attrs.PutInt("execution_count", node.ExecutionCount)
		attrs.PutDouble("avg_elapsed_time_ms", node.AvgElapsedTimeMs)

		// Timestamps
		if node.LastExecutionTime != "" {
			attrs.PutStr("last_execution_time", node.LastExecutionTime)
		}
	}

	s.logger.Info("Emitted execution plan as OTLP logs with attributes",
		zap.String("query_id", queryID),
		zap.String("plan_handle", planHandle),
		zap.Int("operator_count", len(analysis.Nodes)))

	return nil
}

// REMOVED (2025-11-28): Old execution plan functions that created node-level custom events
// - emitActiveQueryExecutionPlanLogs() - created one SqlServerActiveQueryExecutionPlan event per operator node
// - createActiveQueryExecutionPlanNodeLog() - helper function for node-level events
// - emitActiveQueryExecutionPlanMetrics() - created summary metrics from parsed XML
// These have been replaced with aggregated stats approach using dm_exec_query_stats

// REMOVED: Old execution plan fetching functions for active queries
// - fetchTop5PlanHandlesForActiveQuery
// - emitAggregatedExecutionPlanAsMetrics
// - createAggregatedExecutionPlanMetrics
//
// These functions have been replaced by the new approach in scraper_query_performance_montoring_metrics.go:
// - ScrapeSlowQueryExecutionPlans: Fetches plans for ALL slow queries (not just active ones)
// - emitActiveQueryPlanMetrics: Emits metrics with active query correlation (session_id, request_id, request_start_time)
//
// Benefits of the new approach:
// 1. Plans available even when query is NOT currently running
// 2. No duplicate fetches per active execution (fetched once per unique query_id)
// 3. Clean historical context without misleading active session attributes
