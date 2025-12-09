// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

func TestNewConnectionScraper(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	instanceName := "test-instance"
	config := metadata.DefaultMetricsBuilderConfig()

	t.Run("valid scraper creation", func(t *testing.T) {
		scraper, err := NewConnectionScraper(mockClient, mb, logger, instanceName, config)

		require.NoError(t, err)
		assert.NotNil(t, scraper)
		assert.Equal(t, mockClient, scraper.client)
		assert.Equal(t, mb, scraper.mb)
		assert.Equal(t, logger, scraper.logger)
		assert.Equal(t, instanceName, scraper.instanceName)
		assert.Equal(t, config, scraper.metricsBuilderConfig)
	})

	t.Run("nil client", func(t *testing.T) {
		scraper, err := NewConnectionScraper(nil, mb, logger, instanceName, config)

		assert.Error(t, err)
		assert.Nil(t, scraper)
		assert.Contains(t, err.Error(), "client cannot be nil")
	})

	t.Run("nil metrics builder", func(t *testing.T) {
		scraper, err := NewConnectionScraper(mockClient, nil, logger, instanceName, config)

		assert.Error(t, err)
		assert.Nil(t, scraper)
		assert.Contains(t, err.Error(), "metrics builder cannot be nil")
	})

	t.Run("nil logger", func(t *testing.T) {
		scraper, err := NewConnectionScraper(mockClient, mb, nil, instanceName, config)

		assert.Error(t, err)
		assert.Nil(t, scraper)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})

	t.Run("empty instance name", func(t *testing.T) {
		scraper, err := NewConnectionScraper(mockClient, mb, logger, "", config)

		assert.Error(t, err)
		assert.Nil(t, scraper)
		assert.Contains(t, err.Error(), "instance name cannot be empty")
	})
}

func TestConnectionScraper_CoreConnectionCounts(t *testing.T) {
	t.Run("valid session counts", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.TotalSessions = 150
		mockClient.ActiveSessionCount = 85
		mockClient.InactiveSessions = 65

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("query error on total sessions", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.QueryErr = errors.New("total sessions query failed")

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.NotEmpty(t, errs)
		assert.Contains(t, errs[0].Error(), "total sessions query failed")
	})

	t.Run("zero session counts", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.TotalSessions = 0
		mockClient.ActiveSessionCount = 0
		mockClient.InactiveSessions = 0

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_SessionBreakdown(t *testing.T) {
	t.Run("valid session status and types", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.SessionStatusList = []models.SessionStatus{
			{
				Status: sql.NullString{String: "ACTIVE", Valid: true},
				Count:  sql.NullInt64{Int64: 50, Valid: true},
			},
			{
				Status: sql.NullString{String: "INACTIVE", Valid: true},
				Count:  sql.NullInt64{Int64: 30, Valid: true},
			},
		}
		mockClient.SessionTypeList = []models.SessionType{
			{
				Type:  sql.NullString{String: "USER", Valid: true},
				Count: sql.NullInt64{Int64: 60, Valid: true},
			},
			{
				Type:  sql.NullString{String: "BACKGROUND", Valid: true},
				Count: sql.NullInt64{Int64: 20, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty session breakdown", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.SessionStatusList = []models.SessionStatus{}
		mockClient.SessionTypeList = []models.SessionType{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("invalid session status data", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.SessionStatusList = []models.SessionStatus{
			{
				Status: sql.NullString{Valid: false},
				Count:  sql.NullInt64{Int64: 50, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs) // Invalid data is silently skipped
	})
}

func TestConnectionScraper_LogonStats(t *testing.T) {
	t.Run("valid logon statistics", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.LogonStatsList = []models.LogonStat{
			{
				Name:  sql.NullString{String: "logons cumulative", Valid: true},
				Value: sql.NullFloat64{Float64: 1500.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "logons current", Valid: true},
				Value: sql.NullFloat64{Float64: 85.0, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty logon stats", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.LogonStatsList = []models.LogonStat{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_SessionResourceConsumption(t *testing.T) {
	t.Run("valid session resources", func(t *testing.T) {
		mockClient := client.NewMockClient()
		logonTime := time.Date(2024, 10, 28, 10, 0, 0, 0, time.UTC)
		mockClient.SessionResourcesList = []models.SessionResource{
			{
				SID:             sql.NullInt64{Int64: 123, Valid: true},
				Username:        sql.NullString{String: "TESTUSER", Valid: true},
				Status:          sql.NullString{String: "ACTIVE", Valid: true},
				Program:         sql.NullString{String: "sqlplus.exe", Valid: true},
				Machine:         sql.NullString{String: "WORKSTATION1", Valid: true},
				OSUser:          sql.NullString{String: "john.doe", Valid: true},
				LogonTime:       sql.NullTime{Time: logonTime, Valid: true},
				LastCallET:      sql.NullInt64{Int64: 300, Valid: true},
				CPUUsageSeconds: sql.NullFloat64{Float64: 45.5, Valid: true},
				PGAMemoryBytes:  sql.NullInt64{Int64: 2048000, Valid: true},
				LogicalReads:    sql.NullInt64{Int64: 15000, Valid: true},
			},
			{
				SID:             sql.NullInt64{Int64: 456, Valid: true},
				Username:        sql.NullString{String: "APPUSER", Valid: true},
				Status:          sql.NullString{String: "INACTIVE", Valid: true},
				LastCallET:      sql.NullInt64{Int64: 600, Valid: true},
				CPUUsageSeconds: sql.NullFloat64{Float64: 12.3, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty session resources", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.SessionResourcesList = []models.SessionResource{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_WaitEvents(t *testing.T) {
	t.Run("valid current wait events", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.CurrentWaitEventsList = []models.CurrentWaitEvent{
			{
				SID:           sql.NullInt64{Int64: 123, Valid: true},
				Username:      sql.NullString{String: "TESTUSER", Valid: true},
				Event:         sql.NullString{String: "db file sequential read", Valid: true},
				WaitTime:      sql.NullInt64{Int64: 25500, Valid: true},
				State:         sql.NullString{String: "WAITING", Valid: true},
				SecondsInWait: sql.NullInt64{Int64: 25, Valid: true},
				WaitClass:     sql.NullString{String: "User I/O", Valid: true},
			},
			{
				SID:           sql.NullInt64{Int64: 456, Valid: true},
				Username:      sql.NullString{String: "APPUSER", Valid: true},
				Event:         sql.NullString{String: "log file sync", Valid: true},
				WaitTime:      sql.NullInt64{Int64: 10200, Valid: true},
				State:         sql.NullString{String: "WAITING", Valid: true},
				SecondsInWait: sql.NullInt64{Int64: 10, Valid: true},
				WaitClass:     sql.NullString{String: "Commit", Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty wait events", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.CurrentWaitEventsList = []models.CurrentWaitEvent{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_BlockingSessions(t *testing.T) {
	t.Run("valid blocking sessions", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.BlockingSessionsList = []models.BlockingSession{
			{
				SID:             sql.NullInt64{Int64: 456, Valid: true},
				Serial:          sql.NullInt64{Int64: 1234, Valid: true},
				BlockingSession: sql.NullInt64{Int64: 123, Valid: true},
				Event:           sql.NullString{String: "enq: TX - row lock contention", Valid: true},
				Username:        sql.NullString{String: "USER2", Valid: true},
				Program:         sql.NullString{String: "myapp.exe", Valid: true},
				SecondsInWait:   sql.NullInt64{Int64: 5000, Valid: true},
			},
			{
				SID:             sql.NullInt64{Int64: 111, Valid: true},
				Serial:          sql.NullInt64{Int64: 5678, Valid: true},
				BlockingSession: sql.NullInt64{Int64: 789, Valid: true},
				Event:           sql.NullString{String: "enq: TX - row lock contention", Valid: true},
				Username:        sql.NullString{String: "USER4", Valid: true},
				Program:         sql.NullString{String: "otherapp.exe", Valid: true},
				SecondsInWait:   sql.NullInt64{Int64: 3000, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty blocking sessions", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.BlockingSessionsList = []models.BlockingSession{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_WaitEventSummary(t *testing.T) {
	t.Run("valid wait event summary", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.WaitEventSummaryList = []models.WaitEventSummary{
			{
				Event:            sql.NullString{String: "db file sequential read", Valid: true},
				TotalWaits:       sql.NullInt64{Int64: 15000, Valid: true},
				TimeWaitedMicro:  sql.NullInt64{Int64: 30000000, Valid: true},
				AverageWaitMicro: sql.NullFloat64{Float64: 2000.0, Valid: true},
				WaitClass:        sql.NullString{String: "User I/O", Valid: true},
			},
			{
				Event:            sql.NullString{String: "log file sync", Valid: true},
				TotalWaits:       sql.NullInt64{Int64: 8000, Valid: true},
				TimeWaitedMicro:  sql.NullInt64{Int64: 12000000, Valid: true},
				AverageWaitMicro: sql.NullFloat64{Float64: 1500.0, Valid: true},
				WaitClass:        sql.NullString{String: "Commit", Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty wait event summary", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.WaitEventSummaryList = []models.WaitEventSummary{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_ConnectionPoolMetrics(t *testing.T) {
	t.Run("valid pool metrics", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.ConnectionPoolMetricsList = []models.ConnectionPoolMetric{
			{
				MetricName: sql.NullString{String: "shared_servers", Valid: true},
				Value:      sql.NullInt64{Int64: 10, Valid: true},
			},
			{
				MetricName: sql.NullString{String: "dispatchers", Valid: true},
				Value:      sql.NullInt64{Int64: 5, Valid: true},
			},
			{
				MetricName: sql.NullString{String: "circuits", Valid: true},
				Value:      sql.NullInt64{Int64: 20, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty pool metrics", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.ConnectionPoolMetricsList = []models.ConnectionPoolMetric{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_SessionLimits(t *testing.T) {
	t.Run("valid session limits", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.SessionLimitsList = []models.SessionLimit{
			{
				ResourceName:       sql.NullString{String: "sessions", Valid: true},
				CurrentUtilization: sql.NullInt64{Int64: 150, Valid: true},
				MaxUtilization:     sql.NullInt64{Int64: 180, Valid: true},
				InitialAllocation:  sql.NullString{String: "100", Valid: true},
				LimitValue:         sql.NullString{String: "300", Valid: true},
			},
			{
				ResourceName:       sql.NullString{String: "processes", Valid: true},
				CurrentUtilization: sql.NullInt64{Int64: 100, Valid: true},
				MaxUtilization:     sql.NullInt64{Int64: 120, Valid: true},
				InitialAllocation:  sql.NullString{String: "50", Valid: true},
				LimitValue:         sql.NullString{String: "UNLIMITED", Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty session limits", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.SessionLimitsList = []models.SessionLimit{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}

func TestConnectionScraper_ConnectionQuality(t *testing.T) {
	t.Run("valid connection quality metrics", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.ConnectionQualityMetricsList = []models.ConnectionQualityMetric{
			{
				Name:  sql.NullString{String: "user commits", Valid: true},
				Value: sql.NullFloat64{Float64: 5000.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "user rollbacks", Valid: true},
				Value: sql.NullFloat64{Float64: 200.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "parse count (total)", Valid: true},
				Value: sql.NullFloat64{Float64: 8000.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "parse count (hard)", Valid: true},
				Value: sql.NullFloat64{Float64: 1000.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "execute count", Valid: true},
				Value: sql.NullFloat64{Float64: 50000.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "SQL*Net roundtrips to/from client", Valid: true},
				Value: sql.NullFloat64{Float64: 30000.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "bytes sent via SQL*Net to client", Valid: true},
				Value: sql.NullFloat64{Float64: 1048576000.0, Valid: true},
			},
			{
				Name:  sql.NullString{String: "bytes received via SQL*Net from client", Valid: true},
				Value: sql.NullFloat64{Float64: 524288000.0, Valid: true},
			},
		}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})

	t.Run("empty connection quality metrics", func(t *testing.T) {
		mockClient := client.NewMockClient()
		mockClient.ConnectionQualityMetricsList = []models.ConnectionQualityMetric{}

		config := metadata.DefaultMetricsBuilderConfig()
		settings := receivertest.NewNopSettings(metadata.Type)
		mb := metadata.NewMetricsBuilder(config, settings)
		logger := zap.NewNop()

		scraper, err := NewConnectionScraper(mockClient, mb, logger, "test-instance", config)
		require.NoError(t, err)

		errs := scraper.ScrapeConnectionMetrics(context.Background())
		assert.Empty(t, errs)
	})
}
