package models

// SlowQuery represents slow query performance data collected from SQL Server
// This struct provides compatibility with the existing data structure format
type SlowQuery struct {
	QueryID                *QueryID `db:"query_id" metric_name:"query_id" source_type:"attribute"`
	PlanHandle             *QueryID `db:"plan_handle" metric_name:"plan_handle" source_type:"attribute"`
	QueryText              *string  `db:"query_text" metric_name:"query_text" source_type:"attribute"`
	DatabaseName           *string  `db:"database_name" metric_name:"database_name" source_type:"attribute"`
	SchemaName             *string  `db:"schema_name" metric_name:"schema_name" source_type:"attribute"`
	ObjectName             *string  `db:"object_name" metric_name:"object_name" source_type:"attribute"`
	LastExecutionTimestamp *string  `db:"last_execution_timestamp" metric_name:"last_execution_timestamp" source_type:"attribute"`
	ExecutionCount         *int64   `db:"execution_count" metric_name:"execution_count" source_type:"gauge"`
	AvgCPUTimeMS           *float64 `db:"avg_cpu_time_ms" metric_name:"sqlserver.slowquery.avg_cpu_time_ms" source_type:"gauge"`
	AvgElapsedTimeMS       *float64 `db:"avg_elapsed_time_ms" metric_name:"sqlserver.slowquery.avg_elapsed_time_ms" source_type:"gauge"`
	TotalElapsedTimeMS     *float64 `db:"total_elapsed_time_ms"` // Used for precise delta calculation only
	AvgDiskReads           *float64 `db:"avg_disk_reads" metric_name:"sqlserver.slowquery.avg_disk_reads" source_type:"gauge"`
	AvgDiskWrites          *float64 `db:"avg_disk_writes" metric_name:"sqlserver.slowquery.avg_disk_writes" source_type:"gauge"`
	AvgRowsProcessed       *float64 `db:"avg_rows_processed" metric_name:"sqlserver.slowquery.rows_processed" source_type:"gauge"`
	AvgLockWaitTimeMs      *float64 `db:"avg_lock_wait_time_ms" metric_name:"sqlserver.slowquery.avg_lock_wait_time_ms" source_type:"gauge"`
	StatementType          *string  `db:"statement_type" metric_name:"sqlserver.slowquery.statement_type" source_type:"attribute"`
	CollectionTimestamp    *string  `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
	// RCA Enhancement Fields
	MinElapsedTimeMs  *float64 `db:"min_elapsed_time_ms" metric_name:"sqlserver.slowquery.min_elapsed_time_ms" source_type:"gauge"`
	MaxElapsedTimeMs  *float64 `db:"max_elapsed_time_ms" metric_name:"sqlserver.slowquery.max_elapsed_time_ms" source_type:"gauge"`
	LastElapsedTimeMs *float64 `db:"last_elapsed_time_ms" metric_name:"sqlserver.slowquery.last_elapsed_time_ms" source_type:"gauge"`
	LastGrantKB       *float64 `db:"last_grant_kb" metric_name:"sqlserver.slowquery.last_grant_kb" source_type:"gauge"`
	LastUsedGrantKB   *float64 `db:"last_used_grant_kb" metric_name:"sqlserver.slowquery.last_used_grant_kb" source_type:"gauge"`
	LastSpills        *float64 `db:"last_spills" metric_name:"sqlserver.slowquery.last_spills" source_type:"gauge"`
	MaxSpills         *float64 `db:"max_spills" metric_name:"sqlserver.slowquery.max_spills" source_type:"gauge"`
	LastDOP           *float64 `db:"last_dop" metric_name:"sqlserver.slowquery.last_dop" source_type:"gauge"`

	// Interval-based delta metrics (calculated in-memory, not from DB)
	// These are populated by the SimplifiedIntervalCalculator
	// NOTE: Only elapsed time has delta calculation, CPU uses historical average from DB
	IntervalAvgElapsedTimeMS *float64 `db:"-" metric_name:"sqlserver.slowquery.interval_avg_elapsed_time_ms" source_type:"gauge"`
	IntervalExecutionCount   *int64   `db:"-" metric_name:"sqlserver.slowquery.interval_execution_count" source_type:"gauge"`
}

// WaitTimeAnalysis represents wait time analysis data for SQL Server queries
type WaitTimeAnalysis struct {
	QueryID             *QueryID `db:"query_id" metric_name:"query_id" source_type:"attribute"`
	DatabaseName        *string  `db:"database_name" metric_name:"database_name" source_type:"attribute"`
	QueryText           *string  `db:"query_text" metric_name:"query_text" source_type:"attribute"`
	WaitCategory        *string  `db:"wait_category" metric_name:"wait_category" source_type:"attribute"`
	TotalWaitTimeMs     *float64 `db:"total_wait_time_ms" metric_name:"total_wait_time_ms" source_type:"gauge"`
	AvgWaitTimeMs       *float64 `db:"avg_wait_time_ms" metric_name:"avg_wait_time_ms" source_type:"gauge"`
	WaitEventCount      *int64   `db:"wait_event_count" metric_name:"wait_event_count" source_type:"gauge"`
	LastExecutionTime   *string  `db:"last_execution_time" metric_name:"last_execution_time" source_type:"attribute"`
	CollectionTimestamp *string  `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
}

// QueryExecutionPlan represents detailed execution plan analysis data for SQL Server queries
// This model is used for drill-down analysis from slow query detection to specific execution plans
type QueryExecutionPlan struct {
	QueryID           *QueryID `db:"query_id" metric_name:"query_id" source_type:"attribute"`
	PlanHandle        *QueryID `db:"plan_handle" metric_name:"plan_handle" source_type:"attribute"`
	QueryPlanID       *QueryID `db:"query_plan_id" metric_name:"query_plan_id" source_type:"attribute"`
	SQLText           *string  `db:"sql_text" metric_name:"sql_text" source_type:"attribute"`
	TotalCPUMs        *float64 `db:"total_cpu_ms" metric_name:"total_cpu_ms" source_type:"gauge"`
	TotalElapsedMs    *float64 `db:"total_elapsed_ms" metric_name:"total_elapsed_ms" source_type:"gauge"`
	CreationTime      *string  `db:"creation_time" metric_name:"creation_time" source_type:"attribute"`
	LastExecutionTime *string  `db:"last_execution_time" metric_name:"last_execution_time" source_type:"attribute"`
	ExecutionPlanXML  *string  `db:"execution_plan_xml" metric_name:"execution_plan_xml" source_type:"attribute"`
}

// ExecutionPlanNode represents a parsed execution plan node with detailed operator information
// This model contains the parsed data structure from XML execution plans for New Relic logging
type ExecutionPlanNode struct {
	// Identifiers
	QueryID      string `json:"query_id"`
	PlanHandle   string `json:"plan_handle"`
	NodeID       int    `json:"node_id"`
	ParentNodeID int    `json:"parent_node_id"`

	// SQL Query Information
	SQLText string `json:"sql_text"`

	// Operator Information
	PhysicalOp string `json:"physical_op"`
	LogicalOp  string `json:"logical_op"`
	InputType  string `json:"input_type"`

	// Object Information (for Index Scan/Seek operators)
	SchemaName        string `json:"schema_name"`
	TableName         string `json:"table_name"`
	IndexName         string `json:"index_name"`
	ReferencedColumns string `json:"referenced_columns"`

	// Cost Estimates
	EstimateRows          float64 `json:"estimate_rows"`
	EstimateIO            float64 `json:"estimate_io"`
	EstimateCPU           float64 `json:"estimate_cpu"`
	AvgRowSize            float64 `json:"avg_row_size"`
	TotalSubtreeCost      float64 `json:"total_subtree_cost"`
	EstimatedOperatorCost float64 `json:"estimated_operator_cost"`

	// Execution Details
	EstimatedExecutionMode string `json:"estimated_execution_mode"`
	GrantedMemoryKb        int64  `json:"granted_memory_kb"`
	SpillOccurred          bool   `json:"spill_occurred"`
	NoJoinPredicate        bool   `json:"no_join_predicate"`

	// Performance Metrics
	TotalWorkerTime    float64 `json:"total_worker_time"`
	TotalElapsedTime   float64 `json:"total_elapsed_time"`
	TotalLogicalReads  int64   `json:"total_logical_reads"`
	TotalLogicalWrites int64   `json:"total_logical_writes"`
	ExecutionCount     int64   `json:"execution_count"`
	AvgElapsedTimeMs   float64 `json:"avg_elapsed_time_ms"`

	// Timestamps
	CollectionTimestamp string `json:"collection_timestamp"`
	LastExecutionTime   string `json:"last_execution_time"`
}

// ExecutionPlanAnalysis represents the complete parsed execution plan with metadata
type ExecutionPlanAnalysis struct {
	QueryID        string              `json:"query_id"`
	PlanHandle     string              `json:"plan_handle"`
	SQLText        string              `json:"sql_text"`
	TotalCost      float64             `json:"total_cost"`
	CompileTime    string              `json:"compile_time"`
	CompileCPU     int64               `json:"compile_cpu"`
	CompileMemory  int64               `json:"compile_memory"`
	Nodes          []ExecutionPlanNode `json:"nodes"`
	CollectionTime string              `json:"collection_time"`
}

// ActiveRunningQuery represents currently executing queries with wait and blocking details
// This model captures real-time execution state from sys.dm_exec_requests
//
// RCA Enhancement: Includes query_hash (query_id) for correlation with slow queries, plus a
// correlation_query_id fallback that uses text hash when query_hash is NULL (non-cached queries)
type ActiveRunningQuery struct {
	// A. Current Session Details
	CurrentSessionID         *int64   `db:"current_session_id" metric_name:"sqlserver.activequery.session_id" source_type:"gauge"`
	RequestID                *int64   `db:"request_id" metric_name:"request_id" source_type:"attribute"`
	DatabaseName             *string  `db:"database_name" metric_name:"database_name" source_type:"attribute"`
	SchemaName               *string  `db:"schema_name" metric_name:"schema_name" source_type:"attribute"`
	ObjectName               *string  `db:"object_name" metric_name:"object_name" source_type:"attribute"`
	LoginName                *string  `db:"login_name" metric_name:"login_name" source_type:"attribute"`
	HostName                 *string  `db:"host_name" metric_name:"host_name" source_type:"attribute"`
	ProgramName              *string  `db:"program_name" metric_name:"program_name" source_type:"attribute"`
	RequestCommand           *string  `db:"request_command" metric_name:"request_command" source_type:"attribute"`
	RequestStatus            *string  `db:"request_status" metric_name:"request_status" source_type:"attribute"`
	QueryID                  *QueryID `db:"query_id" metric_name:"query_id" source_type:"attribute"`
	WaitType                 *string  `db:"wait_type" metric_name:"wait_type" source_type:"attribute"`
	WaitTimeS                *float64 `db:"wait_time_s" metric_name:"sqlserver.activequery.wait_time_seconds" source_type:"gauge"`
	WaitResource             *string  `db:"wait_resource" metric_name:"wait_resource" source_type:"attribute"`
	WaitResourceDecoded      *string  `db:"wait_resource_decoded" metric_name:"wait_resource_decoded" source_type:"attribute"`
	WaitResourceObjectName   *string  `db:"wait_resource_object_name" metric_name:"wait_resource_object_name" source_type:"attribute"`
	WaitResourceDatabaseName *string  `db:"wait_resource_database_name" metric_name:"wait_resource_database_name" source_type:"attribute"`
	LastWaitType             *string  `db:"last_wait_type" metric_name:"last_wait_type" source_type:"attribute"`

	WaitResourceSchemaNameObject *string `db:"wait_resource_schema_name_object" metric_name:"wait_resource_schema_name_object" source_type:"attribute"`
	WaitResourceObjectType       *string `db:"wait_resource_object_type" metric_name:"wait_resource_object_type" source_type:"attribute"`

	WaitResourceSchemaNameIndex *string `db:"wait_resource_schema_name_index" metric_name:"wait_resource_schema_name_index" source_type:"attribute"`
	WaitResourceTableNameIndex  *string `db:"wait_resource_table_name_index" metric_name:"wait_resource_table_name_index" source_type:"attribute"`
	WaitResourceIndexName       *string `db:"wait_resource_index_name" metric_name:"wait_resource_index_name" source_type:"attribute"`
	WaitResourceIndexType       *string `db:"wait_resource_index_type" metric_name:"wait_resource_index_type" source_type:"attribute"`

	CPUTimeMs               *int64  `db:"cpu_time_ms" metric_name:"sqlserver.activequery.cpu_time_ms" source_type:"gauge"`
	TotalElapsedTimeMs      *int64  `db:"total_elapsed_time_ms" metric_name:"sqlserver.activequery.elapsed_time_ms" source_type:"gauge"`
	Reads                   *int64  `db:"reads" metric_name:"sqlserver.activequery.reads" source_type:"gauge"`
	Writes                  *int64  `db:"writes" metric_name:"sqlserver.activequery.writes" source_type:"gauge"`
	LogicalReads            *int64  `db:"logical_reads" metric_name:"sqlserver.activequery.logical_reads" source_type:"gauge"`
	RowCount                *int64  `db:"row_count" metric_name:"sqlserver.activequery.row_count" source_type:"gauge"`
	GrantedQueryMemoryPages *int64  `db:"granted_query_memory_pages" metric_name:"sqlserver.activequery.granted_query_memory_pages" source_type:"gauge"`
	RequestStartTime        *string `db:"request_start_time" metric_name:"request_start_time" source_type:"attribute"`
	CollectionTimestamp     *string `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`

	// E. Transaction Context (RCA for long-running transactions)
	TransactionID             *int64 `db:"transaction_id" metric_name:"transaction_id" source_type:"attribute"`
	OpenTransactionCount      *int64 `db:"open_transaction_count" metric_name:"open_transaction_count" source_type:"gauge"`
	TransactionIsolationLevel *int64 `db:"transaction_isolation_level" metric_name:"transaction_isolation_level" source_type:"attribute"`

	// F. Parallel Execution Details (RCA for CXPACKET waits)
	DegreeOfParallelism *int64 `db:"degree_of_parallelism" metric_name:"degree_of_parallelism" source_type:"gauge"`
	ParallelWorkerCount *int64 `db:"parallel_worker_count" metric_name:"parallel_worker_count" source_type:"gauge"`

	// G. Session Context
	SessionStatus       *string `db:"session_status" metric_name:"session_status" source_type:"attribute"`
	ClientInterfaceName *string `db:"client_interface_name" metric_name:"client_interface_name" source_type:"attribute"`

	// H. Plan Handle for execution plan fetching
	PlanHandle       *QueryID `db:"plan_handle" metric_name:"plan_handle" source_type:"attribute"`
	ExecutionPlanXML *string  `db:"execution_plan_xml" metric_name:"execution_plan_xml" source_type:"attribute"`

	// I. Blocking Details
	// LINKING FIX: Changed from *string to *int64 for proper numeric joins
	// NULL when not blocked (was "N/A" string) - enables: WHERE blocking_session_id = 123
	BlockingSessionID           *int64  `db:"blocking_session_id" metric_name:"blocking_session_id" source_type:"attribute"`
	BlockerLoginName            *string `db:"blocker_login_name" metric_name:"blocker_login_name" source_type:"attribute"`
	BlockerHostName             *string `db:"blocker_host_name" metric_name:"blocker_host_name" source_type:"attribute"`
	BlockerProgramName          *string `db:"blocker_program_name" metric_name:"blocker_program_name" source_type:"attribute"`
	BlockerStatus               *string `db:"blocker_status" metric_name:"blocker_status" source_type:"attribute"`
	BlockerIsolationLevel       *int64  `db:"blocker_isolation_level" metric_name:"blocker_isolation_level" source_type:"attribute"`
	BlockerOpenTransactionCount *int64  `db:"blocker_open_transaction_count" metric_name:"blocker_open_transaction_count" source_type:"gauge"`

	// J. Query Text
	QueryStatementText         *string `db:"query_statement_text" metric_name:"query_statement_text" source_type:"attribute"`
	BlockingQueryStatementText *string `db:"blocking_query_statement_text" metric_name:"blocking_query_statement_text" source_type:"attribute"`
}

// LockedObject represents detailed information about database objects locked by a session
// This model resolves lock resources to actual table/object names for troubleshooting lock contention
type LockedObject struct {
	// Session and Database Context
	SessionID    *int64  `db:"session_id" metric_name:"session_id" source_type:"attribute"`
	DatabaseName *string `db:"database_name" metric_name:"database_name" source_type:"attribute"`

	// Object Identification
	SchemaName       *string `db:"schema_name" metric_name:"schema_name" source_type:"attribute"`
	LockedObjectName *string `db:"locked_object_name" metric_name:"locked_object_name" source_type:"attribute"`

	// Lock Details
	ResourceType        *string `db:"resource_type" metric_name:"resource_type" source_type:"attribute"`
	LockGranularity     *string `db:"lock_granularity" metric_name:"lock_granularity" source_type:"attribute"`
	LockMode            *string `db:"lock_mode" metric_name:"lock_mode" source_type:"attribute"`
	LockStatus          *string `db:"lock_status" metric_name:"lock_status" source_type:"attribute"`
	LockRequestType     *string `db:"lock_request_type" metric_name:"lock_request_type" source_type:"attribute"`
	ResourceDescription *string `db:"resource_description" metric_name:"resource_description" source_type:"attribute"`

	// Metadata
	CollectionTimestamp *string `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
}

// PlanHandleResult represents a plan_handle and its associated execution plan for an active query
// Used to fetch top N most recently used execution plans for a given query_hash
type PlanHandleResult struct {
	PlanHandle        *QueryID `db:"plan_handle"`
	QueryHash         *QueryID `db:"query_hash"`
	QueryID           *QueryID `db:"query_id"` // Alias for query_hash (used in slow query context)
	QueryPlanHash     *QueryID `db:"query_plan_hash"`
	LastExecutionTime *string  `db:"last_execution_time"`
	CreationTime      *string  `db:"creation_time"`
	ExecutionCount    *int64   `db:"execution_count"`

	// Elapsed Time Statistics
	TotalElapsedTimeMs *float64 `db:"total_elapsed_time_ms"`
	AvgElapsedTimeMs   *float64 `db:"avg_elapsed_time_ms"`
	MinElapsedTimeMs   *float64 `db:"min_elapsed_time_ms"`
	MaxElapsedTimeMs   *float64 `db:"max_elapsed_time_ms"`
	LastElapsedTimeMs  *float64 `db:"last_elapsed_time_ms"`

	// Worker Time Statistics
	TotalWorkerTimeMs *float64 `db:"total_worker_time_ms"`
	AvgWorkerTimeMs   *float64 `db:"avg_worker_time_ms"`

	// I/O Statistics
	TotalLogicalReads  *int64   `db:"total_logical_reads"`
	TotalLogicalWrites *int64   `db:"total_logical_writes"`
	TotalPhysicalReads *int64   `db:"total_physical_reads"`
	AvgLogicalReads    *float64 `db:"avg_logical_reads"`
	AvgLogicalWrites   *float64 `db:"avg_logical_writes"`
	AvgPhysicalReads   *float64 `db:"avg_physical_reads"`

	// Row Statistics
	AvgRows *float64 `db:"avg_rows"`

	// Memory Grant Statistics
	LastGrantKB     *float64 `db:"last_grant_kb"`
	LastUsedGrantKB *float64 `db:"last_used_grant_kb"`
	MinGrantKB      *float64 `db:"min_grant_kb"`
	MaxGrantKB      *float64 `db:"max_grant_kb"`

	// Spill Statistics
	LastSpills *int64 `db:"last_spills"`
	MaxSpills  *int64 `db:"max_spills"`

	// Parallelism Statistics
	LastDOP *int64 `db:"last_dop"`
	MinDOP  *int64 `db:"min_dop"`
	MaxDOP  *int64 `db:"max_dop"`

	// Execution Plan XML
	ExecutionPlanXML *string `db:"execution_plan_xml"`
}

// ExecutionPlanTopLevelDetails represents high-level execution plan details (not node-level)
// Extracted from XML execution plan for lightweight analysis
type ExecutionPlanTopLevelDetails struct {
	// Identifiers
	QueryID       string `json:"query_id"`
	PlanHandle    string `json:"plan_handle"`
	QueryPlanHash string `json:"query_plan_hash"`

	// SQL Query Information
	SQLText string `json:"sql_text"`

	// Top-Level Plan Costs (from root RelOp)
	TotalSubtreeCost     float64 `json:"total_subtree_cost"`
	StatementSubTreeCost float64 `json:"statement_subtree_cost"`
	EstimatedTotalCost   float64 `json:"estimated_total_cost"`

	// Compilation Details
	CompileTime    string `json:"compile_time"`
	CompileCPU     int64  `json:"compile_cpu"`
	CompileMemory  int64  `json:"compile_memory"`
	CachedPlanSize int64  `json:"cached_plan_size"`

	// Optimizer Details
	StatementOptmLevel            string `json:"statement_optm_level"`
	StatementOptmEarlyAbortReason string `json:"statement_optm_early_abort_reason"`

	// Execution Counts and Timing
	ExecutionCount    int64   `json:"execution_count"`
	AvgElapsedTimeMs  float64 `json:"avg_elapsed_time_ms"`
	AvgWorkerTimeMs   float64 `json:"avg_worker_time_ms"`
	LastExecutionTime string  `json:"last_execution_time"`
	CreationTime      string  `json:"creation_time"`

	// Resource Estimates (from StmtSimple)
	EstimateRows float64 `json:"estimate_rows"`
	EstimateIO   float64 `json:"estimate_io"`
	EstimateCPU  float64 `json:"estimate_cpu"`
	AvgRowSize   int64   `json:"avg_row_size"`

	// Plan Details
	DegreeOfParallelism int64 `json:"degree_of_parallelism"`
	MemoryGrant         int64 `json:"memory_grant"`
	CachedPlanSize64    int64 `json:"cached_plan_size_64"`

	// Missing Index Information (if any)
	MissingIndexCount  int     `json:"missing_index_count"`
	MissingIndexImpact float64 `json:"missing_index_impact"`

	// Warnings
	NoJoinPredicate         bool `json:"no_join_predicate"`
	ColumnsWithNoStatistics int  `json:"columns_with_no_statistics"`
	UnmatchedIndexes        int  `json:"unmatched_indexes"`

	// Operator Summary
	TotalOperators int `json:"total_operators"`
	ScansCount     int `json:"scans_count"`
	SeeksCount     int `json:"seeks_count"`

	// Timestamps
	CollectionTimestamp string `json:"collection_timestamp"`
}
