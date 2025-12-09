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

// WaitTimeScraper handles SQL Server wait time metrics collection
type WaitTimeScraper struct {
	connection    SQLConnectionInterface
	logger        *zap.Logger
	startTime     pcommon.Timestamp
	engineEdition int
	mb            *metadata.MetricsBuilder
}

// NewWaitTimeScraper creates a new wait time scraper instance
func NewWaitTimeScraper(connection SQLConnectionInterface, logger *zap.Logger, engineEdition int, mb *metadata.MetricsBuilder) *WaitTimeScraper {
	return &WaitTimeScraper{
		connection:    connection,
		logger:        logger,
		startTime:     pcommon.NewTimestampFromTime(time.Now()),
		engineEdition: engineEdition,
		mb:            mb,
	}
}

// getQueryForMetric retrieves the appropriate query for a metric based on engine edition with Default fallback
func (s *WaitTimeScraper) getQueryForMetric(metricName string) (string, bool) {
	query, found := queries.GetQueryForMetric(queries.WaitTimeQueries, metricName, s.engineEdition)
	return query, found
}

// SetMetricsBuilder sets the MetricsBuilder for the scraper
func (s *WaitTimeScraper) SetMetricsBuilder(mb *metadata.MetricsBuilder) {
	s.mb = mb
}

// ScrapeWaitTimeMetrics collects wait time statistics from SQL Server
func (s *WaitTimeScraper) ScrapeWaitTimeMetrics(ctx context.Context) error {
	query, found := s.getQueryForMetric("sqlserver.wait_stats.wait_time_metrics")
	if !found {
		return fmt.Errorf("no wait time metrics query available for engine edition %d", s.engineEdition)
	}

	var results []models.WaitTimeMetricsModel
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute wait time metrics query", zap.Error(err))
		return fmt.Errorf("failed to execute wait time metrics query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from wait time metrics query")
		return fmt.Errorf("no results returned from wait time metrics query")
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, result := range results {
		if result.WaitType == nil {
			continue
		}

		if result.WaitTimeMs != nil {
			s.mb.RecordSqlserverWaitStatsWaitTimeMsDataPoint(
				now,
				float64(*result.WaitTimeMs),
				*result.WaitType,
			)
		}

		if result.WaitingTasksCount != nil {
			s.mb.RecordSqlserverWaitStatsWaitingTasksCountDataPoint(
				now,
				float64(*result.WaitingTasksCount),
				*result.WaitType,
			)
		}
	}

	return nil
}

// ScrapeLatchWaitTimeMetrics collects latch-specific wait time statistics from SQL Server
func (s *WaitTimeScraper) ScrapeLatchWaitTimeMetrics(ctx context.Context) error {
	query, found := s.getQueryForMetric("sqlserver.wait_stats.latch.wait_time_metrics")
	if !found {
		return fmt.Errorf("no latch wait time metrics query available for engine edition %d", s.engineEdition)
	}

	var results []models.LatchWaitTimeMetricsModel
	if err := s.connection.Query(ctx, &results, query); err != nil {
		s.logger.Error("Failed to execute latch wait time query", zap.Error(err))
		return fmt.Errorf("failed to execute latch wait time query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from latch wait time query")
		return fmt.Errorf("no results returned from latch wait time query")
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, result := range results {
		if result.WaitType == nil {
			continue
		}

		if result.WaitTimeMs != nil {
			s.mb.RecordSqlserverWaitStatsLatchWaitTimeMsDataPoint(
				now,
				float64(*result.WaitTimeMs),
				*result.WaitType,
			)
		}

		if result.WaitingTasksCount != nil {
			s.mb.RecordSqlserverWaitStatsLatchWaitingTasksCountDataPoint(
				now,
				float64(*result.WaitingTasksCount),
				*result.WaitType,
			)
		}
	}

	return nil
}
