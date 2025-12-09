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

// DatabasePrincipalsScraper handles SQL Server database principals metrics collection
// This scraper provides comprehensive monitoring of database security principals
type DatabasePrincipalsScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	mb            *metadata.MetricsBuilder
	engineEdition int
}

// NewDatabasePrincipalsScraper creates a new database principals scraper
func NewDatabasePrincipalsScraper(conn SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder, engineEdition int) *DatabasePrincipalsScraper {
	return &DatabasePrincipalsScraper{
		connection:    conn,
		logger:        logger,
		mb:            mb,
		engineEdition: engineEdition,
	}
}

// ScrapeDatabasePrincipalsMetrics collects individual database principals metrics using engine-specific queries
func (s *DatabasePrincipalsScraper) ScrapeDatabasePrincipalsMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database principals metrics")

	// Get the appropriate query for this engine edition
	query, found := s.getQueryForMetric("database_principals")
	if !found {
		return fmt.Errorf("no database principals query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database principals query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabasePrincipalsMetrics
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database principals query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
		return fmt.Errorf("failed to query database principals: %w", err)
	}

	s.logger.Debug("Database principals query completed",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each principal result
	for _, result := range results {
		if err := s.processDatabasePrincipalsMetrics(result); err != nil {
			s.logger.Error("Failed to process database principals metrics",
				zap.Error(err),
				zap.String("principal_name", result.PrincipalName),
				zap.String("database_name", result.DatabaseName))
			return err
		}
	}

	return nil
}

// ScrapeDatabasePrincipalsSummaryMetrics collects aggregated database principals statistics
func (s *DatabasePrincipalsScraper) ScrapeDatabasePrincipalsSummaryMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database principals summary metrics")

	// Get the appropriate summary query for this engine edition
	query, found := s.getQueryForMetric("database_principals_summary")
	if !found {
		return fmt.Errorf("no database principals summary query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database principals summary query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabasePrincipalsSummary
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database principals summary query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
		return fmt.Errorf("failed to query database principals summary: %w", err)
	}

	s.logger.Debug("Database principals summary query completed",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each summary result
	for _, result := range results {
		if err := s.processDatabasePrincipalsSummaryMetrics(result); err != nil {
			s.logger.Error("Failed to process database principals summary metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			return err
		}
	}

	return nil
}

// ScrapeDatabasePrincipalActivityMetrics collects database principals activity and lifecycle metrics
func (s *DatabasePrincipalsScraper) ScrapeDatabasePrincipalActivityMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server database principals activity metrics")

	// Get the appropriate activity query for this engine edition
	query, found := s.getQueryForMetric("database_principals_activity")
	if !found {
		return fmt.Errorf("no database principals activity query available for engine edition %d", s.engineEdition)
	}

	s.logger.Debug("Executing database principals activity query",
		zap.String("query", queries.TruncateQuery(query, 100)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	var results []models.DatabasePrincipalActivity
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute database principals activity query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(query, 100)),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
		return fmt.Errorf("failed to query database principals activity: %w", err)
	}

	s.logger.Debug("Database principals activity query completed",
		zap.Int("result_count", len(results)),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	// Process each activity result
	for _, result := range results {
		if err := s.processDatabasePrincipalActivityMetrics(result); err != nil {
			s.logger.Error("Failed to process database principals activity metrics",
				zap.Error(err),
				zap.String("database_name", result.DatabaseName))
			return err
		}
	}

	return nil
}

// getQueryForMetric retrieves the appropriate query for a metric based on engine edition
func (s *DatabasePrincipalsScraper) getQueryForMetric(metricName string) (string, bool) {
	queryMap := map[string]map[int]string{
		"database_principals": {
			queries.StandardSQLServerEngineEdition:       queries.DatabasePrincipalsQuery,
			queries.AzureSQLDatabaseEngineEdition:        queries.DatabasePrincipalsQueryAzureSQL,
			queries.AzureSQLManagedInstanceEngineEdition: queries.DatabasePrincipalsQueryAzureMI,
			// Add support for other SQL Server editions (Enterprise, Standard, etc.)
			1: queries.DatabasePrincipalsQuery, // Personal/Desktop
			2: queries.DatabasePrincipalsQuery, // Standard
			3: queries.DatabasePrincipalsQuery, // Enterprise
			4: queries.DatabasePrincipalsQuery, // Express
			6: queries.DatabasePrincipalsQuery, // Azure Synapse Analytics
		},
		"database_principals_summary": {
			queries.StandardSQLServerEngineEdition:       queries.DatabasePrincipalsSummaryQuery,
			queries.AzureSQLDatabaseEngineEdition:        queries.DatabasePrincipalsSummaryQueryAzureSQL,
			queries.AzureSQLManagedInstanceEngineEdition: queries.DatabasePrincipalsSummaryQueryAzureMI,
			// Add support for other SQL Server editions
			1: queries.DatabasePrincipalsSummaryQuery, // Personal/Desktop
			2: queries.DatabasePrincipalsSummaryQuery, // Standard
			3: queries.DatabasePrincipalsSummaryQuery, // Enterprise
			4: queries.DatabasePrincipalsSummaryQuery, // Express
			6: queries.DatabasePrincipalsSummaryQuery, // Azure Synapse Analytics
		},
		"database_principals_activity": {
			queries.StandardSQLServerEngineEdition:       queries.DatabasePrincipalActivityQuery,
			queries.AzureSQLDatabaseEngineEdition:        queries.DatabasePrincipalActivityQueryAzureSQL,
			queries.AzureSQLManagedInstanceEngineEdition: queries.DatabasePrincipalActivityQueryAzureMI,
			// Add support for other SQL Server editions
			1: queries.DatabasePrincipalActivityQuery, // Personal/Desktop
			2: queries.DatabasePrincipalActivityQuery, // Standard
			3: queries.DatabasePrincipalActivityQuery, // Enterprise
			4: queries.DatabasePrincipalActivityQuery, // Express
			6: queries.DatabasePrincipalActivityQuery, // Azure Synapse Analytics
		},
	}

	if engineQueries, exists := queryMap[metricName]; exists {
		if query, found := engineQueries[s.engineEdition]; found {
			return query, true
		}
	}

	return "", false
}

// processDatabasePrincipalsMetrics processes individual database principals metrics and creates OpenTelemetry metrics
func (s *DatabasePrincipalsScraper) processDatabasePrincipalsMetrics(result models.DatabasePrincipalsMetrics) error {
	// Record creation date if available
	if result.CreateDate != nil {
		now := pcommon.NewTimestampFromTime(time.Now())
		timestamp := result.CreateDate.Unix()

		databaseName := ""
		if result.DatabaseName != "" {
			databaseName = result.DatabaseName
		}

		principalName := ""
		if result.PrincipalName != "" {
			principalName = result.PrincipalName
		}

		principalType := ""
		if result.TypeDesc != "" {
			principalType = result.TypeDesc
		}

		s.mb.RecordSqlserverDatabasePrincipalCreateDateDataPoint(now, timestamp, databaseName, principalName, principalType)

		s.logger.Debug("Recorded database principal creation date",
			zap.String("database_name", result.DatabaseName),
			zap.String("principal_name", result.PrincipalName),
			zap.String("principal_type", result.TypeDesc),
			zap.Int64("create_date_unix", timestamp))
	}

	return nil
}

// processDatabasePrincipalsSummaryMetrics processes summary metrics and creates OpenTelemetry metrics
func (s *DatabasePrincipalsScraper) processDatabasePrincipalsSummaryMetrics(result models.DatabasePrincipalsSummary) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	databaseName := ""
	if result.DatabaseName != "" {
		databaseName = result.DatabaseName
	}

	if result.TotalPrincipals != nil {
		s.mb.RecordSqlserverDatabasePrincipalsTotalDataPoint(now, *result.TotalPrincipals, databaseName)
	}
	if result.UserCount != nil {
		s.mb.RecordSqlserverDatabasePrincipalsUsersDataPoint(now, *result.UserCount, databaseName)
	}
	if result.RoleCount != nil {
		s.mb.RecordSqlserverDatabasePrincipalsRolesDataPoint(now, *result.RoleCount, databaseName)
	}
	if result.SQLUserCount != nil {
		s.mb.RecordSqlserverDatabasePrincipalsSQLUsersDataPoint(now, *result.SQLUserCount, databaseName)
	}
	if result.WindowsUserCount != nil {
		s.mb.RecordSqlserverDatabasePrincipalsWindowsUsersDataPoint(now, *result.WindowsUserCount, databaseName)
	}
	if result.ApplicationRoleCount != nil {
		s.mb.RecordSqlserverDatabasePrincipalsApplicationRolesDataPoint(now, *result.ApplicationRoleCount, databaseName)
	}

	s.logger.Debug("Recorded database principals summary metrics",
		zap.String("database_name", result.DatabaseName))

	return nil
}

// processDatabasePrincipalActivityMetrics processes activity metrics and creates OpenTelemetry metrics
func (s *DatabasePrincipalsScraper) processDatabasePrincipalActivityMetrics(result models.DatabasePrincipalActivity) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	databaseName := ""
	if result.DatabaseName != "" {
		databaseName = result.DatabaseName
	}

	if result.RecentPrincipals != nil {
		s.mb.RecordSqlserverDatabasePrincipalsRecentlyCreatedDataPoint(now, *result.RecentPrincipals, databaseName)
	}
	if result.OldPrincipals != nil {
		s.mb.RecordSqlserverDatabasePrincipalsOldDataPoint(now, *result.OldPrincipals, databaseName)
	}
	if result.OrphanedUsers != nil {
		s.mb.RecordSqlserverDatabasePrincipalsOrphanedUsersDataPoint(now, *result.OrphanedUsers, databaseName)
	}

	s.logger.Debug("Recorded database principals activity metrics",
		zap.String("database_name", result.DatabaseName))

	return nil
}
