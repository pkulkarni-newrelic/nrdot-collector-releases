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

// Test constructor validation
func TestNewCoreScraper_ValidInputs(t *testing.T) {
	mockClient := client.NewMockClient()
	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)

	require.NoError(t, err)
	assert.NotNil(t, scraper)
	assert.Equal(t, "test-instance", scraper.instanceName)
}

func TestNewCoreScraper_NilClient(t *testing.T) {
	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(nil, mb, zap.NewNop(), "test-instance", config)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "client cannot be nil")
}

func TestNewCoreScraper_NilMetricsBuilder(t *testing.T) {
	mockClient := client.NewMockClient()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewCoreScraper(mockClient, nil, zap.NewNop(), "test-instance", config)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "metrics builder cannot be nil")
}

func TestNewCoreScraper_NilLogger(t *testing.T) {
	mockClient := client.NewMockClient()
	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, nil, "test-instance", config)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "logger cannot be nil")
}

func TestNewCoreScraper_EmptyInstanceName(t *testing.T) {
	mockClient := client.NewMockClient()
	config := metadata.DefaultMetricsBuilderConfig()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "", config)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "instance name cannot be empty")
}

// Test scrapeReadWriteMetrics with all metrics disabled
func TestScrapeReadWriteMetrics_AllMetricsDisabled(t *testing.T) {
	mockClient := client.NewMockClient()
	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = false
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = false
	config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled = false
	config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled = false
	config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled = false
	config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled = false
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics success case with all metrics enabled
func TestScrapeReadWriteMetrics_Success_AllMetricsEnabled(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              1,
			PhysicalReads:       1000,
			PhysicalWrites:      500,
			PhysicalBlockReads:  2000,
			PhysicalBlockWrites: 1500,
			ReadTime:            100,
			WriteTime:           50,
		},
		{
			InstID:              2,
			PhysicalReads:       800,
			PhysicalWrites:      400,
			PhysicalBlockReads:  1600,
			PhysicalBlockWrites: 1200,
			ReadTime:            80,
			WriteTime:           40,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled = true
	config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled = true
	config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics success case with partial metrics enabled
func TestScrapeReadWriteMetrics_Success_PartialMetricsEnabled(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              1,
			PhysicalReads:       1000,
			PhysicalWrites:      500,
			PhysicalBlockReads:  2000,
			PhysicalBlockWrites: 1500,
			ReadTime:            100,
			WriteTime:           50,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = false
	config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled = false
	config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled = true
	config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled = false
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics with query error
func TestScrapeReadWriteMetrics_QueryError(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("database connection failed")

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.NotNil(t, errors)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Error(), "database connection failed")
}

// Test scrapeReadWriteMetrics with empty result set
func TestScrapeReadWriteMetrics_EmptyResultSet(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics with string instance ID
func TestScrapeReadWriteMetrics_StringInstanceID(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              "inst-1",
			PhysicalReads:       1000,
			PhysicalWrites:      500,
			PhysicalBlockReads:  2000,
			PhysicalBlockWrites: 1500,
			ReadTime:            100,
			WriteTime:           50,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics with nil instance ID
func TestScrapeReadWriteMetrics_NilInstanceID(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              nil,
			PhysicalReads:       1000,
			PhysicalWrites:      500,
			PhysicalBlockReads:  2000,
			PhysicalBlockWrites: 1500,
			ReadTime:            100,
			WriteTime:           50,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics with zero values
func TestScrapeReadWriteMetrics_ZeroValues(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              1,
			PhysicalReads:       0,
			PhysicalWrites:      0,
			PhysicalBlockReads:  0,
			PhysicalBlockWrites: 0,
			ReadTime:            0,
			WriteTime:           0,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled = true
	config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled = true
	config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics with large values
func TestScrapeReadWriteMetrics_LargeValues(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              1,
			PhysicalReads:       9223372036854775807, // max int64
			PhysicalWrites:      9223372036854775807,
			PhysicalBlockReads:  9223372036854775807,
			PhysicalBlockWrites: 9223372036854775807,
			ReadTime:            9223372036854775807,
			WriteTime:           9223372036854775807,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled = true
	config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled = true
	config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}

// Test scrapeReadWriteMetrics with multiple instances
func TestScrapeReadWriteMetrics_MultipleInstances(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.DiskIOMetricsList = []models.DiskIOMetrics{
		{
			InstID:              1,
			PhysicalReads:       1000,
			PhysicalWrites:      500,
			PhysicalBlockReads:  2000,
			PhysicalBlockWrites: 1500,
			ReadTime:            100,
			WriteTime:           50,
		},
		{
			InstID:              2,
			PhysicalReads:       800,
			PhysicalWrites:      400,
			PhysicalBlockReads:  1600,
			PhysicalBlockWrites: 1200,
			ReadTime:            80,
			WriteTime:           40,
		},
		{
			InstID:              3,
			PhysicalReads:       1200,
			PhysicalWrites:      600,
			PhysicalBlockReads:  2400,
			PhysicalBlockWrites: 1800,
			ReadTime:            120,
			WriteTime:           60,
		},
	}

	config := metadata.DefaultMetricsBuilderConfig()
	config.Metrics.NewrelicoracledbDiskReads.Enabled = true
	config.Metrics.NewrelicoracledbDiskWrites.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled = true
	config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled = true
	config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled = true
	config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled = true
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(config, settings)

	scraper, err := NewCoreScraper(mockClient, mb, zap.NewNop(), "test-instance", config)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	errors := scraper.scrapeReadWriteMetrics(ctx, now)

	assert.Nil(t, errors)
}
