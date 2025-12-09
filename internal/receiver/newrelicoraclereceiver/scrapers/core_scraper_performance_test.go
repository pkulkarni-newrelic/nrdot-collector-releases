// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

func testTS() pcommon.Timestamp {
	return pcommon.NewTimestampFromTime(time.Now())
}

// Test scrapeSysstatMetrics

func TestScrapeSysstatMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SysstatMetricsList: []models.SysstatMetric{
			{InstID: "1", Name: "redo buffer allocation retries", Value: sql.NullInt64{Int64: 100, Valid: true}},
			{InstID: "1", Name: "redo entries", Value: sql.NullInt64{Int64: 5000, Valid: true}},
			{InstID: "1", Name: "sorts (memory)", Value: sql.NullInt64{Int64: 1000, Valid: true}},
			{InstID: "1", Name: "sorts (disk)", Value: sql.NullInt64{Int64: 50, Valid: true}},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeSysstatMetrics(ctx, testTS())
	require.Empty(t, errs)
}

func TestScrapeSysstatMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeSysstatMetrics(ctx, testTS())
	require.Len(t, errs, 1)
}

func TestScrapeSysstatMetrics_NullValues(t *testing.T) {
	mockClient := &client.MockClient{
		SysstatMetricsList: []models.SysstatMetric{
			{InstID: "1", Name: "redo entries", Value: sql.NullInt64{Valid: false}},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeSysstatMetrics(ctx, testTS())
	require.Empty(t, errs)
}

// Test scrapeRollbackSegmentsMetrics

func TestScrapeRollbackSegmentsMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		RollbackSegmentsMetricsList: []models.RollbackSegmentsMetric{
			{
				Gets:   sql.NullInt64{Int64: 1000, Valid: true},
				Waits:  sql.NullInt64{Int64: 10, Valid: true},
				Ratio:  sql.NullFloat64{Float64: 0.01, Valid: true},
				InstID: "1",
			},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeRollbackSegmentsMetrics(ctx, testTS())
	require.Empty(t, errs)
}

func TestScrapeRollbackSegmentsMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeRollbackSegmentsMetrics(ctx, testTS())
	require.Len(t, errs, 1)
}

// Test scrapeRedoLogWaitsMetrics

func TestScrapeRedoLogWaitsMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		RedoLogWaitsMetricsList: []models.RedoLogWaitsMetric{
			{TotalWaits: sql.NullInt64{Int64: 100, Valid: true}, InstID: "1", Event: "log file parallel write"},
			{TotalWaits: sql.NullInt64{Int64: 50, Valid: true}, InstID: "1", Event: "log file switch completion"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeRedoLogWaitsMetrics(ctx, testTS())
	require.Empty(t, errs)
}

func TestScrapeRedoLogWaitsMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeRedoLogWaitsMetrics(ctx, testTS())
	require.Len(t, errs, 1)
}
