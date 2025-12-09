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

// UserConnectionScraper handles scraping of SQL Server user connection metrics
type UserConnectionScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	startTime     pcommon.Timestamp
	engineEdition int
	mb            *metadata.MetricsBuilder
}

// NewUserConnectionScraper creates a new UserConnectionScraper instance
func NewUserConnectionScraper(connection SQLConnectionInterface, logger *zap.Logger, engineEdition int, mb *metadata.MetricsBuilder) *UserConnectionScraper {
	return &UserConnectionScraper{
		connection:    connection,
		logger:        logger,
		startTime:     pcommon.NewTimestampFromTime(time.Now()),
		engineEdition: engineEdition,
		mb:            mb,
	}
}

// SetMetricsBuilder sets the metrics builder for the scraper
func (s *UserConnectionScraper) SetMetricsBuilder(mb *metadata.MetricsBuilder) {
	s.mb = mb
}

// ScrapeUserConnectionStatusMetrics collects user connection status distribution metrics
// This method retrieves the count of user connections grouped by their current status
func (s *UserConnectionScraper) ScrapeUserConnectionStatusMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server user connection status metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.status.metrics")
	if !found {
		return fmt.Errorf("no user connection status metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing user connection status metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.UserConnectionStatusMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute user connection status query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute user connection status query: %w", err)
	}

	// If no results, this may indicate no user connections or query issues
	if len(results) == 0 {
		s.logger.Debug("No user connection status metrics found - may indicate no active user connections")
		return nil
	}

	s.logger.Debug("Processing user connection status metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each status result
	for _, result := range results {
		if err := s.processUserConnectionStatusMetrics(result); err != nil {
			s.logger.Error("Failed to process user connection status metrics",
				zap.Error(err),
				zap.String("status", result.Status))
			return err
		}
	}

	return nil
}

// getQueryForMetric retrieves the appropriate query for a metric based on engine edition with Default fallback
func (s *UserConnectionScraper) getQueryForMetric(metricName string) (string, bool) {
	query, found := queries.GetQueryForMetric(queries.UserConnectionQueries, metricName, s.engineEdition)
	if found {
		s.logger.Debug("Using query for metric",
			zap.String("metric_name", metricName),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
	}
	return query, found
}

// processUserConnectionStatusMetrics processes user connection status metrics and creates OpenTelemetry metrics
func (s *UserConnectionScraper) processUserConnectionStatusMetrics(result models.UserConnectionStatusMetrics) error {
	// Process SessionCount as a gauge metric
	if result.SessionCount != nil {
		s.mb.RecordSqlserverUserConnectionsStatusCountDataPoint(
			pcommon.NewTimestampFromTime(time.Now()),
			int64(*result.SessionCount),
			result.Status,
		)
	}

	return nil
}

// ScrapeLoginLogoutMetrics collects login and logout rate metrics
// This method retrieves authentication activity counters from performance counters
func (s *UserConnectionScraper) ScrapeLoginLogoutMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server login/logout rate metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.authentication.metrics")
	if !found {
		return fmt.Errorf("no login/logout rate metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing login/logout rate metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.LoginLogoutMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute login/logout rate query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute login/logout rate query: %w", err)
	}

	// If no results, this may indicate no authentication activity or query issues
	if len(results) == 0 {
		s.logger.Debug("No login/logout rate metrics found - may indicate no recent authentication activity")
		return nil
	}

	s.logger.Debug("Processing login/logout rate metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each authentication result
	for _, result := range results {
		if err := s.processLoginLogoutMetrics(result); err != nil {
			s.logger.Error("Failed to process login/logout rate metrics",
				zap.Error(err),
				zap.String("counter_name", result.CounterName))
			return err
		}
	}

	return nil
}

// ScrapeLoginLogoutSummaryMetrics collects aggregated authentication activity statistics
func (s *UserConnectionScraper) ScrapeLoginLogoutSummaryMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server login/logout summary metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.authentication.summary")
	if !found {
		return fmt.Errorf("no login/logout summary metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing login/logout summary metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.LoginLogoutSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute login/logout summary query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute login/logout summary query: %w", err)
	}

	// If no results, this may indicate no authentication activity or query issues
	if len(results) == 0 {
		s.logger.Debug("No login/logout summary metrics found - may indicate no recent authentication activity")
		return nil
	}

	s.logger.Debug("Processing login/logout summary metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each summary result
	for _, result := range results {
		if err := s.processLoginLogoutSummaryMetrics(result); err != nil {
			s.logger.Error("Failed to process login/logout summary metrics",
				zap.Error(err))
			return err
		}
	}

	return nil
}

// processLoginLogoutMetrics processes login/logout rate metrics and creates OpenTelemetry metrics
func (s *UserConnectionScraper) processLoginLogoutMetrics(result models.LoginLogoutMetrics) error {
	// Process CntrValue as a gauge metric
	if result.CntrValue != nil {
		s.mb.RecordSqlserverUserConnectionsAuthenticationRateDataPoint(
			pcommon.NewTimestampFromTime(time.Now()),
			int64(*result.CntrValue),
			result.CounterName,
		)
	}

	return nil
}

// processLoginLogoutSummaryMetrics processes login/logout summary metrics and creates OpenTelemetry metrics
func (s *UserConnectionScraper) processLoginLogoutSummaryMetrics(result models.LoginLogoutSummary) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Process LoginsPerSec
	if result.LoginsPerSec != nil && *result.LoginsPerSec > 0 {
		s.mb.RecordSqlserverUserConnectionsAuthenticationLoginsPerSecDataPoint(
			timestamp,
			int64(*result.LoginsPerSec),
		)
	}

	// Process LogoutsPerSec
	if result.LogoutsPerSec != nil && *result.LogoutsPerSec > 0 {
		s.mb.RecordSqlserverUserConnectionsAuthenticationLogoutsPerSecDataPoint(
			timestamp,
			int64(*result.LogoutsPerSec),
		)
	}

	// Process TotalAuthActivity
	if result.TotalAuthActivity != nil && *result.TotalAuthActivity > 0 {
		s.mb.RecordSqlserverUserConnectionsAuthenticationTotalActivityDataPoint(
			timestamp,
			int64(*result.TotalAuthActivity),
		)
	}

	// Process ConnectionChurnRate
	if result.ConnectionChurnRate != nil && *result.ConnectionChurnRate >= 0 {
		s.mb.RecordSqlserverUserConnectionsAuthenticationChurnRateDataPoint(
			timestamp,
			float64(*result.ConnectionChurnRate),
		)
	}

	return nil
}

// ScrapeFailedLoginMetrics collects failed login attempts from SQL Server error log
// This method retrieves failed login messages from the error log for security monitoring
func (s *UserConnectionScraper) ScrapeFailedLoginMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server failed login metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.failed_logins.metrics")
	if !found {
		return fmt.Errorf("no failed login metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing failed login metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailedLoginMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute failed login query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute failed login query: %w", err)
	}

	s.logger.Debug("Processing failed login metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each failed login result
	for _, result := range results {
		if err := s.processFailedLoginMetrics(result); err != nil {
			logDate := ""
			if result.LogDate != nil {
				logDate = *result.LogDate
			}
			s.logger.Error("Failed to process failed login metrics",
				zap.Error(err),
				zap.String("log_date", logDate))
			return err
		}
	}

	return nil
}

// ScrapeFailedLoginSummaryMetrics collects aggregated failed login statistics
func (s *UserConnectionScraper) ScrapeFailedLoginSummaryMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server failed login summary metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.failed_logins_summary.metrics")
	if !found {
		return fmt.Errorf("no failed login summary metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing failed login summary metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.FailedLoginSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute failed login summary query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute failed login summary query: %w", err)
	}

	s.logger.Debug("Processing failed login summary metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each summary result
	for _, result := range results {
		if err := s.processFailedLoginSummaryMetrics(result); err != nil {
			s.logger.Error("Failed to process failed login summary metrics",
				zap.Error(err))
			return err
		}
	}

	return nil
}

// processFailedLoginMetrics processes failed login metrics and creates OpenTelemetry metrics
func (s *UserConnectionScraper) processFailedLoginMetrics(result models.FailedLoginMetrics) error {
	// Record failed login event count with attributes based on engine edition
	
	// Initialize all attribute values
	eventType := ""
	description := ""
	startTime := ""
	clientIP := ""
	logDate := ""
	processInfo := ""
	errorText := ""
	
	// Handle different query formats based on engine edition
	if s.engineEdition == queries.AzureSQLDatabaseEngineEdition {
		// Azure SQL Database format
		if result.EventType != nil {
			eventType = *result.EventType
		}
		if result.Description != nil {
			description = *result.Description
		}
		if result.StartTime != nil {
			startTime = *result.StartTime
		}
		if result.ClientIP != nil {
			clientIP = *result.ClientIP
		}
	} else {
		// Standard SQL Server format using sp_readerrorlog
		if result.LogDate != nil {
			logDate = *result.LogDate
		}
		if result.ProcessInfo != nil {
			processInfo = *result.ProcessInfo
		}
		if result.Text != nil {
			errorText = *result.Text
		}
	}
	
	s.mb.RecordSqlserverUserConnectionsAuthenticationFailedLoginEventsDataPoint(
		pcommon.NewTimestampFromTime(time.Now()),
		1, // Each record represents one failed login event
		eventType,
		description,
		startTime,
		clientIP,
		logDate,
		processInfo,
		errorText,
	)

	return nil
}

// ScrapeUserConnectionStatsMetrics scrapes user connection statistical analysis metrics
func (s *UserConnectionScraper) ScrapeUserConnectionStatsMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server user connection stats metrics")

	// Scrape summary metrics
	if err := s.ScrapeUserConnectionSummaryMetrics(ctx); err != nil {
		s.logger.Error("Failed to scrape user connection summary metrics", zap.Error(err))
		return fmt.Errorf("failed to scrape user connection summary metrics: %w", err)
	}

	// Scrape utilization metrics
	if err := s.ScrapeUserConnectionUtilizationMetrics(ctx); err != nil {
		s.logger.Error("Failed to scrape user connection utilization metrics", zap.Error(err))
		return fmt.Errorf("failed to scrape user connection utilization metrics: %w", err)
	}

	// Scrape client breakdown metrics
	if err := s.ScrapeUserConnectionByClientMetrics(ctx); err != nil {
		s.logger.Error("Failed to scrape user connection by client metrics", zap.Error(err))
		return fmt.Errorf("failed to scrape user connection by client metrics: %w", err)
	}

	// Scrape client summary metrics
	if err := s.ScrapeUserConnectionClientSummaryMetrics(ctx); err != nil {
		s.logger.Error("Failed to scrape user connection client summary metrics", zap.Error(err))
		return fmt.Errorf("failed to scrape user connection client summary metrics: %w", err)
	}

	return nil
}

// ScrapeUserConnectionSummaryMetrics scrapes user connection summary metrics
func (s *UserConnectionScraper) ScrapeUserConnectionSummaryMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server user connection summary metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.status.summary")
	if !found {
		return fmt.Errorf("no user connection summary metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing user connection summary metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.UserConnectionStatusSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute user connection summary query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute user connection summary query: %w", err)
	}

	// If no results, this may indicate no user connections
	if len(results) == 0 {
		s.logger.Debug("No user connection summary metrics found")
		return nil
	}

	s.logger.Debug("Processing user connection summary metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each summary result
	for _, result := range results {
		if err := s.processUserConnectionSummaryMetrics(result); err != nil {
			s.logger.Error("Failed to process user connection summary metrics",
				zap.Error(err))
			return err
		}
	}

	return nil
}

// ScrapeUserConnectionUtilizationMetrics scrapes user connection utilization metrics
func (s *UserConnectionScraper) ScrapeUserConnectionUtilizationMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server user connection utilization metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.utilization")
	if !found {
		return fmt.Errorf("no user connection utilization metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing user connection utilization metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.UserConnectionUtilization
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute user connection utilization query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute user connection utilization query: %w", err)
	}

	// If no results, this may indicate no user connections
	if len(results) == 0 {
		s.logger.Debug("No user connection utilization metrics found")
		return nil
	}

	s.logger.Debug("Processing user connection utilization metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each utilization result
	for _, result := range results {
		if err := s.processUserConnectionUtilizationMetrics(result); err != nil {
			s.logger.Error("Failed to process user connection utilization metrics",
				zap.Error(err))
			return err
		}
	}

	return nil
}

// ScrapeUserConnectionByClientMetrics scrapes user connection by client metrics
func (s *UserConnectionScraper) ScrapeUserConnectionByClientMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server user connection by client metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.by_client")
	if !found {
		return fmt.Errorf("no user connection by client metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing user connection by client metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.UserConnectionByClientMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute user connection by client query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute user connection by client query: %w", err)
	}

	// If no results, this may indicate no user connections
	if len(results) == 0 {
		s.logger.Debug("No user connection by client metrics found")
		return nil
	}

	s.logger.Debug("Processing user connection by client metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each client result
	for _, result := range results {
		if err := s.processUserConnectionByClientMetrics(result); err != nil {
			s.logger.Error("Failed to process user connection by client metrics",
				zap.Error(err),
				zap.String("program_name", result.ProgramName))
			return err
		}
	}

	return nil
}

// ScrapeUserConnectionClientSummaryMetrics scrapes user connection client summary metrics
func (s *UserConnectionScraper) ScrapeUserConnectionClientSummaryMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server user connection client summary metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("sqlserver.user_connections.client.summary")
	if !found {
		return fmt.Errorf("no user connection client summary metrics query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing user connection client summary metrics query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.UserConnectionClientSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute user connection client summary query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return fmt.Errorf("failed to execute user connection client summary query: %w", err)
	}

	// If no results, this may indicate no user connections
	if len(results) == 0 {
		s.logger.Debug("No user connection client summary metrics found")
		return nil
	}

	s.logger.Debug("Processing user connection client summary metrics results",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each client summary result
	for _, result := range results {
		if err := s.processUserConnectionClientSummaryMetrics(result); err != nil {
			s.logger.Error("Failed to process user connection client summary metrics",
				zap.Error(err))
			return err
		}
	}

	return nil
}

// processUserConnectionSummaryMetrics processes user connection summary metrics
func (s *UserConnectionScraper) processUserConnectionSummaryMetrics(result models.UserConnectionStatusSummary) error {
	s.logger.Debug("Processing user connection summary metrics")

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Process total user connections
	if result.TotalUserConnections != nil {
		s.mb.RecordSqlserverUserConnectionsTotalDataPoint(
			timestamp,
			int64(*result.TotalUserConnections),
		)
	}

	// Process sleeping connections
	if result.SleepingConnections != nil {
		s.mb.RecordSqlserverUserConnectionsSleepingDataPoint(
			timestamp,
			int64(*result.SleepingConnections),
		)
	}

	// Process running connections
	if result.RunningConnections != nil {
		s.mb.RecordSqlserverUserConnectionsRunningDataPoint(
			timestamp,
			int64(*result.RunningConnections),
		)
	}

	// Process suspended connections
	if result.SuspendedConnections != nil {
		s.mb.RecordSqlserverUserConnectionsSuspendedDataPoint(
			timestamp,
			int64(*result.SuspendedConnections),
		)
	}

	// Process runnable connections
	if result.RunnableConnections != nil {
		s.mb.RecordSqlserverUserConnectionsRunnableDataPoint(
			timestamp,
			int64(*result.RunnableConnections),
		)
	}

	// Process dormant connections
	if result.DormantConnections != nil {
		s.mb.RecordSqlserverUserConnectionsDormantDataPoint(
			timestamp,
			int64(*result.DormantConnections),
		)
	}

	return nil
}

// processUserConnectionUtilizationMetrics processes user connection utilization metrics
func (s *UserConnectionScraper) processUserConnectionUtilizationMetrics(result models.UserConnectionUtilization) error {
	s.logger.Debug("Processing user connection utilization metrics")

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Process active connection ratio
	if result.ActiveConnectionRatio != nil {
		s.mb.RecordSqlserverUserConnectionsUtilizationActiveRatioDataPoint(
			timestamp,
			float64(*result.ActiveConnectionRatio),
		)
	}

	// Process idle connection ratio
	if result.IdleConnectionRatio != nil {
		s.mb.RecordSqlserverUserConnectionsUtilizationIdleRatioDataPoint(
			timestamp,
			float64(*result.IdleConnectionRatio),
		)
	}

	// Process waiting connection ratio
	if result.WaitingConnectionRatio != nil {
		s.mb.RecordSqlserverUserConnectionsUtilizationWaitingRatioDataPoint(
			timestamp,
			float64(*result.WaitingConnectionRatio),
		)
	}

	// Process connection efficiency
	if result.ConnectionEfficiency != nil {
		s.mb.RecordSqlserverUserConnectionsUtilizationEfficiencyDataPoint(
			timestamp,
			float64(*result.ConnectionEfficiency),
		)
	}

	return nil
}

// processUserConnectionByClientMetrics processes user connection by client metrics
func (s *UserConnectionScraper) processUserConnectionByClientMetrics(result models.UserConnectionByClientMetrics) error {
	s.logger.Debug("Processing user connection by client metrics",
		zap.String("host_name", result.HostName),
		zap.String("program_name", result.ProgramName))

	if result.ConnectionCount != nil {
		s.mb.RecordSqlserverUserConnectionsClientCountDataPoint(
			pcommon.NewTimestampFromTime(time.Now()),
			int64(*result.ConnectionCount),
			result.HostName,
			result.ProgramName,
		)
	}

	return nil
}

// processUserConnectionClientSummaryMetrics processes user connection client summary metrics
func (s *UserConnectionScraper) processUserConnectionClientSummaryMetrics(result models.UserConnectionClientSummary) error {
	s.logger.Debug("Processing user connection client summary metrics")

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Process unique hosts
	if result.UniqueHosts != nil {
		s.mb.RecordSqlserverUserConnectionsClientUniqueHostsDataPoint(
			timestamp,
			int64(*result.UniqueHosts),
		)
	}

	// Process unique programs
	if result.UniquePrograms != nil {
		s.mb.RecordSqlserverUserConnectionsClientUniqueProgramsDataPoint(
			timestamp,
			int64(*result.UniquePrograms),
		)
	}

	// Process top host connection count
	if result.TopHostConnectionCount != nil {
		s.mb.RecordSqlserverUserConnectionsClientTopHostConnectionsDataPoint(
			timestamp,
			int64(*result.TopHostConnectionCount),
		)
	}

	// Process top program connection count
	if result.TopProgramConnectionCount != nil {
		s.mb.RecordSqlserverUserConnectionsClientTopProgramConnectionsDataPoint(
			timestamp,
			int64(*result.TopProgramConnectionCount),
		)
	}

	// Process hosts with multiple programs
	if result.HostsWithMultiplePrograms != nil {
		s.mb.RecordSqlserverUserConnectionsClientHostsMultiProgramDataPoint(
			timestamp,
			int64(*result.HostsWithMultiplePrograms),
		)
	}

	// Process programs from multiple hosts
	if result.ProgramsFromMultipleHosts != nil {
		s.mb.RecordSqlserverUserConnectionsClientProgramsMultiHostDataPoint(
			timestamp,
			int64(*result.ProgramsFromMultipleHosts),
		)
	}

	return nil
}

// processFailedLoginSummaryMetrics processes failed login summary metrics
func (s *UserConnectionScraper) processFailedLoginSummaryMetrics(result models.FailedLoginSummary) error {
	s.logger.Debug("Processing failed login summary metrics")

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Process total failed logins
	if result.TotalFailedLogins != nil {
		s.mb.RecordSqlserverUserConnectionsAuthenticationTotalFailedLoginsDataPoint(
			timestamp,
			int64(*result.TotalFailedLogins),
		)
	}

	// Process recent failed logins
	if result.RecentFailedLogins != nil {
		s.mb.RecordSqlserverUserConnectionsAuthenticationRecentFailedLoginsDataPoint(
			timestamp,
			int64(*result.RecentFailedLogins),
		)
	}

	// Process unique failed users
	if result.UniqueFailedUsers != nil {
		s.mb.RecordSqlserverUserConnectionsAuthenticationUniqueFailedUsersDataPoint(
			timestamp,
			int64(*result.UniqueFailedUsers),
		)
	}

	// Process unique failed sources
	if result.UniqueFailedSources != nil {
		s.mb.RecordSqlserverUserConnectionsAuthenticationUniqueFailedSourcesDataPoint(
			timestamp,
			int64(*result.UniqueFailedSources),
		)
	}

	return nil
}
