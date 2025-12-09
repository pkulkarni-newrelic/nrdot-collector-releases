package scrapers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
)

func TestLockScraper(t *testing.T) {
	mockClient := &client.MockClient{
		LockCountsList: []models.LockCount{
			{
				LockType:  sql.NullString{String: "Row", Valid: true},
				LockMode:  sql.NullString{String: "X", Valid: true},
				LockCount: sql.NullInt64{Int64: 5, Valid: true},
			},
			{
				LockType:  sql.NullString{String: "Table", Valid: true},
				LockMode:  sql.NullString{String: "S", Valid: true},
				LockCount: sql.NullInt64{Int64: 3, Valid: true},
			},
		},
		LockSessionCountsList: []models.LockSessionCount{
			{
				LockType:     sql.NullString{String: "Row", Valid: true},
				SessionCount: sql.NullInt64{Int64: 2, Valid: true},
			},
			{
				LockType:     sql.NullString{String: "Table", Valid: true},
				SessionCount: sql.NullInt64{Int64: 1, Valid: true},
			},
		},
		LockedObjectCountsList: []models.LockedObjectCount{
			{
				LockType:    sql.NullString{String: "Row", Valid: true},
				ObjectType:  sql.NullString{String: "TABLE", Valid: true},
				ObjectCount: sql.NullInt64{Int64: 2, Valid: true},
			},
			{
				LockType:    sql.NullString{String: "Table", Valid: true},
				ObjectType:  sql.NullString{String: "INDEX", Valid: true},
				ObjectCount: sql.NullInt64{Int64: 1, Valid: true},
			},
		},
	}

	logger := zap.NewNop()
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type))

	lockScraper := NewLockScraper(logger, mockClient, mb, "test_instance")
	require.NotNil(t, lockScraper)

	ctx := context.Background()
	errs := lockScraper.ScrapeLocks(ctx)
	require.Len(t, errs, 0)

	// Verify metrics were generated
	metrics := mb.Emit()
	assert.Greater(t, metrics.ResourceMetrics().Len(), 0)

	// Check that we have lock metrics
	resourceMetrics := metrics.ResourceMetrics().At(0)
	scopeMetrics := resourceMetrics.ScopeMetrics().At(0)
	assert.Greater(t, scopeMetrics.Metrics().Len(), 0)
}

func TestLockScraperWithError(t *testing.T) {
	mockClient := &client.MockClient{
		QueryErr: assert.AnError,
	}

	logger := zap.NewNop()
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopSettings(metadata.Type))

	lockScraper := NewLockScraper(logger, mockClient, mb, "test_instance")
	require.NotNil(t, lockScraper)

	ctx := context.Background()
	errs := lockScraper.ScrapeLocks(ctx)
	require.Len(t, errs, 3) // Should have 3 errors (one for each query type)
}
