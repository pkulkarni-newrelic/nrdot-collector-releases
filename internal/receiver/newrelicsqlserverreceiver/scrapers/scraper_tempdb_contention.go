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

// TempDBContentionScraper handles TempDB contention metrics collection
type TempDBContentionScraper struct {
	connection SQLConnectionInterface
	logger     *zap.Logger
	mb         *metadata.MetricsBuilder
}

// NewTempDBContentionScraper creates a new TempDB contention scraper
func NewTempDBContentionScraper(connection SQLConnectionInterface, logger *zap.Logger, mb *metadata.MetricsBuilder) *TempDBContentionScraper {
	return &TempDBContentionScraper{
		connection: connection,
		logger:     logger,
		mb:         mb,
	}
}

// SetMetricsBuilder sets the MetricsBuilder for this scraper
func (s *TempDBContentionScraper) SetMetricsBuilder(mb *metadata.MetricsBuilder) {
	s.mb = mb
}

// ScrapeTempDBContentionMetrics collects TempDB contention metrics
func (s *TempDBContentionScraper) ScrapeTempDBContentionMetrics(ctx context.Context) error {
	s.logger.Debug("Executing TempDB contention metrics collection")

	var results []models.TempDBContention
	if err := s.connection.Query(ctx, &results, queries.TempDBContentionQuery); err != nil {
		return fmt.Errorf("failed to execute TempDB contention query: %w", err)
	}

	s.logger.Debug("TempDB contention metrics fetched", zap.Int("result_count", len(results)))

	for i, result := range results {
		if err := s.processTempDBContentionMetrics(result, i); err != nil {
			s.logger.Error("Failed to process TempDB contention metric", zap.Error(err), zap.Int("index", i))
		}
	}

	return nil
}

// processTempDBContentionMetrics processes and emits metrics for TempDB contention
func (s *TempDBContentionScraper) processTempDBContentionMetrics(result models.TempDBContention, index int) error {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Extract attributes with nil checks
	sqlHostname := ""
	if result.SQLHostname != nil {
		sqlHostname = *result.SQLHostname
	}
	tempdbHealthStatus := ""
	if result.TempDBHealthStatus != nil {
		tempdbHealthStatus = *result.TempDBHealthStatus
	}
	collectionTimestamp := ""
	if result.CollectionTimestamp != nil {
		collectionTimestamp = *result.CollectionTimestamp
	}

	if result.CurrentWaiters != nil {
		s.mb.RecordSqlserverTempdbCurrentWaitersDataPoint(timestamp, *result.CurrentWaiters, sqlHostname, tempdbHealthStatus, collectionTimestamp)
	}

	if result.PagelatchWaits != nil {
		s.mb.RecordSqlserverTempdbPagelatchWaitsMsDataPoint(timestamp, *result.PagelatchWaits, sqlHostname, tempdbHealthStatus, collectionTimestamp)
	}

	if result.AllocationWaits != nil {
		s.mb.RecordSqlserverTempdbAllocationWaitsMsDataPoint(timestamp, *result.AllocationWaits, sqlHostname, tempdbHealthStatus, collectionTimestamp)
	}

	if result.TempDBDataFileCount != nil {
		s.mb.RecordSqlserverTempdbDataFileCountDataPoint(timestamp, *result.TempDBDataFileCount, sqlHostname, tempdbHealthStatus, collectionTimestamp)
	}

	if result.TempDBTotalSizeMB != nil {
		s.mb.RecordSqlserverTempdbTotalSizeMbDataPoint(timestamp, *result.TempDBTotalSizeMB, sqlHostname, tempdbHealthStatus, collectionTimestamp)
	}

	return nil
}
