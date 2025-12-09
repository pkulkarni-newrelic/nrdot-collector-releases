// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

func TestNewRacScraper(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	instanceName := "test-instance"
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewRacScraper(mockClient, mb, logger, instanceName, config)

	assert.NotNil(t, scraper)
	assert.Equal(t, mockClient, scraper.client)
	assert.Equal(t, mb, scraper.mb)
	assert.Equal(t, logger, scraper.logger)
	assert.Equal(t, instanceName, scraper.instanceName)
	assert.Equal(t, config, scraper.metricsBuilderConfig)
	assert.Nil(t, scraper.isRacMode)
}

func TestNullStringToString_Valid(t *testing.T) {
	ns := sql.NullString{String: "test", Valid: true}
	result := nullStringToString(ns)
	assert.Equal(t, "test", result)
}

func TestNullStringToString_Invalid(t *testing.T) {
	ns := sql.NullString{String: "", Valid: false}
	result := nullStringToString(ns)
	assert.Equal(t, "", result)
}

func TestStringStatusToBinary_Match(t *testing.T) {
	result := stringStatusToBinary("OPEN", "OPEN")
	assert.Equal(t, int64(1), result)
}

func TestStringStatusToBinary_MatchCaseInsensitive(t *testing.T) {
	result := stringStatusToBinary("open", "OPEN")
	assert.Equal(t, int64(1), result)
}

func TestStringStatusToBinary_NoMatch(t *testing.T) {
	result := stringStatusToBinary("CLOSED", "OPEN")
	assert.Equal(t, int64(0), result)
}

func TestStringStatusToBinary_EmptyString(t *testing.T) {
	result := stringStatusToBinary("", "OPEN")
	assert.Equal(t, int64(0), result)
}

func TestRacScraper_CachedRacMode(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()

	t.Run("uninitialized", func(t *testing.T) {
		scraper := NewRacScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())
		assert.Nil(t, scraper.isRacMode)
	})

	t.Run("cached_enabled", func(t *testing.T) {
		scraper := NewRacScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())
		racEnabled := true
		scraper.isRacMode = &racEnabled
		assert.NotNil(t, scraper.isRacMode)
		assert.True(t, *scraper.isRacMode)
	})

	t.Run("cached_disabled", func(t *testing.T) {
		scraper := NewRacScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())
		racDisabled := false
		scraper.isRacMode = &racDisabled
		assert.NotNil(t, scraper.isRacMode)
		assert.False(t, *scraper.isRacMode)
	})
}

func TestRacScraper_MetricConfiguration(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	t.Run("default_config", func(t *testing.T) {
		scraper := NewRacScraper(mockClient, mb, logger, "test-instance", config)
		assert.NotNil(t, scraper.metricsBuilderConfig)
	})

	t.Run("custom_config", func(t *testing.T) {
		customConfig := metadata.DefaultMetricsBuilderConfig()
		customConfig.Metrics.NewrelicoracledbAsmDiskgroupTotalMb.Enabled = false
		scraper := NewRacScraper(mockClient, mb, logger, "test-instance", customConfig)
		assert.False(t, scraper.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupTotalMb.Enabled)
	})
}

func TestRacScraper_ClientAssignment(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewRacScraper(mockClient, mb, logger, "test-instance", config)
	assert.NotNil(t, scraper.client)
	assert.Equal(t, mockClient, scraper.client)
}

func TestRacScraper_MultipleScraper(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb1 := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	mb2 := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper1 := NewRacScraper(mockClient, mb1, logger, "instance-1", config)
	scraper2 := NewRacScraper(mockClient, mb2, logger, "instance-2", config)

	assert.Equal(t, "instance-1", scraper1.instanceName)
	assert.Equal(t, "instance-2", scraper2.instanceName)
	assert.NotEqual(t, scraper1.mb, scraper2.mb)
}

func TestRacScraper_ConcurrentAccess(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	scraper := NewRacScraper(mockClient, mb, logger, "test-instance", metadata.DefaultMetricsBuilderConfig())

	done := make(chan bool)
	racEnabled := true

	go func() {
		scraper.racModeMutex.Lock()
		scraper.isRacMode = &racEnabled
		scraper.racModeMutex.Unlock()
		done <- true
	}()

	go func() {
		scraper.racModeMutex.RLock()
		_ = scraper.isRacMode
		scraper.racModeMutex.RUnlock()
		done <- true
	}()

	<-done
	<-done
}

func TestRacScraper_EmptyInstanceName(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewRacScraper(mockClient, mb, logger, "", config)
	assert.Equal(t, "", scraper.instanceName)
}

func TestRacScraper_LoggerConfiguration(t *testing.T) {
	mockClient := &client.MockClient{}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper := NewRacScraper(mockClient, mb, logger, "test-instance", config)
	assert.NotNil(t, scraper.logger)
	assert.Equal(t, logger, scraper.logger)
}
