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

// ThreadPoolHealthScraper handles thread pool health metrics collection
type ThreadPoolHealthScraper struct {
	connection SQLConnectionInterface
	logger     *zap.Logger
	mb         *metadata.MetricsBuilder
}

// NewThreadPoolHealthScraper creates a new thread pool health scraper
func NewThreadPoolHealthScraper(connection SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder) *ThreadPoolHealthScraper {
	return &ThreadPoolHealthScraper{
		connection: connection,
		logger:     logger,
		mb:         mb,
	}
}

// SetMetricsBuilder sets the MetricsBuilder for this scraper
func (s *ThreadPoolHealthScraper) SetMetricsBuilder(mb *metadata.MetricsBuilder) {
	s.mb = mb
}

// ScrapeThreadPoolHealthMetrics collects thread pool health metrics
func (s *ThreadPoolHealthScraper) ScrapeThreadPoolHealthMetrics(ctx context.Context) error {
	s.logger.Debug("Executing thread pool health metrics collection")

	var results []models.ThreadPoolHealth
	if err := s.connection.Query(ctx, &results, queries.ThreadPoolHealthQuery); err != nil {
		return fmt.Errorf("failed to execute thread pool health query: %w", err)
	}

	s.logger.Debug("Thread pool health metrics fetched", zap.Int("result_count", len(results)))

	for i, result := range results {
		if err := s.processThreadPoolHealthMetrics(result, i); err != nil {
			s.logger.Error("Failed to process thread pool health metric", zap.Error(err), zap.Int("index", i))
		}
	}

	return nil
}

// processThreadPoolHealthMetrics processes and emits metrics for thread pool health
func (s *ThreadPoolHealthScraper) processThreadPoolHealthMetrics(result models.ThreadPoolHealth, index int) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Extract attributes with nil checks
	sqlHostname := ""
	if result.SQLHostname != nil {
		sqlHostname = *result.SQLHostname
	}
	healthStatus := ""
	if result.HealthStatus != nil {
		healthStatus = *result.HealthStatus
	}
	collectionTimestamp := ""
	if result.CollectionTimestamp != nil {
		collectionTimestamp = *result.CollectionTimestamp
	}

	if result.CurrentlyWaitingForThreadpool != nil {
		s.mb.RecordSqlserverThreadpoolWaitingTasksDataPoint(timestamp, *result.CurrentlyWaitingForThreadpool, sqlHostname, healthStatus, collectionTimestamp)
	}

	if result.TotalWorkQueueCount != nil {
		s.mb.RecordSqlserverThreadpoolWorkQueueCountDataPoint(timestamp, *result.TotalWorkQueueCount, sqlHostname, healthStatus, collectionTimestamp)
	}

	if result.RunningWorkers != nil {
		s.mb.RecordSqlserverThreadpoolRunningWorkersDataPoint(timestamp, *result.RunningWorkers, sqlHostname, healthStatus, collectionTimestamp)
	}

	if result.MaxConfiguredWorkers != nil {
		s.mb.RecordSqlserverThreadpoolMaxWorkersDataPoint(timestamp, *result.MaxConfiguredWorkers, sqlHostname, healthStatus, collectionTimestamp)
	}

	if result.WorkerUtilizationPercent != nil {
		s.mb.RecordSqlserverThreadpoolUtilizationPercentDataPoint(timestamp, *result.WorkerUtilizationPercent, sqlHostname, healthStatus, collectionTimestamp)
	}

	if result.TotalCurrentTasks != nil {
		s.mb.RecordSqlserverThreadpoolCurrentTasksDataPoint(timestamp, *result.TotalCurrentTasks, sqlHostname, healthStatus, collectionTimestamp)
	}

	if result.TotalRunnableTasks != nil {
		s.mb.RecordSqlserverThreadpoolRunnableTasksDataPoint(timestamp, *result.TotalRunnableTasks, sqlHostname, healthStatus, collectionTimestamp)
	}

	return nil
}
