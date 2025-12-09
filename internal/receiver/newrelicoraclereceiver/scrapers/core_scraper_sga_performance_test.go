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

func testSGATimestamp() pcommon.Timestamp {
	return pcommon.NewTimestampFromTime(time.Now())
}

// Test scrapeSGASharedPoolLibraryCacheReloadRatioMetrics

func TestScrapeSGASharedPoolLibraryCacheReloadRatioMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGASharedPoolLibraryCacheReloadRatioList: []models.SGASharedPoolLibraryCacheReloadRatioMetric{
			{Ratio: 0.05, InstID: "1"},
			{Ratio: 0.03, InstID: "2"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGASharedPoolLibraryCacheReloadRatioMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGASharedPoolLibraryCacheReloadRatioMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGASharedPoolLibraryCacheReloadRatioMetrics(context.Background(), testSGATimestamp())
	require.Len(t, errs, 1)
}

// Test scrapeSGASharedPoolLibraryCacheHitRatioMetrics

func TestScrapeSGASharedPoolLibraryCacheHitRatioMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGASharedPoolLibraryCacheHitRatioList: []models.SGASharedPoolLibraryCacheHitRatioMetric{
			{Ratio: 0.95, InstID: "1"},
			{Ratio: 0.97, InstID: "2"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGASharedPoolLibraryCacheHitRatioMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGASharedPoolLibraryCacheHitRatioMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGASharedPoolLibraryCacheHitRatioMetrics(context.Background(), testSGATimestamp())
	require.Len(t, errs, 1)
}

// Test scrapeSGASharedPoolDictCacheMissRatioMetrics

func TestScrapeSGASharedPoolDictCacheMissRatioMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGASharedPoolDictCacheMissRatioList: []models.SGASharedPoolDictCacheMissRatioMetric{
			{Ratio: 0.02, InstID: "1"},
			{Ratio: 0.01, InstID: "2"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGASharedPoolDictCacheMissRatioMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGASharedPoolDictCacheMissRatioMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGASharedPoolDictCacheMissRatioMetrics(context.Background(), testSGATimestamp())
	require.Len(t, errs, 1)
}

// Test scrapeSGALogBufferSpaceWaitsMetrics

func TestScrapeSGALogBufferSpaceWaitsMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGALogBufferSpaceWaitsList: []models.SGALogBufferSpaceWaitsMetric{
			{Count: 5, InstID: "1"},
			{Count: 3, InstID: "2"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGALogBufferSpaceWaitsMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGALogBufferSpaceWaitsMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGALogBufferSpaceWaitsMetrics(context.Background(), testSGATimestamp())
	require.Len(t, errs, 1)
}

// Test scrapeSGALogAllocRetriesMetrics

func TestScrapeSGALogAllocRetriesMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGALogAllocRetriesList: []models.SGALogAllocRetriesMetric{
			{Ratio: sql.NullFloat64{Float64: 0.04, Valid: true}, InstID: "1"},
			{Ratio: sql.NullFloat64{Float64: 0.02, Valid: true}, InstID: "2"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGALogAllocRetriesMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGALogAllocRetriesMetrics_NullRatio(t *testing.T) {
	mockClient := &client.MockClient{
		SGALogAllocRetriesList: []models.SGALogAllocRetriesMetric{
			{Ratio: sql.NullFloat64{Valid: false}, InstID: "1"},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGALogAllocRetriesMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGALogAllocRetriesMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGALogAllocRetriesMetrics(context.Background(), testSGATimestamp())
	require.Len(t, errs, 1)
}

// Test scrapeSGAHitRatioMetrics

func TestScrapeSGAHitRatioMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGAHitRatioList: []models.SGAHitRatioMetric{
			{InstID: "1", Ratio: sql.NullFloat64{Float64: 0.92, Valid: true}},
			{InstID: "2", Ratio: sql.NullFloat64{Float64: 0.94, Valid: true}},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGAHitRatioMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGAHitRatioMetrics_NullRatio(t *testing.T) {
	mockClient := &client.MockClient{
		SGAHitRatioList: []models.SGAHitRatioMetric{
			{InstID: "1", Ratio: sql.NullFloat64{Valid: false}},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGAHitRatioMetrics(context.Background(), testSGATimestamp())
	require.Empty(t, errs)
}

func TestScrapeSGAHitRatioMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	errs := scraper.scrapeSGAHitRatioMetrics(context.Background(), testSGATimestamp())
	require.Len(t, errs, 1)
}
