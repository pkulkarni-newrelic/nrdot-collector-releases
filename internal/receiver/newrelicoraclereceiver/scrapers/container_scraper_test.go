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
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

func TestNewContainerScraper_ValidInputs(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, scraper)
	assert.Equal(t, "test-instance", scraper.instanceName)
	assert.Equal(t, mockClient, scraper.client)
	assert.Equal(t, mb, scraper.mb)
	assert.Equal(t, logger, scraper.logger)
	assert.False(t, scraper.environmentChecked)
	assert.Nil(t, scraper.isCDBCapable)
	assert.Nil(t, scraper.isPDBCapable)
}

func TestNewContainerScraper_NilClient(t *testing.T) {
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(nil, mb, logger, "test-instance", config, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "client cannot be nil")
}

func TestNewContainerScraper_NilMetricsBuilder(t *testing.T) {
	mockClient := client.NewMockClient()
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, nil, logger, "test-instance", config, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "metrics builder cannot be nil")
}

func TestNewContainerScraper_NilLogger(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, nil, "test-instance", config, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "logger cannot be nil")
}

func TestNewContainerScraper_EmptyInstanceName(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "", config, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "instance name cannot be empty")
}

func TestContainerScraper_CheckEnvironmentCapability_CDBCapable(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CheckCDBFeatureResult = 1
	mockClient.CheckPDBCapabilityResult = 1
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = scraper.checkEnvironmentCapability(ctx)

	assert.NoError(t, err)
	assert.True(t, scraper.environmentChecked)
	assert.True(t, scraper.isCDBSupported())
	assert.True(t, scraper.isPDBSupported())
}

func TestContainerScraper_CheckEnvironmentCapability_NotCapable(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CheckCDBFeatureResult = 0
	mockClient.CheckPDBCapabilityResult = 0
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = scraper.checkEnvironmentCapability(ctx)

	assert.NoError(t, err)
	assert.True(t, scraper.environmentChecked)
	assert.False(t, scraper.isCDBSupported())
	assert.False(t, scraper.isPDBSupported())
}

func TestContainerScraper_CheckEnvironmentCapability_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("connection failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = scraper.checkEnvironmentCapability(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestContainerScraper_CheckCurrentContext_CDBRoot(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CheckCurrentContainerResult = models.ContainerContext{
		ContainerName: sql.NullString{String: "CDB$ROOT", Valid: true},
		ContainerID:   sql.NullString{String: "1", Valid: true},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = scraper.checkCurrentContext(ctx)

	assert.NoError(t, err)
	assert.True(t, scraper.isConnectedToCDBRoot())
	assert.False(t, scraper.isConnectedToPDB())
}

func TestContainerScraper_CheckCurrentContext_PDB(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CheckCurrentContainerResult = models.ContainerContext{
		ContainerName: sql.NullString{String: "MYPDB", Valid: true},
		ContainerID:   sql.NullString{String: "3", Valid: true},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = scraper.checkCurrentContext(ctx)

	assert.NoError(t, err)
	assert.False(t, scraper.isConnectedToCDBRoot())
	assert.True(t, scraper.isConnectedToPDB())
}

func TestContainerScraper_CheckCurrentContext_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("query failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = scraper.checkCurrentContext(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query failed")
}

func TestContainerScraper_ScrapeContainerStatus_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.ContainerStatusList = []models.ContainerStatus{
		{
			ConID:         sql.NullInt64{Int64: 1, Valid: true},
			ContainerName: sql.NullString{String: "CDB$ROOT", Valid: true},
			OpenMode:      sql.NullString{String: "READ WRITE", Valid: true},
			Restricted:    sql.NullString{String: "NO", Valid: true},
		},
		{
			ConID:         sql.NullInt64{Int64: 3, Valid: true},
			ContainerName: sql.NullString{String: "PDB1", Valid: true},
			OpenMode:      sql.NullString{String: "READ WRITE", Valid: true},
			Restricted:    sql.NullString{String: "NO", Valid: true},
		},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeContainerStatus(ctx, now)

	assert.Empty(t, errs)
}

func TestContainerScraper_ScrapeContainerStatus_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("query failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeContainerStatus(ctx, now)

	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "query failed")
}

func TestContainerScraper_ScrapeContainerStatus_InvalidData(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.ContainerStatusList = []models.ContainerStatus{
		{
			ConID:         sql.NullInt64{Valid: false},
			ContainerName: sql.NullString{String: "CDB$ROOT", Valid: true},
		},
		{
			ConID:         sql.NullInt64{Int64: 3, Valid: true},
			ContainerName: sql.NullString{Valid: false},
		},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeContainerStatus(ctx, now)

	assert.Empty(t, errs) // Should handle invalid data gracefully
}

func TestContainerScraper_ScrapePDBStatus_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.PDBStatusList = []models.PDBStatus{
		{
			ConID:     sql.NullInt64{Int64: 3, Valid: true},
			PDBName:   sql.NullString{String: "PDB1", Valid: true},
			OpenMode:  sql.NullString{String: "READ WRITE", Valid: true},
			TotalSize: sql.NullInt64{Int64: 1073741824, Valid: true},
		},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapePDBStatus(ctx, now)

	assert.Empty(t, errs)
}

func TestContainerScraper_ScrapePDBStatus_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("PDB query failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapePDBStatus(ctx, now)

	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "PDB query failed")
}

func TestContainerScraper_ScrapeCDBTablespaceUsage_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CDBTablespaceUsageList = []models.CDBTablespaceUsage{
		{
			ConID:          sql.NullInt64{Int64: 1, Valid: true},
			TablespaceName: sql.NullString{String: "SYSTEM", Valid: true},
			UsedBytes:      sql.NullInt64{Int64: 524288000, Valid: true},
			TotalBytes:     sql.NullInt64{Int64: 1073741824, Valid: true},
			UsedPercent:    sql.NullFloat64{Float64: 48.83, Valid: true},
		},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeCDBTablespaceUsage(ctx, now)

	assert.Empty(t, errs)
}

func TestContainerScraper_ScrapeCDBTablespaceUsage_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("tablespace query failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeCDBTablespaceUsage(ctx, now)

	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "tablespace query failed")
}

func TestContainerScraper_ScrapeCDBDataFiles_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CDBDataFilesList = []models.CDBDataFile{
		{
			ConID:          sql.NullInt64{Int64: 1, Valid: true},
			FileName:       sql.NullString{String: "/u01/oradata/system01.dbf", Valid: true},
			TablespaceName: sql.NullString{String: "SYSTEM", Valid: true},
			Bytes:          sql.NullInt64{Int64: 1073741824, Valid: true},
			Status:         sql.NullString{String: "AVAILABLE", Valid: true},
			Autoextensible: sql.NullString{String: "YES", Valid: true},
			MaxBytes:       sql.NullInt64{Int64: 34359738368, Valid: true},
			UserBytes:      sql.NullInt64{Int64: 1073479680, Valid: true},
		},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeCDBDataFiles(ctx, now)

	assert.Empty(t, errs)
}

func TestContainerScraper_ScrapeCDBDataFiles_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("datafile query failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeCDBDataFiles(ctx, now)

	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "datafile query failed")
}

func TestContainerScraper_ScrapeCDBServices_Success(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.CDBServicesList = []models.CDBService{
		{
			ConID:       sql.NullInt64{Int64: 1, Valid: true},
			ServiceName: sql.NullString{String: "ORCLCDB", Valid: true},
			NetworkName: sql.NullString{String: "orclcdb.example.com", Valid: true},
			PDB:         sql.NullString{String: "CDB$ROOT", Valid: true},
			Enabled:     sql.NullString{String: "YES", Valid: true},
		},
	}
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeCDBServices(ctx, now)

	assert.Empty(t, errs)
}

func TestContainerScraper_ScrapeCDBServices_Error(t *testing.T) {
	mockClient := client.NewMockClient()
	mockClient.QueryErr = errors.New("service query failed")
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())
	errs := scraper.scrapeCDBServices(ctx, now)

	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "service query failed")
}

func TestContainerScraper_IsCDBSupported(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	// Test nil capability
	assert.False(t, scraper.isCDBSupported())

	// Test false capability
	cdbCapable := false
	scraper.isCDBCapable = &cdbCapable
	assert.False(t, scraper.isCDBSupported())

	// Test true capability
	cdbCapable = true
	scraper.isCDBCapable = &cdbCapable
	assert.True(t, scraper.isCDBSupported())
}

func TestContainerScraper_IsPDBSupported(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	// Test nil capability
	assert.False(t, scraper.isPDBSupported())

	// Test false capability
	pdbCapable := false
	scraper.isPDBCapable = &pdbCapable
	assert.False(t, scraper.isPDBSupported())

	// Test true capability
	pdbCapable = true
	scraper.isPDBCapable = &pdbCapable
	assert.True(t, scraper.isPDBSupported())
}

func TestContainerScraper_IsConnectedToCDBRoot(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	// Test initial state
	assert.False(t, scraper.isConnectedToCDBRoot())

	// Test after checking context with CDB$ROOT
	mockClient.CheckCurrentContainerResult = models.ContainerContext{
		ContainerName: sql.NullString{String: "CDB$ROOT", Valid: true},
		ContainerID:   sql.NullString{String: "1", Valid: true},
	}
	ctx := context.Background()
	err = scraper.checkCurrentContext(ctx)
	require.NoError(t, err)
	assert.True(t, scraper.isConnectedToCDBRoot())

	// Test with PDB
	mockClient.CheckCurrentContainerResult = models.ContainerContext{
		ContainerName: sql.NullString{String: "PDB1", Valid: true},
		ContainerID:   sql.NullString{String: "3", Valid: true},
	}
	scraper.contextChecked = false // Reset to force recheck
	err = scraper.checkCurrentContext(ctx)
	require.NoError(t, err)
	assert.False(t, scraper.isConnectedToCDBRoot())
}

func TestContainerScraper_IsConnectedToPDB(t *testing.T) {
	mockClient := client.NewMockClient()
	settings := receivertest.NewNopSettings(metadata.Type)
	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), settings)
	logger := zap.NewNop()
	config := metadata.DefaultMetricsBuilderConfig()

	scraper, err := NewContainerScraper(mockClient, mb, logger, "test-instance", config, nil, nil)
	require.NoError(t, err)

	// Test initial state
	assert.False(t, scraper.isConnectedToPDB())

	// Test after checking context with PDB
	mockClient.CheckCurrentContainerResult = models.ContainerContext{
		ContainerName: sql.NullString{String: "PDB1", Valid: true},
		ContainerID:   sql.NullString{String: "3", Valid: true},
	}
	ctx := context.Background()
	err = scraper.checkCurrentContext(ctx)
	require.NoError(t, err)
	assert.True(t, scraper.isConnectedToPDB())

	// Test with CDB$ROOT
	mockClient.CheckCurrentContainerResult = models.ContainerContext{
		ContainerName: sql.NullString{String: "CDB$ROOT", Valid: true},
		ContainerID:   sql.NullString{String: "1", Valid: true},
	}
	scraper.contextChecked = false // Reset to force recheck
	err = scraper.checkCurrentContext(ctx)
	require.NoError(t, err)
	assert.False(t, scraper.isConnectedToPDB())
}
