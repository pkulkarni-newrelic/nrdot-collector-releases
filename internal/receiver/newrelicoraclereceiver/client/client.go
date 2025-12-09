// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

// OracleClient defines the interface for Oracle database operations.
// This abstraction allows for easy testing by injecting mock implementations.
type OracleClient interface {
	// Connection management
	Connect() error
	Close() error
	Ping(ctx context.Context) error

	// Execution plan queries
	QueryExecutionPlanForChild(ctx context.Context, sqlID string, childNumber int64) ([]models.ExecutionPlanRow, error)

	// Slow queries
	// intervalSeconds: Time window to fetch queries (e.g., 60 = last 60 seconds)
	// responseTimeThreshold: Threshold filtering done in Go after delta calculation
	// countThreshold: TOP N selection done in Go after delta calculation
	QuerySlowQueries(ctx context.Context, intervalSeconds, responseTimeThreshold, countThreshold int) ([]models.SlowQuery, error)

	// Child cursors (V$SQL)
	QuerySpecificChildCursor(ctx context.Context, sqlID string, childNumber int64) (*models.ChildCursor, error)

	// Combined wait events with blocking information
	// This replaces the separate QueryBlockingQueries and QueryWaitEvents methods
	// slowQuerySQLIDs: Optional list of SQL_IDs to filter by (empty slice returns all)
	QueryWaitEventsWithBlocking(ctx context.Context, countThreshold int, slowQuerySQLIDs []string) ([]models.WaitEventWithBlocking, error)

	// Connection metrics - simple counts
	QueryTotalSessions(ctx context.Context) (int64, error)
	QueryActiveSessions(ctx context.Context) (int64, error)
	QueryInactiveSessions(ctx context.Context) (int64, error)

	// Connection metrics - breakdowns
	QuerySessionStatus(ctx context.Context) ([]models.SessionStatus, error)
	QuerySessionTypes(ctx context.Context) ([]models.SessionType, error)
	QueryLogonStats(ctx context.Context) ([]models.LogonStat, error)

	// Connection metrics - resource consumption
	QuerySessionResources(ctx context.Context) ([]models.SessionResource, error)
	QueryCurrentWaitEvents(ctx context.Context) ([]models.CurrentWaitEvent, error)
	QueryBlockingSessions(ctx context.Context) ([]models.BlockingSession, error)
	QueryWaitEventSummary(ctx context.Context) ([]models.WaitEventSummary, error)

	// Connection metrics - pool and limits
	QueryConnectionPoolMetrics(ctx context.Context) ([]models.ConnectionPoolMetric, error)
	QuerySessionLimits(ctx context.Context) ([]models.SessionLimit, error)
	QueryConnectionQuality(ctx context.Context) ([]models.ConnectionQualityMetric, error)

	// Container metrics - capability checks
	CheckCDBFeature(ctx context.Context) (int64, error)
	CheckPDBCapability(ctx context.Context) (int64, error)
	CheckCurrentContainer(ctx context.Context) (models.ContainerContext, error)

	// Container metrics - data queries
	QueryContainerStatus(ctx context.Context) ([]models.ContainerStatus, error)
	QueryPDBStatus(ctx context.Context) ([]models.PDBStatus, error)
	QueryCDBTablespaceUsage(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.CDBTablespaceUsage, error)
	QueryCDBDataFiles(ctx context.Context) ([]models.CDBDataFile, error)
	QueryCDBServices(ctx context.Context) ([]models.CDBService, error)

	// Disk I/O metrics
	QueryDiskIOMetrics(ctx context.Context) ([]models.DiskIOMetrics, error)

	// Instance metrics
	QueryLockedAccounts(ctx context.Context) ([]models.LockedAccountsMetric, error)
	QueryGlobalName(ctx context.Context) ([]models.GlobalNameMetric, error)
	QueryDBID(ctx context.Context) ([]models.DBIDMetric, error)
	QueryLongRunningQueries(ctx context.Context) ([]models.LongRunningQueriesMetric, error)

	// Lock metrics
	QueryLockCounts(ctx context.Context) ([]models.LockCount, error)
	QueryLockSessionCounts(ctx context.Context) ([]models.LockSessionCount, error)
	QueryLockedObjectCounts(ctx context.Context) ([]models.LockedObjectCount, error)

	// Memory metrics
	QueryPGAMetrics(ctx context.Context) ([]models.PGAMetric, error)
	QuerySGAUGATotalMemory(ctx context.Context) ([]models.SGAUGATotalMemoryMetric, error)
	QuerySGASharedPoolLibraryCache(ctx context.Context) ([]models.SGASharedPoolLibraryCacheMetric, error)
	QuerySGASharedPoolLibraryCacheUser(ctx context.Context) ([]models.SGASharedPoolLibraryCacheUserMetric, error)
	QuerySGAMetrics(ctx context.Context) ([]models.SGAMetric, error)

	// Performance metrics
	QuerySysstatMetrics(ctx context.Context) ([]models.SysstatMetric, error)
	QueryRollbackSegmentsMetrics(ctx context.Context) ([]models.RollbackSegmentsMetric, error)
	QueryRedoLogWaitsMetrics(ctx context.Context) ([]models.RedoLogWaitsMetric, error)

	// SGA Performance metrics
	QuerySGASharedPoolLibraryCacheReloadRatio(ctx context.Context) ([]models.SGASharedPoolLibraryCacheReloadRatioMetric, error)
	QuerySGASharedPoolLibraryCacheHitRatio(ctx context.Context) ([]models.SGASharedPoolLibraryCacheHitRatioMetric, error)
	QuerySGASharedPoolDictCacheMissRatio(ctx context.Context) ([]models.SGASharedPoolDictCacheMissRatioMetric, error)
	QuerySGALogBufferSpaceWaits(ctx context.Context) ([]models.SGALogBufferSpaceWaitsMetric, error)
	QuerySGALogAllocRetries(ctx context.Context) ([]models.SGALogAllocRetriesMetric, error)
	QuerySGAHitRatio(ctx context.Context) ([]models.SGAHitRatioMetric, error)

	// Database Info metrics
	QueryDatabaseInfo(ctx context.Context) ([]models.DatabaseInfoMetric, error)
	QueryDatabaseRole(ctx context.Context) (*models.DatabaseRole, error)

	// PDB metrics
	QueryPDBSysMetrics(ctx context.Context) ([]models.PDBSysMetric, error)
	QueryCDBCapability(ctx context.Context) (*models.CDBCapability, error)

	// RAC metrics
	QueryRACDetection(ctx context.Context) (*models.RACDetection, error)
	QueryASMDetection(ctx context.Context) (*models.ASMDetection, error)
	QueryASMDiskGroups(ctx context.Context) ([]models.ASMDiskGroup, error)
	QueryClusterWaitEvents(ctx context.Context) ([]models.ClusterWaitEvent, error)
	QueryRACInstanceStatus(ctx context.Context) ([]models.RACInstanceStatus, error)
	QueryRACActiveServices(ctx context.Context) ([]models.RACActiveService, error)

	// Session metrics
	QuerySessionCount(ctx context.Context) (*models.SessionCount, error)

	// System metrics
	QuerySystemMetrics(ctx context.Context) ([]models.SystemMetric, error)

	// Tablespace metrics with filtering support
	QueryTablespaceUsage(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespaceUsage, error)
	QueryTablespaceGlobalName(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespaceGlobalName, error)
	QueryTablespaceDBID(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespaceDBID, error)
	QueryTablespaceCDBDatafilesOffline(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespaceCDBDatafilesOffline, error)
	QueryTablespacePDBDatafilesOffline(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespacePDBDatafilesOffline, error)
	QueryTablespacePDBDatafilesOfflineCurrentContainer(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespacePDBDatafilesOffline, error)
	QueryTablespacePDBNonWrite(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespacePDBNonWrite, error)
	QueryTablespacePDBNonWriteCurrentContainer(ctx context.Context, includeTablespaces, excludeTablespaces []string) ([]models.TablespacePDBNonWrite, error)
}
