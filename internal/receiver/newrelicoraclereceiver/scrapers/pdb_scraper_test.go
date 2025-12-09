// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

func TestNewPdbScraper(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	instanceName := "test-instance"
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewPdbScraper(mockClient, mb, logger, instanceName, config)

	assert.NotNil(t, scraper)
	assert.Equal(t, mockClient, scraper.client)
	assert.Equal(t, mb, scraper.mb)
	assert.Equal(t, logger, scraper.logger)
	assert.Equal(t, instanceName, scraper.instanceName)
	assert.Equal(t, config, scraper.config)
	assert.Nil(t, scraper.isCDBCapable)
	assert.False(t, scraper.environmentChecked)
}

func TestIsCDBSupported_True(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	cdbCapable := true
	scraper.isCDBCapable = &cdbCapable

	assert.True(t, scraper.isCDBSupported())
}

func TestIsCDBSupported_False(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	cdbCapable := false
	scraper.isCDBCapable = &cdbCapable

	assert.False(t, scraper.isCDBSupported())
}

func TestIsCDBSupported_Nil(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	assert.False(t, scraper.isCDBSupported())
}

func TestContainsORACode_ORA00942(t *testing.T) {
	err := errors.New("ORA-00942: table or view does not exist")
	assert.True(t, containsORACode(err, "ORA-00942"))
}

func TestContainsORACode_ORA01722(t *testing.T) {
	err := errors.New("ORA-01722: invalid number")
	assert.True(t, containsORACode(err, "ORA-01722"))
}

func TestContainsORACode_MultipleCodes(t *testing.T) {
	err := errors.New("ORA-01722: invalid number")
	assert.True(t, containsORACode(err, "ORA-00942", "ORA-01722"))
}

func TestContainsORACode_NoMatch(t *testing.T) {
	err := errors.New("network timeout")
	assert.False(t, containsORACode(err, "ORA-00942"))
}

func TestContainsORACode_NilError(t *testing.T) {
	assert.False(t, containsORACode(nil, "ORA-00942"))
}

func TestRecordMetric_SessionMetrics(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	testCases := []string{
		"Active Parallel Sessions",
		"Active Serial Sessions",
		"Average Active Sessions",
		"Session Count",
	}

	for _, metricName := range testCases {
		scraper.recordMetric(0, metricName, 100.0, "1", "TEST_PDB")
	}
}

func TestRecordMetric_CPUMetrics(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	testCases := []string{
		"Background CPU Usage Per Sec",
		"CPU Usage Per Sec",
		"CPU Usage Per Txn",
		"Database CPU Time Ratio",
	}

	for _, metricName := range testCases {
		scraper.recordMetric(0, metricName, 50.5, "1", "TEST_PDB")
	}
}

func TestRecordMetric_IOMetrics(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	testCases := []string{
		"Physical Read Total Bytes Per Sec",
		"Physical Reads Per Sec",
		"Physical Write Total Bytes Per Sec",
		"Physical Writes Per Sec",
	}

	for _, metricName := range testCases {
		scraper.recordMetric(0, metricName, 1024.0, "1", "TEST_PDB")
	}
}

func TestRecordMetric_ParseMetrics(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	testCases := []string{
		"Hard Parse Count Per Sec",
		"Soft Parse Ratio",
		"Total Parse Count Per Sec",
	}

	for _, metricName := range testCases {
		scraper.recordMetric(0, metricName, 75.0, "1", "TEST_PDB")
	}
}

func TestRecordMetric_TransactionMetrics(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	testCases := []string{
		"User Transaction Per Sec",
		"User Commits Per Sec",
		"User Rollbacks Per Sec",
	}

	for _, metricName := range testCases {
		scraper.recordMetric(0, metricName, 10.0, "1", "TEST_PDB")
	}
}

func TestRecordMetric_UnknownMetric(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	scraper.recordMetric(0, "Unknown Metric Name", 100.0, "1", "TEST_PDB")
}

func TestPdbScraper_CDBCapableStates(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	t.Run("uninitialized", func(t *testing.T) {
		scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())
		assert.False(t, scraper.isCDBSupported())
		assert.Nil(t, scraper.isCDBCapable)
	})

	t.Run("capable_true", func(t *testing.T) {
		scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())
		capable := true
		scraper.isCDBCapable = &capable
		assert.True(t, scraper.isCDBSupported())
	})

	t.Run("capable_false", func(t *testing.T) {
		scraper := NewPdbScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())
		capable := false
		scraper.isCDBCapable = &capable
		assert.False(t, scraper.isCDBSupported())
	})
}
