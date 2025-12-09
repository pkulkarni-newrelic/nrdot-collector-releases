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

// SecurityScraper handles SQL Server security-level metrics collection
type SecurityScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	mb            *metadata.MetricsBuilder
	startTime     pcommon.Timestamp
	engineEdition int
}

// NewSecurityScraper creates a new security scraper
func NewSecurityScraper(conn SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder, engineEdition int) *SecurityScraper {
	return &SecurityScraper{
		connection:    conn,
		logger:        logger,
		mb:            mb,
		engineEdition: engineEdition,
	}
}

// SetMetricsBuilder sets the metrics builder for this scraper
func (s *SecurityScraper) SetMetricsBuilder(mb *metadata.MetricsBuilder) {
	s.mb = mb
}

// ScrapeSecurityPrincipalsMetrics scrapes server principals count metrics
func (s *SecurityScraper) ScrapeSecurityPrincipalsMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server security principals metrics")

	var results []models.SecurityPrincipalsModel
	if err := s.connection.Query(ctx, &results, queries.SecurityPrincipalsQuery); err != nil {
		s.logger.Error("Failed to execute security principals query", zap.Error(err))
		return fmt.Errorf("failed to execute security principals query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from security principals query")
		return fmt.Errorf("no results returned from security principals query")
	}

	result := results[0]
	if result.ServerPrincipalsCount == nil {
		s.logger.Error("Security principals metric is null - invalid query result")
		return fmt.Errorf("security principals metric is null in query result")
	}

	if err := s.processSecurityPrincipalsMetrics(result); err != nil {
		s.logger.Error("Failed to process security principals metrics", zap.Error(err))
		return fmt.Errorf("failed to process security principals metrics: %w", err)
	}

	s.logger.Debug("Successfully scraped security principals metrics")
	return nil
}

// processSecurityPrincipalsMetrics converts security principals metrics to OpenTelemetry format
func (s *SecurityScraper) processSecurityPrincipalsMetrics(result models.SecurityPrincipalsModel) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	if result.ServerPrincipalsCount != nil {
		s.mb.RecordSqlserverSecurityServerPrincipalsCountDataPoint(timestamp, *result.ServerPrincipalsCount, "gauge")
	}

	return nil
}

// ScrapeSecurityRoleMembersMetrics scrapes server role membership count metrics
func (s *SecurityScraper) ScrapeSecurityRoleMembersMetrics(ctx context.Context) error {
	s.logger.Debug("Scraping SQL Server security role members metrics")

	var results []models.SecurityRoleMembersModel
	if err := s.connection.Query(ctx, &results, queries.SecurityRoleMembersQuery); err != nil {
		s.logger.Error("Failed to execute security role members query", zap.Error(err))
		return fmt.Errorf("failed to execute security role members query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from security role members query")
		return fmt.Errorf("no results returned from security role members query")
	}

	result := results[0]
	if result.ServerRoleMembersCount == nil {
		s.logger.Error("Security role members metric is null - invalid query result")
		return fmt.Errorf("security role members metric is null in query result")
	}

	if err := s.processSecurityRoleMembersMetrics(result); err != nil {
		s.logger.Error("Failed to process security role members metrics", zap.Error(err))
		return fmt.Errorf("failed to process security role members metrics: %w", err)
	}

	s.logger.Debug("Successfully scraped security role members metrics")
	return nil
}

// processSecurityRoleMembersMetrics converts security role members metrics to OpenTelemetry format
func (s *SecurityScraper) processSecurityRoleMembersMetrics(result models.SecurityRoleMembersModel) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	if result.ServerRoleMembersCount != nil {
		s.mb.RecordSqlserverSecurityServerRoleMembersCountDataPoint(timestamp, *result.ServerRoleMembersCount, "gauge")
	}

	return nil
}
