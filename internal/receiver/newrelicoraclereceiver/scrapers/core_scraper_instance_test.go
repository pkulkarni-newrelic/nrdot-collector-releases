// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

// Test scrapeLockedAccountsMetrics
func TestScrapeLockedAccountsMetrics_Disabled(t *testing.T) {
	mockClient := client.NewMockClient()
	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbLockedAccounts.Enabled = false
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLockedAccountsMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLockedAccountsMetrics_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LockedAccountsList = []models.LockedAccountsMetric{
		{
			InstID:         1,
			LockedAccounts: 5,
		},
		{
			InstID:         2,
			LockedAccounts: 3,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbLockedAccounts.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLockedAccountsMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLockedAccountsMetrics_QueryError(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("database connection failed")

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbLockedAccounts.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLockedAccountsMetrics(ctx, now)

	assert.NotNil(t, errors)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Error(), "database connection failed")
}

func TestScrapeLockedAccountsMetrics_EmptyResult(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LockedAccountsList = []models.LockedAccountsMetric{}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbLockedAccounts.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLockedAccountsMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLockedAccountsMetrics_ZeroAccounts(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LockedAccountsList = []models.LockedAccountsMetric{
		{
			InstID:         1,
			LockedAccounts: 0,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbLockedAccounts.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLockedAccountsMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeGlobalNameInstanceMetrics
func TestScrapeGlobalNameInstanceMetrics_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.GlobalNameList = []models.GlobalNameMetric{
		{
			InstID:     1,
			GlobalName: "PROD.EXAMPLE.COM",
		},
		{
			InstID:     2,
			GlobalName: "TEST.EXAMPLE.COM",
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeGlobalNameInstanceMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeGlobalNameInstanceMetrics_QueryError(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("query execution failed")

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeGlobalNameInstanceMetrics(ctx, now)

	assert.NotNil(t, errors)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Error(), "query execution failed")
}

func TestScrapeGlobalNameInstanceMetrics_EmptyResult(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.GlobalNameList = []models.GlobalNameMetric{}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeGlobalNameInstanceMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeGlobalNameInstanceMetrics_StringInstanceID(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.GlobalNameList = []models.GlobalNameMetric{
		{
			InstID:     "inst-1",
			GlobalName: "PROD.EXAMPLE.COM",
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeGlobalNameInstanceMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeDBIDInstanceMetrics
func TestScrapeDBIDInstanceMetrics_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DBIDList = []models.DBIDMetric{
		{
			InstID: 1,
			DBID:   "1234567890",
		},
		{
			InstID: 2,
			DBID:   "0987654321",
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeDBIDInstanceMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeDBIDInstanceMetrics_QueryError(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("network timeout")

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeDBIDInstanceMetrics(ctx, now)

	assert.NotNil(t, errors)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Error(), "network timeout")
}

func TestScrapeDBIDInstanceMetrics_EmptyResult(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DBIDList = []models.DBIDMetric{}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeDBIDInstanceMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeDBIDInstanceMetrics_NilInstanceID(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DBIDList = []models.DBIDMetric{
		{
			InstID: nil,
			DBID:   "1234567890",
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeDBIDInstanceMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeLongRunningQueriesMetrics
func TestScrapeLongRunningQueriesMetrics_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LongRunningQueriesList = []models.LongRunningQueriesMetric{
		{
			InstID: 1,
			Total:  10,
		},
		{
			InstID: 2,
			Total:  5,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLongRunningQueriesMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLongRunningQueriesMetrics_QueryError(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("permission denied")

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLongRunningQueriesMetrics(ctx, now)

	assert.NotNil(t, errors)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Error(), "permission denied")
}

func TestScrapeLongRunningQueriesMetrics_EmptyResult(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LongRunningQueriesList = []models.LongRunningQueriesMetric{}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLongRunningQueriesMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLongRunningQueriesMetrics_ZeroQueries(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LongRunningQueriesList = []models.LongRunningQueriesMetric{
		{
			InstID: 1,
			Total:  0,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLongRunningQueriesMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLongRunningQueriesMetrics_LargeValue(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LongRunningQueriesList = []models.LongRunningQueriesMetric{
		{
			InstID: 1,
			Total:  9223372036854775807, // max int64
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLongRunningQueriesMetrics(ctx, now)

	assert.Nil(t, errors)
}

func TestScrapeLongRunningQueriesMetrics_MultipleInstances(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.LongRunningQueriesList = []models.LongRunningQueriesMetric{
		{
			InstID: 1,
			Total:  10,
		},
		{
			InstID: 2,
			Total:  5,
		},
		{
			InstID: 3,
			Total:  15,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeLongRunningQueriesMetrics(ctx, now)

	assert.Nil(t, errors)
}
