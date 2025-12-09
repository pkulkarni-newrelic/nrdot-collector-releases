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

func testTimestamp() pcommon.Timestamp {
	return pcommon.NewTimestampFromTime(time.Now())
}

func TestScrapePGAMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		PGAMetricsList: []models.PGAMetric{
			{InstID: "1", Name: "total PGA inuse", Value: 1024000},
			{InstID: "1", Name: "total PGA allocated", Value: 2048000},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapePGAMetrics(ctx, testTimestamp())
	require.Empty(t, errs)
}

func TestScrapePGAMetrics_QueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	mockClient := &client.MockClient{QueryErr: expectedErr}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapePGAMetrics(ctx, testTimestamp())
	require.Len(t, errs, 1)
}

func TestScrapeSGAMetrics_Success(t *testing.T) {
	mockClient := &client.MockClient{
		SGAMetricsList: []models.SGAMetric{
			{InstID: "1", Name: "Fixed Size", Value: sql.NullInt64{Int64: 8388608, Valid: true}},
		},
	}

	scraper := &CoreScraper{
		client:       mockClient,
		instanceName: "test-db",
		logger:       zap.NewNop(),
		mb:           metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type)),
	}

	ctx := context.Background()
	errs := scraper.scrapeSGAMetrics(ctx, testTimestamp())
	require.Empty(t, errs)
}
