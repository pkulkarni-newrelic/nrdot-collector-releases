// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

// MockClient is a mock implementation of OracleClient for testing.
type MockClient struct {
	SlowQueries            []models.SlowQuery
	ChildCursors           []models.ChildCursor
	WaitEventsWithBlocking []models.WaitEventWithBlocking

	// Connection metrics
	TotalSessions      int64
	ActiveSessionCount int64
	InactiveSessions   int64

	SessionStatusList            []models.SessionStatus
	SessionTypeList              []models.SessionType
	LogonStatsList               []models.LogonStat
	SessionResourcesList         []models.SessionResource
	CurrentWaitEventsList        []models.CurrentWaitEvent
	BlockingSessionsList         []models.BlockingSession
	WaitEventSummaryList         []models.WaitEventSummary
	ConnectionPoolMetricsList    []models.ConnectionPoolMetric
	SessionLimitsList            []models.SessionLimit
	ConnectionQualityMetricsList []models.ConnectionQualityMetric

	// Container metrics
	CheckCDBFeatureResult       int64
	CheckPDBCapabilityResult    int64
	CheckCurrentContainerResult models.ContainerContext
	ContainerStatusList         []models.ContainerStatus
	PDBStatusList               []models.PDBStatus
	CDBTablespaceUsageList      []models.CDBTablespaceUsage
	CDBDataFilesList            []models.CDBDataFile
	CDBServicesList             []models.CDBService

	// Disk I/O metrics
	DiskIOMetricsList []models.DiskIOMetrics

	// Instance metrics
	LockedAccountsList     []models.LockedAccountsMetric
	GlobalNameList         []models.GlobalNameMetric
	DBIDList               []models.DBIDMetric
	LongRunningQueriesList []models.LongRunningQueriesMetric

	// Lock metrics
	LockCountsList         []models.LockCount
	LockSessionCountsList  []models.LockSessionCount
	LockedObjectCountsList []models.LockedObjectCount

	// Memory metrics
	PGAMetricsList                    []models.PGAMetric
	SGAUGATotalMemoryList             []models.SGAUGATotalMemoryMetric
	SGASharedPoolLibraryCacheList     []models.SGASharedPoolLibraryCacheMetric
	SGASharedPoolLibraryCacheUserList []models.SGASharedPoolLibraryCacheUserMetric
	SGAMetricsList                    []models.SGAMetric

	// Performance metrics
	SysstatMetricsList          []models.SysstatMetric
	RollbackSegmentsMetricsList []models.RollbackSegmentsMetric
	RedoLogWaitsMetricsList     []models.RedoLogWaitsMetric

	// SGA Performance metrics
	SGASharedPoolLibraryCacheReloadRatioList []models.SGASharedPoolLibraryCacheReloadRatioMetric
	SGASharedPoolLibraryCacheHitRatioList    []models.SGASharedPoolLibraryCacheHitRatioMetric
	SGASharedPoolDictCacheMissRatioList      []models.SGASharedPoolDictCacheMissRatioMetric
	SGALogBufferSpaceWaitsList               []models.SGALogBufferSpaceWaitsMetric
	SGALogAllocRetriesList                   []models.SGALogAllocRetriesMetric
	SGAHitRatioList                          []models.SGAHitRatioMetric

	// Database Info metrics
	DatabaseInfoList []models.DatabaseInfoMetric
	DatabaseRole     *models.DatabaseRole

	// PDB metrics
	PDBSysMetricsList []models.PDBSysMetric
	CDBCapability     *models.CDBCapability

	// RAC metrics
	RACDetection          *models.RACDetection
	ASMDetection          *models.ASMDetection
	ASMDiskGroupsList     []models.ASMDiskGroup
	ClusterWaitEventsList []models.ClusterWaitEvent
	RACInstanceStatusList []models.RACInstanceStatus
	RACActiveServicesList []models.RACActiveService

	// Session metrics
	SessionCount *models.SessionCount

	// System metrics
	SystemMetricsList []models.SystemMetric

	// Tablespace metrics
	TablespaceUsageList                               []models.TablespaceUsage
	TablespaceGlobalNameList                          []models.TablespaceGlobalName
	TablespaceDBIDList                                []models.TablespaceDBID
	TablespaceCDBDatafilesOfflineList                 []models.TablespaceCDBDatafilesOffline
	TablespacePDBDatafilesOfflineList                 []models.TablespacePDBDatafilesOffline
	TablespacePDBDatafilesOfflineCurrentContainerList []models.TablespacePDBDatafilesOffline
	TablespacePDBNonWriteList                         []models.TablespacePDBNonWrite
	TablespacePDBNonWriteCurrentContainerList         []models.TablespacePDBNonWrite

	ConnectErr error
	CloseErr   error
	PingErr    error
	QueryErr   error
}

// NewMockClient creates a new mock client for testing.
func NewMockClient() *MockClient {
	return &MockClient{
		SlowQueries:            []models.SlowQuery{},
		WaitEventsWithBlocking: []models.WaitEventWithBlocking{},
	}
}

func (m *MockClient) Connect() error {
	return m.ConnectErr
}

func (m *MockClient) Close() error {
	return m.CloseErr
}

func (m *MockClient) Ping(ctx context.Context) error {
	return m.PingErr
}

func (m *MockClient) QueryExecutionPlanForChild(ctx context.Context, sqlID string, childNumber int64) ([]models.ExecutionPlanRow, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}

	// For mock testing, return empty rows
	// In real tests, this can be populated with test data
	return []models.ExecutionPlanRow{}, nil
}

func (m *MockClient) QuerySlowQueries(ctx context.Context, intervalSeconds, responseTimeThreshold, countThreshold int) ([]models.SlowQuery, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SlowQueries, nil
}

func (m *MockClient) QuerySpecificChildCursor(ctx context.Context, sqlID string, childNumber int64) (*models.ChildCursor, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	// Return first matching child cursor from mock data
	for _, cursor := range m.ChildCursors {
		if cursor.GetSQLID() == sqlID && cursor.GetChildNumber() == childNumber {
			return &cursor, nil
		}
	}
	return nil, nil
}

func (m *MockClient) QueryWaitEventsWithBlocking(ctx context.Context, countThreshold int, slowQuerySQLIDs []string) ([]models.WaitEventWithBlocking, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.WaitEventsWithBlocking, nil
}

func (m *MockClient) QueryTotalSessions(ctx context.Context) (int64, error) {
	if m.QueryErr != nil {
		return 0, m.QueryErr
	}
	return m.TotalSessions, nil
}

func (m *MockClient) QueryActiveSessions(ctx context.Context) (int64, error) {
	if m.QueryErr != nil {
		return 0, m.QueryErr
	}
	return m.ActiveSessionCount, nil
}

func (m *MockClient) QueryInactiveSessions(ctx context.Context) (int64, error) {
	if m.QueryErr != nil {
		return 0, m.QueryErr
	}
	return m.InactiveSessions, nil
}

func (m *MockClient) QuerySessionStatus(ctx context.Context) ([]models.SessionStatus, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SessionStatusList, nil
}

func (m *MockClient) QuerySessionTypes(ctx context.Context) ([]models.SessionType, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SessionTypeList, nil
}

func (m *MockClient) QueryLogonStats(ctx context.Context) ([]models.LogonStat, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.LogonStatsList, nil
}

func (m *MockClient) QuerySessionResources(ctx context.Context) ([]models.SessionResource, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SessionResourcesList, nil
}

func (m *MockClient) QueryCurrentWaitEvents(ctx context.Context) ([]models.CurrentWaitEvent, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.CurrentWaitEventsList, nil
}

func (m *MockClient) QueryBlockingSessions(ctx context.Context) ([]models.BlockingSession, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.BlockingSessionsList, nil
}

func (m *MockClient) QueryWaitEventSummary(ctx context.Context) ([]models.WaitEventSummary, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.WaitEventSummaryList, nil
}

func (m *MockClient) QueryConnectionPoolMetrics(ctx context.Context) ([]models.ConnectionPoolMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.ConnectionPoolMetricsList, nil
}

func (m *MockClient) QuerySessionLimits(ctx context.Context) ([]models.SessionLimit, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SessionLimitsList, nil
}

func (m *MockClient) QueryConnectionQuality(ctx context.Context) ([]models.ConnectionQualityMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.ConnectionQualityMetricsList, nil
}

// CheckCDBFeature mock
func (m *MockClient) CheckCDBFeature(ctx context.Context) (int64, error) {
	if m.QueryErr != nil {
		return 0, m.QueryErr
	}
	return m.CheckCDBFeatureResult, nil
}

// CheckPDBCapability mock
func (m *MockClient) CheckPDBCapability(ctx context.Context) (int64, error) {
	if m.QueryErr != nil {
		return 0, m.QueryErr
	}
	return m.CheckPDBCapabilityResult, nil
}

// CheckCurrentContainer mock
func (m *MockClient) CheckCurrentContainer(ctx context.Context) (models.ContainerContext, error) {
	if m.QueryErr != nil {
		return models.ContainerContext{}, m.QueryErr
	}
	return m.CheckCurrentContainerResult, nil
}

// QueryContainerStatus mock
func (m *MockClient) QueryContainerStatus(ctx context.Context) ([]models.ContainerStatus, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.ContainerStatusList, nil
}

// QueryPDBStatus mock
func (m *MockClient) QueryPDBStatus(ctx context.Context) ([]models.PDBStatus, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.PDBStatusList, nil
}

// QueryCDBTablespaceUsage mock
func (m *MockClient) QueryCDBTablespaceUsage(_ context.Context, _, _ []string) ([]models.CDBTablespaceUsage, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.CDBTablespaceUsageList, nil
}

// QueryCDBDataFiles mock
func (m *MockClient) QueryCDBDataFiles(ctx context.Context) ([]models.CDBDataFile, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.CDBDataFilesList, nil
}

// QueryCDBServices mock
func (m *MockClient) QueryCDBServices(ctx context.Context) ([]models.CDBService, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.CDBServicesList, nil
}

// QueryDiskIOMetrics mock
func (m *MockClient) QueryDiskIOMetrics(ctx context.Context) ([]models.DiskIOMetrics, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.DiskIOMetricsList, nil
}

// QueryLockedAccounts mock
func (m *MockClient) QueryLockedAccounts(ctx context.Context) ([]models.LockedAccountsMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.LockedAccountsList, nil
}

// QueryGlobalName mock
func (m *MockClient) QueryGlobalName(ctx context.Context) ([]models.GlobalNameMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.GlobalNameList, nil
}

// QueryDBID mock
func (m *MockClient) QueryDBID(ctx context.Context) ([]models.DBIDMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.DBIDList, nil
}

// QueryLongRunningQueries mock
func (m *MockClient) QueryLongRunningQueries(_ context.Context) ([]models.LongRunningQueriesMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.LongRunningQueriesList, nil
}

// QueryLockCounts mock
func (m *MockClient) QueryLockCounts(ctx context.Context) ([]models.LockCount, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.LockCountsList, nil
}

// QueryLockSessionCounts mock
func (m *MockClient) QueryLockSessionCounts(ctx context.Context) ([]models.LockSessionCount, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.LockSessionCountsList, nil
}

// QueryLockedObjectCounts mock
func (m *MockClient) QueryLockedObjectCounts(ctx context.Context) ([]models.LockedObjectCount, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.LockedObjectCountsList, nil
}

func (m *MockClient) QueryPGAMetrics(_ context.Context) ([]models.PGAMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.PGAMetricsList, nil
}

func (m *MockClient) QuerySGAUGATotalMemory(_ context.Context) ([]models.SGAUGATotalMemoryMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGAUGATotalMemoryList, nil
}

func (m *MockClient) QuerySGASharedPoolLibraryCache(_ context.Context) ([]models.SGASharedPoolLibraryCacheMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGASharedPoolLibraryCacheList, nil
}

func (m *MockClient) QuerySGASharedPoolLibraryCacheUser(_ context.Context) ([]models.SGASharedPoolLibraryCacheUserMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGASharedPoolLibraryCacheUserList, nil
}

func (m *MockClient) QuerySGAMetrics(_ context.Context) ([]models.SGAMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGAMetricsList, nil
}

func (m *MockClient) QuerySysstatMetrics(_ context.Context) ([]models.SysstatMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SysstatMetricsList, nil
}

func (m *MockClient) QueryRollbackSegmentsMetrics(_ context.Context) ([]models.RollbackSegmentsMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.RollbackSegmentsMetricsList, nil
}

func (m *MockClient) QueryRedoLogWaitsMetrics(_ context.Context) ([]models.RedoLogWaitsMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.RedoLogWaitsMetricsList, nil
}

func (m *MockClient) QuerySGASharedPoolLibraryCacheReloadRatio(_ context.Context) ([]models.SGASharedPoolLibraryCacheReloadRatioMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGASharedPoolLibraryCacheReloadRatioList, nil
}

func (m *MockClient) QuerySGASharedPoolLibraryCacheHitRatio(_ context.Context) ([]models.SGASharedPoolLibraryCacheHitRatioMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGASharedPoolLibraryCacheHitRatioList, nil
}

func (m *MockClient) QuerySGASharedPoolDictCacheMissRatio(_ context.Context) ([]models.SGASharedPoolDictCacheMissRatioMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGASharedPoolDictCacheMissRatioList, nil
}

func (m *MockClient) QuerySGALogBufferSpaceWaits(_ context.Context) ([]models.SGALogBufferSpaceWaitsMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGALogBufferSpaceWaitsList, nil
}

func (m *MockClient) QuerySGALogAllocRetries(_ context.Context) ([]models.SGALogAllocRetriesMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGALogAllocRetriesList, nil
}

func (m *MockClient) QuerySGAHitRatio(_ context.Context) ([]models.SGAHitRatioMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SGAHitRatioList, nil
}

func (m *MockClient) QueryDatabaseInfo(_ context.Context) ([]models.DatabaseInfoMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.DatabaseInfoList, nil
}

func (m *MockClient) QueryDatabaseRole(_ context.Context) (*models.DatabaseRole, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.DatabaseRole, nil
}

func (m *MockClient) QueryPDBSysMetrics(_ context.Context) ([]models.PDBSysMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.PDBSysMetricsList, nil
}

func (m *MockClient) QueryCDBCapability(_ context.Context) (*models.CDBCapability, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.CDBCapability, nil
}

func (m *MockClient) QueryRACDetection(_ context.Context) (*models.RACDetection, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.RACDetection, nil
}

func (m *MockClient) QueryASMDetection(_ context.Context) (*models.ASMDetection, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.ASMDetection, nil
}

func (m *MockClient) QueryASMDiskGroups(_ context.Context) ([]models.ASMDiskGroup, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.ASMDiskGroupsList, nil
}

func (m *MockClient) QueryClusterWaitEvents(_ context.Context) ([]models.ClusterWaitEvent, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.ClusterWaitEventsList, nil
}

func (m *MockClient) QueryRACInstanceStatus(_ context.Context) ([]models.RACInstanceStatus, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.RACInstanceStatusList, nil
}

func (m *MockClient) QueryRACActiveServices(_ context.Context) ([]models.RACActiveService, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.RACActiveServicesList, nil
}

func (m *MockClient) QuerySessionCount(_ context.Context) (*models.SessionCount, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SessionCount, nil
}

func (m *MockClient) QuerySystemMetrics(_ context.Context) ([]models.SystemMetric, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.SystemMetricsList, nil
}

func (m *MockClient) QueryTablespaceUsage(_ context.Context, _, _ []string) ([]models.TablespaceUsage, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespaceUsageList, nil
}

func (m *MockClient) QueryTablespaceGlobalName(_ context.Context, _, _ []string) ([]models.TablespaceGlobalName, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespaceGlobalNameList, nil
}

func (m *MockClient) QueryTablespaceDBID(_ context.Context, _, _ []string) ([]models.TablespaceDBID, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespaceDBIDList, nil
}

func (m *MockClient) QueryTablespaceCDBDatafilesOffline(_ context.Context, _, _ []string) ([]models.TablespaceCDBDatafilesOffline, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespaceCDBDatafilesOfflineList, nil
}

func (m *MockClient) QueryTablespacePDBDatafilesOffline(_ context.Context, _, _ []string) ([]models.TablespacePDBDatafilesOffline, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespacePDBDatafilesOfflineList, nil
}

func (m *MockClient) QueryTablespacePDBDatafilesOfflineCurrentContainer(_ context.Context, _, _ []string) ([]models.TablespacePDBDatafilesOffline, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespacePDBDatafilesOfflineCurrentContainerList, nil
}

func (m *MockClient) QueryTablespacePDBNonWrite(_ context.Context, _, _ []string) ([]models.TablespacePDBNonWrite, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespacePDBNonWriteList, nil
}

func (m *MockClient) QueryTablespacePDBNonWriteCurrentContainer(_ context.Context, _, _ []string) ([]models.TablespacePDBNonWrite, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.TablespacePDBNonWriteCurrentContainerList, nil
}
