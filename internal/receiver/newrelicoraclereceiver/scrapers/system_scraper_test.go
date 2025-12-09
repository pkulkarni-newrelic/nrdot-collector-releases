// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

func TestNewSystemScraper(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	instanceName := "test-instance"
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewSystemScraper(mockClient, mb, logger, instanceName, config)

	assert.NotNil(t, scraper)
	assert.Equal(t, mockClient, scraper.client)
	assert.Equal(t, mb, scraper.mb)
	assert.Equal(t, logger, scraper.logger)
	assert.Equal(t, instanceName, scraper.instanceName)
	assert.Equal(t, config, scraper.metricsBuilderConfig)
}

func TestSystemScraper_NilDatabase(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotNil(t, scraper)
	assert.Nil(t, scraper.client)
}

func TestSystemScraper_NilMetricsBuilder(t *testing.T) {
	mockClient := &client.MockClient{}
	logger := zap.NewNop()

	scraper := NewSystemScraper(mockClient, nil, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotNil(t, scraper)
	assert.Nil(t, scraper.mb)
}

func TestSystemScraper_NilLogger(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)

	scraper := NewSystemScraper(mockClient, mb, nil, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotNil(t, scraper)
	assert.Nil(t, scraper.logger)
}

func TestSystemScraper_EmptyInstanceName(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(mockClient, mb, logger, "", metadata.DefaultMetricsBuilderConfig())

	assert.NotNil(t, scraper)
	assert.Equal(t, "", scraper.instanceName)
}

func TestSystemScraper_MetricsBuilderConfig(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewSystemScraper(mockClient, mb, logger, "test-instance", config)

	assert.NotNil(t, scraper)
	assert.Equal(t, config, scraper.metricsBuilderConfig)
}

func TestSystemScraper_MultipleInstances(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper1 := NewSystemScraper(mockClient, mb, logger, "instance-1", metadata.DefaultMetricsBuilderConfig())
	scraper2 := NewSystemScraper(mockClient, mb, logger, "instance-2", metadata.DefaultMetricsBuilderConfig())

	assert.NotEqual(t, scraper1, scraper2)
	assert.Equal(t, "instance-1", scraper1.instanceName)
	assert.Equal(t, "instance-2", scraper2.instanceName)
}

func TestSystemScraper_RecordMetric_BufferCacheMetrics(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "Buffer Cache Hit Ratio", 95.5, "1")
	})
}

func TestSystemScraper_RecordMetric_TransactionMetrics(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "User Transaction Per Sec", 100.0, "1")
		scraper.recordMetric(0, "User Commits Per Sec", 50.0, "1")
		scraper.recordMetric(0, "User Rollbacks Per Sec", 5.0, "1")
	})
}

func TestSystemScraper_RecordMetric_IOMetrics(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "Physical Reads Per Sec", 1000.0, "1")
		scraper.recordMetric(0, "Physical Writes Per Sec", 500.0, "1")
		scraper.recordMetric(0, "Logical Reads Per Sec", 5000.0, "1")
	})
}

func TestSystemScraper_RecordMetric_ParseMetrics(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "Total Parse Count Per Sec", 200.0, "1")
		scraper.recordMetric(0, "Hard Parse Count Per Sec", 10.0, "1")
		scraper.recordMetric(0, "Soft Parse Ratio", 95.0, "1")
	})
}

func TestSystemScraper_RecordMetric_CPUMetrics(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "CPU Usage Per Sec", 75.5, "1")
		scraper.recordMetric(0, "Host CPU Utilization (%)", 60.0, "1")
		scraper.recordMetric(0, "Background CPU Usage Per Sec", 10.0, "1")
	})
}

func TestSystemScraper_RecordMetric_SessionMetrics(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "Current Logons Count", 50.0, "1")
		scraper.recordMetric(0, "Session Count", 100.0, "1")
		scraper.recordMetric(0, "Average Active Sessions", 25.0, "1")
	})
}

func TestSystemScraper_RecordMetric_UnknownMetric(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	scraper := NewSystemScraper(nil, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.NotPanics(t, func() {
		scraper.recordMetric(0, "Unknown Metric Name", 42.0, "1")
	})
}
