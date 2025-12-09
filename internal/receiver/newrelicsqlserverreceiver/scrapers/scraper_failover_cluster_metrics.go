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

// FailoverClusterScraper handles SQL Server Always On Availability Group replica metrics collection
type FailoverClusterScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	mb            *metadata.MetricsBuilder
	engineEdition int
}

// NewFailoverClusterScraper creates a new failover cluster scraper
func NewFailoverClusterScraper(conn SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder, engineEdition int) *FailoverClusterScraper {
	return &FailoverClusterScraper{
		connection:    conn,
		logger:        logger,
		mb:            mb,
		engineEdition: engineEdition,
	}
}

// ScrapeFailoverClusterMetrics collects Always On Availability Group replica performance metrics
// This method is only applicable to SQL Server deployments with Always On AG enabled
func (s *FailoverClusterScraper) ScrapeFailoverClusterMetrics(ctx context.Context) error {
	// Skip failover cluster metrics for Azure SQL Database - Always On AG is not supported
	if s.engineEdition == 5 { // Azure SQL Database
		s.logger.Debug("Skipping failover cluster replica metrics - not supported in Azure SQL Database")
		return nil
	}

	s.logger.Debug("Scraping SQL Server Always On failover cluster replica metrics")

	// Get the appropriate query for this engine edition using centralized query selection
	query, found := queries.GetQueryForMetric(queries.FailoverClusterQueries, "sqlserver.failover_cluster.replica_metrics", s.engineEdition)
	if !found {
		return fmt.Errorf("no failover cluster replica metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing failover cluster replica metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailoverClusterReplicaMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute failover cluster replica query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute failover cluster replica query: %w", err)
	}

	// If no results, this SQL Server instance does not have Always On AG enabled or configured
	if len(results) == 0 {
		s.logger.Debug("No Always On replica metrics found - SQL Server may not have Always On Availability Groups enabled")
		return nil
	}

	s.logger.Debug("Processing failover cluster replica metrics results",
		zap.Int("result_count", len(results)))

	// Process each replica's metrics
	for _, result := range results {
		if err := s.processFailoverClusterReplicaMetrics(result); err != nil {
			s.logger.Error("Failed to process failover cluster replica metrics",
				zap.Error(err))
			continue
		}

		s.logger.Info("Successfully scraped SQL Server Always On replica metrics",
			zap.Int64p("log_bytes_received_per_sec", result.LogBytesReceivedPerSec),
			zap.Int64p("transaction_delay_ms", result.TransactionDelayMs),
			zap.Int64p("flow_control_time_ms", result.FlowControlTimeMs))
	}

	s.logger.Debug("Successfully scraped failover cluster replica metrics",
		zap.Int("result_count", len(results)))

	return nil
}

// ScrapeFailoverClusterReplicaStateMetrics collects Always On Availability Group database replica state metrics
// This method provides detailed log synchronization metrics for each database in the availability group
func (s *FailoverClusterScraper) ScrapeFailoverClusterReplicaStateMetrics(ctx context.Context) error {
	// Skip failover cluster metrics for Azure SQL Database - Always On AG is not supported
	if s.engineEdition == 5 { // Azure SQL Database
		s.logger.Debug("Skipping failover cluster replica state metrics - not supported in Azure SQL Database")
		return nil
	}

	s.logger.Debug("Scraping SQL Server Always On failover cluster replica state metrics")

	// Get the appropriate query for this engine edition using centralized query selection
	query, found := queries.GetQueryForMetric(queries.FailoverClusterQueries, "sqlserver.failover_cluster.replica_state_metrics", s.engineEdition)
	if !found {
		return fmt.Errorf("no failover cluster replica state metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing failover cluster replica state metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailoverClusterReplicaStateMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute failover cluster replica state query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute failover cluster replica state query: %w", err)
	}

	// If no results, this SQL Server instance does not have Always On AG enabled or configured
	if len(results) == 0 {
		s.logger.Debug("No Always On replica state metrics found - SQL Server may not have Always On Availability Groups enabled")
		return nil
	}

	s.logger.Debug("Processing failover cluster replica state metrics results",
		zap.Int("result_count", len(results)))

	// Process each replica state's metrics
	for _, result := range results {
		if err := s.processFailoverClusterReplicaStateMetrics(result); err != nil {
			s.logger.Error("Failed to process failover cluster replica state metrics",
				zap.Error(err))
			continue
		}

		s.logger.Info("Successfully scraped SQL Server Always On replica state metrics",
			zap.String("replica_server_name", result.ReplicaServerName),
			zap.String("database_name", result.DatabaseName),
			zap.Int64p("log_send_queue_kb", result.LogSendQueueKB),
			zap.Int64p("redo_queue_kb", result.RedoQueueKB),
			zap.Int64p("redo_rate_kb_sec", result.RedoRateKBSec))
	}

	s.logger.Debug("Successfully scraped failover cluster replica state metrics",
		zap.Int("result_count", len(results)))

	return nil
}

// processFailoverClusterReplicaMetrics processes replica metrics and creates OpenTelemetry metrics
func (s *FailoverClusterScraper) processFailoverClusterReplicaMetrics(result models.FailoverClusterReplicaMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	instanceName := ""
	if result.InstanceName != "" {
		instanceName = result.InstanceName
	}

	if result.LogBytesReceivedPerSec != nil {
		s.mb.RecordSqlserverFailoverClusterLogBytesReceivedPerSecDataPoint(now, *result.LogBytesReceivedPerSec, instanceName)
	}
	if result.TransactionDelayMs != nil {
		s.mb.RecordSqlserverFailoverClusterTransactionDelayMsDataPoint(now, *result.TransactionDelayMs, instanceName)
	}
	if result.FlowControlTimeMs != nil {
		s.mb.RecordSqlserverFailoverClusterFlowControlTimeMsDataPoint(now, *result.FlowControlTimeMs, instanceName)
	}

	s.logger.Debug("Recorded failover cluster replica metrics",
		zap.String("instance_name", result.InstanceName))

	return nil
}

// processFailoverClusterReplicaStateMetrics processes replica state metrics and creates OpenTelemetry metrics
func (s *FailoverClusterScraper) processFailoverClusterReplicaStateMetrics(result models.FailoverClusterReplicaStateMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	replicaServerName := ""
	if result.ReplicaServerName != "" {
		replicaServerName = result.ReplicaServerName
	}

	databaseName := ""
	if result.DatabaseName != "" {
		databaseName = result.DatabaseName
	}

	if result.LogSendQueueKB != nil {
		s.mb.RecordSqlserverFailoverClusterLogSendQueueKbDataPoint(now, *result.LogSendQueueKB, replicaServerName, databaseName)
	}
	if result.RedoQueueKB != nil {
		s.mb.RecordSqlserverFailoverClusterRedoQueueKbDataPoint(now, *result.RedoQueueKB, replicaServerName, databaseName)
	}
	if result.RedoRateKBSec != nil {
		s.mb.RecordSqlserverFailoverClusterRedoRateKbSecDataPoint(now, *result.RedoRateKBSec, replicaServerName, databaseName)
	}

	s.logger.Debug("Recorded failover cluster replica state metrics",
		zap.String("replica_server_name", result.ReplicaServerName),
		zap.String("database_name", result.DatabaseName))

	return nil
}

// ScrapeFailoverClusterAvailabilityGroupHealthMetrics collects Availability Group health status
// This method retrieves health and role information for all availability group replicas
func (s *FailoverClusterScraper) ScrapeFailoverClusterAvailabilityGroupHealthMetrics(ctx context.Context) error {
	// Skip failover cluster metrics for Azure SQL Database - Always On AG is not supported
	if s.engineEdition == 5 { // Azure SQL Database
		s.logger.Debug("Skipping Availability Group health metrics - not supported in Azure SQL Database")
		return nil
	}

	s.logger.Debug("Scraping SQL Server Availability Group health metrics")

	// Get the appropriate query for this engine edition using centralized query selection
	query, found := queries.GetQueryForMetric(queries.FailoverClusterQueries, "sqlserver.failover_cluster.availability_group_health_metrics", s.engineEdition)
	if !found {
		return fmt.Errorf("no availability group health metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing availability group health metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailoverClusterAvailabilityGroupHealthMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute availability group health query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute availability group health query: %w", err)
	}

	// If no results, this SQL Server instance may not have Always On AG configured
	if len(results) == 0 {
		s.logger.Debug("No availability group health metrics found - SQL Server may not have Always On Availability Groups configured")
		return nil
	}

	s.logger.Debug("Processing availability group health metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each availability group health result
	for _, result := range results {
		if err := s.processFailoverClusterAvailabilityGroupHealthMetrics(result); err != nil {
			s.logger.Error("Failed to process availability group health metrics",
				zap.Error(err),
				zap.String("replica_server_name", result.ReplicaServerName))
			return err
		}
	}

	return nil
}

// processFailoverClusterAvailabilityGroupHealthMetrics processes availability group health metrics and creates OpenTelemetry metrics
func (s *FailoverClusterScraper) processFailoverClusterAvailabilityGroupHealthMetrics(result models.FailoverClusterAvailabilityGroupHealthMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	replicaServerName := ""
	if result.ReplicaServerName != "" {
		replicaServerName = result.ReplicaServerName
	}

	roleDesc := ""
	if result.RoleDesc != "" {
		roleDesc = result.RoleDesc
	}

	synchronizationHealthDesc := ""
	if result.SynchronizationHealthDesc != "" {
		synchronizationHealthDesc = result.SynchronizationHealthDesc
	}

	// Record role as info metric
	s.mb.RecordSqlserverFailoverClusterAgReplicaRoleDataPoint(now, 1, replicaServerName, roleDesc, synchronizationHealthDesc)

	// Record synchronization health as info metric
	s.mb.RecordSqlserverFailoverClusterAgSynchronizationHealthDataPoint(now, 1, replicaServerName, roleDesc, synchronizationHealthDesc)

	s.logger.Debug("Recorded availability group health metrics",
		zap.String("replica_server_name", result.ReplicaServerName),
		zap.String("role_desc", result.RoleDesc),
		zap.String("synchronization_health_desc", result.SynchronizationHealthDesc))

	return nil
}

// ScrapeFailoverClusterAvailabilityGroupMetrics collects Availability Group configuration and status
// This method retrieves detailed configuration and state information for all availability groups
func (s *FailoverClusterScraper) ScrapeFailoverClusterAvailabilityGroupMetrics(ctx context.Context) error {
	// Skip failover cluster metrics for Azure SQL Database - Always On AG is not supported
	if s.engineEdition == 5 { // Azure SQL Database
		s.logger.Debug("Skipping Availability Group configuration metrics - not supported in Azure SQL Database")
		return nil
	}

	s.logger.Debug("Scraping SQL Server Availability Group configuration metrics")

	// Get the appropriate query for this engine edition using centralized query selection
	query, found := queries.GetQueryForMetric(queries.FailoverClusterQueries, "sqlserver.failover_cluster.availability_group_metrics", s.engineEdition)
	if !found {
		return fmt.Errorf("no availability group configuration metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing availability group configuration metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailoverClusterAvailabilityGroupMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute availability group configuration query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute availability group configuration query: %w", err)
	}

	// If no results, this SQL Server instance may not have Always On AG configured
	if len(results) == 0 {
		s.logger.Debug("No availability group configuration metrics found - SQL Server may not have Always On Availability Groups configured")
		return nil
	}

	s.logger.Debug("Processing availability group configuration metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each availability group configuration result
	for _, result := range results {
		if err := s.processFailoverClusterAvailabilityGroupMetrics(result); err != nil {
			s.logger.Error("Failed to process availability group configuration metrics",
				zap.Error(err),
				zap.String("group_name", result.GroupName))
			return err
		}
	}

	return nil
}

// processFailoverClusterAvailabilityGroupMetrics processes availability group configuration metrics and creates OpenTelemetry metrics
func (s *FailoverClusterScraper) processFailoverClusterAvailabilityGroupMetrics(result models.FailoverClusterAvailabilityGroupMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	groupName := ""
	if result.GroupName != "" {
		groupName = result.GroupName
	}

	clusterTypeDesc := ""
	if result.ClusterTypeDesc != "" {
		clusterTypeDesc = result.ClusterTypeDesc
	}

	if result.FailureConditionLevel != nil {
		s.mb.RecordSqlserverFailoverClusterAgFailureConditionLevelDataPoint(now, *result.FailureConditionLevel, groupName, clusterTypeDesc)
	}
	if result.HealthCheckTimeout != nil {
		s.mb.RecordSqlserverFailoverClusterAgHealthCheckTimeoutDataPoint(now, *result.HealthCheckTimeout, groupName, clusterTypeDesc)
	}

	// Record cluster type as info metric
	s.mb.RecordSqlserverFailoverClusterAgClusterTypeDataPoint(now, 1, groupName, clusterTypeDesc)

	s.logger.Debug("Recorded availability group configuration metrics",
		zap.String("group_name", result.GroupName),
		zap.String("cluster_type_desc", result.ClusterTypeDesc))

	return nil
}

// ScrapeFailoverClusterPerformanceCounterMetrics collects Always On performance counter metrics
// This method retrieves key performance metrics for availability group log transport
// ScrapeFailoverClusterRedoQueueMetrics collects Always On redo queue metrics
// This method retrieves log send queue, redo queue, and redo rate metrics for monitoring replication performance
// Compatible with both Standard SQL Server and Azure SQL Managed Instance
func (s *FailoverClusterScraper) ScrapeFailoverClusterRedoQueueMetrics(ctx context.Context) error {
	// Skip for all engine types except Azure SQL Managed Instance
	if s.engineEdition != 8 { // Azure SQL Managed Instance
		s.logger.Debug("Skipping failover cluster redo queue metrics - only supported in Azure SQL Managed Instance",
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
		return nil
	}

	s.logger.Debug("Scraping SQL Server Always On redo queue metrics")

	// Get the appropriate query for this engine edition using centralized query selection
	query, found := queries.GetQueryForMetric(queries.FailoverClusterQueries, "sqlserver.failover_cluster.redo_queue_metrics", s.engineEdition)
	if !found {
		return fmt.Errorf("no failover cluster redo queue metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing failover cluster redo queue metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailoverClusterRedoQueueMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute failover cluster redo queue query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute failover cluster redo queue query: %w", err)
	}

	// If no results, this Azure SQL Managed Instance may not have Always On AG enabled or configured
	if len(results) == 0 {
		s.logger.Debug("No Always On redo queue metrics found - Azure SQL Managed Instance may not have Always On Availability Groups enabled")
		return nil
	}

	s.logger.Debug("Processing failover cluster redo queue metrics results",
		zap.Int("result_count", len(results)))

	// Process each redo queue result
	for _, result := range results {
		if err := s.processFailoverClusterRedoQueueMetrics(result); err != nil {
			s.logger.Error("Failed to process failover cluster redo queue metrics",
				zap.Error(err),
				zap.String("replica_server_name", result.ReplicaServerName),
				zap.String("database_name", result.DatabaseName))
			continue
		}

		s.logger.Info("Successfully scraped SQL Server Always On redo queue metrics",
			zap.String("replica_server_name", result.ReplicaServerName),
			zap.String("database_name", result.DatabaseName),
			zap.Int64p("log_send_queue_kb", result.LogSendQueueKB),
			zap.Int64p("redo_queue_kb", result.RedoQueueKB),
			zap.Int64p("redo_rate_kb_sec", result.RedoRateKBSec))
	}

	s.logger.Debug("Successfully scraped failover cluster redo queue metrics",
		zap.Int("result_count", len(results)))

	return nil
}

// processFailoverClusterRedoQueueMetrics processes redo queue metrics and creates OpenTelemetry metrics
func (s *FailoverClusterScraper) processFailoverClusterRedoQueueMetrics(result models.FailoverClusterRedoQueueMetrics) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	replicaServerName := ""
	if result.ReplicaServerName != "" {
		replicaServerName = result.ReplicaServerName
	}

	databaseName := ""
	if result.DatabaseName != "" {
		databaseName = result.DatabaseName
	}

	if result.LogSendQueueKB != nil {
		s.mb.RecordSqlserverFailoverClusterLogSendQueueKbDataPoint(now, *result.LogSendQueueKB, replicaServerName, databaseName)
	}
	if result.RedoQueueKB != nil {
		s.mb.RecordSqlserverFailoverClusterRedoQueueKbDataPoint(now, *result.RedoQueueKB, replicaServerName, databaseName)
	}
	if result.RedoRateKBSec != nil {
		s.mb.RecordSqlserverFailoverClusterRedoRateKbSecDataPoint(now, *result.RedoRateKBSec, replicaServerName, databaseName)
	}

	s.logger.Debug("Recorded failover cluster redo queue metrics",
		zap.String("replica_server_name", result.ReplicaServerName),
		zap.String("database_name", result.DatabaseName))

	return nil
}
