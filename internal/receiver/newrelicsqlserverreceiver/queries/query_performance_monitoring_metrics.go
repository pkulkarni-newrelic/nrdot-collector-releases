package queries

const SlowQuery = `DECLARE @IntervalSeconds INT = %d; 		-- Define the interval in seconds
DECLARE @TopN INT = %d; 				-- Number of top queries to retrieve
DECLARE @ElapsedTimeThreshold INT = %d;  -- Elapsed time threshold in milliseconds
DECLARE @TextTruncateLimit INT = %d; 	-- Truncate limit for query_text
				
WITH StatementDetails AS (
	SELECT
		qs.plan_handle,
		qs.sql_handle,
		-- Extract query text using Microsoft's official offset logic (no +1 on length)
		-- Reference: https://learn.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/sys-dm-exec-query-stats-transact-sql
		LEFT(SUBSTRING(
			qt.text,
			(qs.statement_start_offset / 2) + 1,
			(
				CASE
					qs.statement_end_offset
					WHEN -1 THEN DATALENGTH(qt.text)
					ELSE qs.statement_end_offset
				END - qs.statement_start_offset
			) / 2
		), @TextTruncateLimit) AS query_text,
		-- query_id: SQL Server's query_hash - used for correlating with active query metrics
		qs.query_hash AS query_id,
		qs.last_execution_time,
		qs.execution_count,
        -- Historical average metrics (reflecting all runs since caching)
		(qs.total_worker_time / qs.execution_count) / 1000.0 AS avg_cpu_time_ms,
		(qs.total_elapsed_time / qs.execution_count) / 1000.0 AS avg_elapsed_time_ms,
		-- Total elapsed time for precise delta calculation (avoids floating point precision loss)
		qs.total_elapsed_time / 1000.0 AS total_elapsed_time_ms,
		(qs.total_logical_reads / qs.execution_count) AS avg_disk_reads,
		(qs.total_logical_writes / qs.execution_count) AS avg_disk_writes,
		-- Average rows processed (returned by query)
		(qs.total_rows / qs.execution_count) AS avg_rows_processed,
		-- RCA Enhancement Fields: Performance variance
		qs.min_elapsed_time / 1000.0 AS min_elapsed_time_ms,
		qs.max_elapsed_time / 1000.0 AS max_elapsed_time_ms,
		qs.last_elapsed_time / 1000.0 AS last_elapsed_time_ms,
		-- RCA Enhancement Fields: Memory grants
		qs.last_grant_kb,
		qs.last_used_grant_kb,
		-- RCA Enhancement Fields: TempDB spills
		qs.last_spills,
		qs.max_spills,
		-- RCA Enhancement Fields: Parallelism
		qs.last_dop,
		-- Lock wait time approximation: elapsed_time - (cpu_time + io_time)
		-- NOTE: This is an approximation as dm_exec_query_stats doesn't track lock waits separately
		-- For precise lock wait time, Query Store wait_category = 4 (Lock waits) should be used
		CASE
			WHEN (qs.total_elapsed_time / qs.execution_count) > (qs.total_worker_time / qs.execution_count)
			THEN ((qs.total_elapsed_time - qs.total_worker_time) / qs.execution_count) / 1000.0
			ELSE 0.0
		END AS avg_lock_wait_time_ms,
		-- Determine statement type (SELECT, INSERT, etc.)
		CASE
			WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'SELECT' THEN 'SELECT'
			WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'INSERT' THEN 'INSERT'
			WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'UPDATE' THEN 'UPDATE'
			WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'DELETE' THEN 'DELETE'
			ELSE 'OTHER'
		END AS statement_type,
		CONVERT(INT, pa.value) AS database_id,
		qt.objectid
	FROM
		sys.dm_exec_query_stats qs
		CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS qt
		OUTER APPLY (
			SELECT TOP 1 pa.value
			FROM sys.dm_exec_plan_attributes(qs.plan_handle) AS pa
			WHERE pa.attribute = 'dbid'
		) AS pa
	WHERE
		-- *** KEY FILTER: Only plans that ran in the last @IntervalSeconds (e.g., 15) ***
		qs.last_execution_time >= DATEADD(SECOND, -@IntervalSeconds, GETUTCDATE())
		AND qs.execution_count > 0
		AND qt.text IS NOT NULL
		AND LTRIM(RTRIM(qt.text)) <> ''
		AND DB_NAME(CONVERT(INT, pa.value)) NOT IN ('master', 'model', 'msdb', 'tempdb')
		-- OPTIMIZED: Use objectid filter instead of expensive LIKE (100x faster)
		-- System objects have objectid < 100, user objects >= 100, ad-hoc queries = NULL
		AND (qt.objectid IS NULL OR qt.objectid >= 100)
)
-- Select the raw, non-aggregated statement data.
-- NOTE: TOP N filtering removed - will be applied in Go code AFTER delta calculation
-- This ensures we get enough candidates for interval-based (delta) averaging
SELECT
    s.query_id,
	s.plan_handle,
    s.query_text,
    DB_NAME(s.database_id) AS database_name,
    COALESCE(
        OBJECT_SCHEMA_NAME(s.objectid, s.database_id),
        'N/A'
    ) AS schema_name,
    CONVERT(VARCHAR(25), SWITCHOFFSET(CAST(s.last_execution_time AS DATETIMEOFFSET), '+00:00'), 127) + 'Z' AS last_execution_timestamp,
    s.execution_count,
    s.avg_cpu_time_ms,
    s.avg_elapsed_time_ms,
    s.total_elapsed_time_ms,
    s.avg_disk_reads,
    s.avg_disk_writes,
    s.avg_rows_processed,
    s.avg_lock_wait_time_ms,
    s.statement_type,
    -- RCA Enhancement Fields
    s.min_elapsed_time_ms,
    s.max_elapsed_time_ms,
    s.last_elapsed_time_ms,
    s.last_grant_kb,
    s.last_used_grant_kb,
    s.last_spills,
    s.max_spills,
    s.last_dop,
    CONVERT(VARCHAR(25), SWITCHOFFSET(SYSDATETIMEOFFSET(), '+00:00'), 127) + 'Z' AS collection_timestamp
FROM
    StatementDetails s`



// ActiveQueryExecutionPlanQuery fetches the execution plan for an active query using its plan_handle
// NOTE: This is used only for active running queries, NOT for slow queries from dm_exec_query_stats
const ActiveQueryExecutionPlanQuery = `
SELECT
    CAST(qp.query_plan AS NVARCHAR(MAX)) AS execution_plan_xml
FROM sys.dm_exec_query_plan(%s) AS qp
WHERE qp.query_plan IS NOT NULL;`

// ActiveRunningQueriesQuery retrieves currently executing queries with wait and blocking details
// This query captures real-time execution state including wait types, blocking chains, and query text
//
// RCA Enhancement: Includes query_hash for correlation with slow queries, with fallback to text hash
// when query_hash is NULL (queries not yet cached in dm_exec_query_stats)
const ActiveRunningQueriesQuery = `
DECLARE @Limit INT = %d; -- Set the maximum number of rows to return
DECLARE @TextTruncateLimit INT = %d; -- Set the maximum length for query text
DECLARE @ElapsedTimeThresholdMs INT = %d; -- Minimum elapsed time threshold in milliseconds

SELECT TOP (@Limit)
    -- A. CURRENT SESSION DETAILS
    r_wait.session_id AS current_session_id,
    r_wait.request_id AS request_id,
    DB_NAME(r_wait.database_id) AS database_name,
    s_wait.login_name AS login_name,
    s_wait.host_name AS host_name,
    s_wait.program_name AS program_name,
    r_wait.command AS request_command,
    r_wait.status AS request_status,

    -- B. CORRELATION KEY (Critical for RCA)
    -- query_id: SQL Server's query_hash - used for correlating active queries with slow query metrics
    -- NULL indicates query is not correlatable (ad-hoc SQL with different literals, OPTION(RECOMPILE), etc.)
    -- Only parameterized queries and stored procedures get consistent query_hash values
    r_wait.query_hash AS query_id,

    -- B2. QUERY TEXT (moved up before wait decoding to avoid SUBSTRING errors in later sections)
    -- Using full text to avoid SUBSTRING calculation errors when offsets are invalid
    LEFT(st_wait.text, @TextTruncateLimit) AS query_statement_text,

    -- B3. QUERY CONTEXT - Schema and Object Name (for stored procedures)
    -- Use query plan's objectid (qp_wait.objectid) not SQL text's objectid (st_wait.objectid)
    -- When stored procedures execute, sql_text shows the statement inside the procedure (NULL objectid)
    -- but query_plan contains the actual stored procedure's objectid
    OBJECT_SCHEMA_NAME(qp_wait.objectid, qp_wait.dbid) AS schema_name,
    OBJECT_NAME(qp_wait.objectid, qp_wait.dbid) AS object_name,

    -- C. WAIT DETAILS
    r_wait.wait_type AS wait_type,
    r_wait.wait_time / 1000.0 AS wait_time_s,
    r_wait.wait_resource AS wait_resource,
    r_wait.last_wait_type AS last_wait_type,

    -- C2. WAIT RESOURCE DECODED (Lightweight - only resolve what's guaranteed to work)
    -- Extracts object name for OBJECT locks using fast built-in OBJECT_NAME function
    CASE
        WHEN r_wait.wait_resource LIKE 'OBJECT:%%' THEN
            OBJECT_NAME(
                TRY_CAST(
                    SUBSTRING(
                        r_wait.wait_resource,
                        CHARINDEX(':', r_wait.wait_resource, 8) + 1,
                        CASE
                            WHEN CHARINDEX(':', r_wait.wait_resource, CHARINDEX(':', r_wait.wait_resource, 8) + 1) > 0
                            THEN CHARINDEX(':', r_wait.wait_resource, CHARINDEX(':', r_wait.wait_resource, 8) + 1) - CHARINDEX(':', r_wait.wait_resource, 8) - 1
                            ELSE LEN(r_wait.wait_resource) - CHARINDEX(':', r_wait.wait_resource, 8)
                        END
                    ) AS INT
                ),
                r_wait.database_id
            )
        ELSE NULL
    END AS wait_resource_object_name,

    -- Extract database name for wait resources that contain database_id
    CASE
        WHEN r_wait.wait_resource LIKE 'KEY:%%' OR
             r_wait.wait_resource LIKE 'PAGE:%%' OR
             r_wait.wait_resource LIKE 'RID:%%' OR
             r_wait.wait_resource LIKE 'OBJECT:%%' OR
             r_wait.wait_resource LIKE 'DATABASE:%%' OR
             r_wait.wait_resource LIKE 'FILE:%%' OR
             r_wait.wait_resource LIKE 'EXTENT:%%' THEN
            DB_NAME(
                TRY_CAST(
                    SUBSTRING(
                        r_wait.wait_resource,
                        PATINDEX('%[0-9]%', r_wait.wait_resource),
                        CASE
                            WHEN CHARINDEX(':', r_wait.wait_resource, PATINDEX('%[0-9]%', r_wait.wait_resource)) > 0
                            THEN CHARINDEX(':', r_wait.wait_resource, PATINDEX('%[0-9]%', r_wait.wait_resource)) - PATINDEX('%[0-9]%', r_wait.wait_resource)
                            ELSE LEN(r_wait.wait_resource)
                        END
                    ) AS INT
                )
            )
        ELSE NULL
    END AS wait_resource_database_name,

    -- D. PERFORMANCE/EXECUTION METRICS
    r_wait.cpu_time AS cpu_time_ms,
    r_wait.total_elapsed_time AS total_elapsed_time_ms,
    r_wait.reads AS reads,
    r_wait.writes AS writes,
    r_wait.logical_reads AS logical_reads,
    r_wait.row_count AS row_count,
    r_wait.granted_query_memory AS granted_query_memory_pages,
    CONVERT(VARCHAR(25), SWITCHOFFSET(CAST(r_wait.start_time AS DATETIMEOFFSET), '+00:00'), 127) + 'Z' AS request_start_time,
    CONVERT(VARCHAR(25), SWITCHOFFSET(SYSDATETIMEOFFSET(), '+00:00'), 127) + 'Z' AS collection_timestamp,

    -- E. TRANSACTION CONTEXT (RCA for long-running transactions)
    r_wait.transaction_id AS transaction_id,
    r_wait.open_transaction_count AS open_transaction_count,
    r_wait.transaction_isolation_level AS transaction_isolation_level,

    -- F. PARALLEL EXECUTION DETAILS (RCA for CXPACKET waits)
    r_wait.dop AS degree_of_parallelism,
    r_wait.parallel_worker_count AS parallel_worker_count,

    -- G. SESSION CONTEXT
    s_wait.status AS session_status,
    s_wait.client_interface_name AS client_interface_name,

    -- H. PLAN HANDLE (for execution plan retrieval)
    r_wait.plan_handle AS plan_handle,

    -- I. BLOCKING DETAILS
    -- LINKING FIX: Return INT64 instead of STRING for proper numeric joins
    -- NULL when not blocked (instead of 'N/A' string) for type consistency
    CASE
        WHEN r_wait.blocking_session_id = 0 THEN NULL
        ELSE r_wait.blocking_session_id
    END AS blocking_session_id,

    ISNULL(s_blocker.login_name, 'N/A') AS blocker_login_name,
    ISNULL(s_blocker.host_name, 'N/A') AS blocker_host_name,
    ISNULL(s_blocker.program_name, 'N/A') AS blocker_program_name,
    s_blocker.status AS blocker_status,
    s_blocker.transaction_isolation_level AS blocker_isolation_level,
    s_blocker.open_transaction_count AS blocker_open_transaction_count,

    -- J. QUERY TEXT - Blocking Session
    -- Simplified to avoid SUBSTRING errors
    CASE
        WHEN r_wait.blocking_session_id = 0 THEN 'N/A'
        WHEN r_blocker.command IS NULL AND ib_blocker.event_info IS NOT NULL THEN LEFT(ib_blocker.event_info, @TextTruncateLimit)
        WHEN st_blocker.text IS NOT NULL THEN LEFT(st_blocker.text, @TextTruncateLimit)
        ELSE 'N/A'
    END AS blocking_query_statement_text,

    -- K. EXECUTION PLAN XML
    CAST(qp_wait.query_plan AS NVARCHAR(MAX)) AS execution_plan_xml,

    -- L. ENHANCED WAIT RESOURCE DECODING (OBJECT locks - table-level locks)
    SCHEMA_NAME(obj_lock.schema_id) AS wait_resource_schema_name_object,
    obj_lock.type_desc AS wait_resource_object_type,

    -- M. ENHANCED WAIT RESOURCE DECODING (INDEX locks - KEY locks, row-level locks)
    -- Cannot reliably decode KEY locks across databases without dynamic SQL
    NULL AS wait_resource_schema_name_index,
    NULL AS wait_resource_table_name_index,
    NULL AS wait_resource_index_name,
    NULL AS wait_resource_index_type

FROM
    sys.dm_exec_requests AS r_wait
INNER JOIN
    sys.dm_exec_sessions AS s_wait ON s_wait.session_id = r_wait.session_id
CROSS APPLY
    sys.dm_exec_sql_text(r_wait.sql_handle) AS st_wait
OUTER APPLY
    sys.dm_exec_query_plan(r_wait.plan_handle) AS qp_wait
LEFT JOIN
    sys.dm_exec_requests AS r_blocker ON r_wait.blocking_session_id = r_blocker.session_id
LEFT JOIN
    sys.dm_exec_sessions AS s_blocker ON r_wait.blocking_session_id = s_blocker.session_id
OUTER APPLY
    sys.dm_exec_sql_text(r_blocker.sql_handle) AS st_blocker
OUTER APPLY
    sys.dm_exec_input_buffer(r_wait.blocking_session_id, NULL) AS ib_blocker

-- Enhanced wait resource decoding: OBJECT locks (table-level locks)
-- Format: "OBJECT: <database_id>:<object_id>:<lock_type>"
LEFT JOIN sys.objects obj_lock ON
    r_wait.wait_resource LIKE 'OBJECT:%%'
    AND CHARINDEX(':', r_wait.wait_resource, 8) > 0  -- Safety: Ensure 2nd colon exists
    AND CHARINDEX(':', r_wait.wait_resource, CHARINDEX(':', r_wait.wait_resource, 8) + 1) > 0  -- Safety: Ensure 3rd colon exists
    AND TRY_CAST(
        SUBSTRING(
            r_wait.wait_resource,
            CHARINDEX(':', r_wait.wait_resource, 8) + 1,
            CASE
                WHEN CHARINDEX(':', r_wait.wait_resource, CHARINDEX(':', r_wait.wait_resource, 8) + 1) > 0
                THEN CHARINDEX(':', r_wait.wait_resource, CHARINDEX(':', r_wait.wait_resource, 8) + 1) - CHARINDEX(':', r_wait.wait_resource, 8) - 1
                ELSE LEN(r_wait.wait_resource) - CHARINDEX(':', r_wait.wait_resource, 8)
            END
        ) AS INT
    ) = obj_lock.object_id
    AND r_wait.database_id = DB_ID(DB_NAME(r_wait.database_id))

-- Enhanced wait resource decoding: INDEX locks (KEY locks - row-level locks)
-- Note: Cannot reliably resolve INDEX/KEY locks across databases without dynamic SQL
-- KEY lock format: "KEY: database_id:hobt_id (key_hash)"
-- We can only extract database_id and hobt_id, but cannot JOIN to sys.partitions cross-database
-- So we'll leave these as NULL for now (schema_name, table_name, index_name, index_type)
-- The wait_resource_description already shows the parsed HOBT and database info

WHERE
    r_wait.session_id > 50
    AND r_wait.database_id > 4
    AND r_wait.wait_type IS NOT NULL
    AND r_wait.plan_handle IS NOT NULL  -- Required for execution plan retrieval
    AND r_wait.query_hash IS NOT NULL  -- Filter out queries without query_hash (PREEMPTIVE waits, system queries)
    AND r_wait.total_elapsed_time >= @ElapsedTimeThresholdMs  -- Filter by elapsed time threshold
    %s  -- Placeholder for additional query_hash IN filter (injected from Go code)
ORDER BY
    r_wait.total_elapsed_time DESC  -- Sort by slowest executions first (not wait_time)
OPTION (RECOMPILE);  -- OPTIMIZED: Recompile for current parameter values`



// ExecutionStatsForActivePlanHandleQuery retrieves execution statistics for a specific plan_handle
// from sys.dm_exec_query_stats - used for active running queries
// This provides historical performance data for the exact plan currently executing
// Parameters: plan_handle (hex string with 0x prefix, e.g., '0x060005...')
// Returns: Execution statistics WITHOUT XML execution plan (plan already available in ActiveRunningQueriesQuery)
const ExecutionStatsForActivePlanHandleQuery = `
DECLARE @PlanHandle VARBINARY(MAX) = CONVERT(VARBINARY(MAX), %s, 1);

SELECT
    qs.plan_handle,
    qs.query_hash AS query_id,
    qs.query_plan_hash,
    CONVERT(VARCHAR(25), SWITCHOFFSET(CAST(qs.last_execution_time AS DATETIMEOFFSET), '+00:00'), 127) + 'Z' AS last_execution_time,
    CONVERT(VARCHAR(25), SWITCHOFFSET(CAST(qs.creation_time AS DATETIMEOFFSET), '+00:00'), 127) + 'Z' AS creation_time,
    qs.execution_count,
    qs.total_elapsed_time / 1000.0 AS total_elapsed_time_ms,
    (qs.total_elapsed_time / qs.execution_count) / 1000.0 AS avg_elapsed_time_ms,
    qs.min_elapsed_time / 1000.0 AS min_elapsed_time_ms,
    qs.max_elapsed_time / 1000.0 AS max_elapsed_time_ms,
    qs.last_elapsed_time / 1000.0 AS last_elapsed_time_ms,
    qs.total_worker_time / 1000.0 AS total_worker_time_ms,
    (qs.total_worker_time / qs.execution_count) / 1000.0 AS avg_worker_time_ms,
    qs.total_logical_reads,
    qs.total_logical_writes,
    qs.total_physical_reads,
    (qs.total_logical_reads / qs.execution_count) AS avg_logical_reads,
    (qs.total_logical_writes / qs.execution_count) AS avg_logical_writes,
    (qs.total_physical_reads / qs.execution_count) AS avg_physical_reads,
    (qs.total_rows / qs.execution_count) AS avg_rows,
    qs.last_grant_kb,
    qs.last_used_grant_kb,
    qs.min_grant_kb,
    qs.max_grant_kb,
    qs.last_spills,
    qs.max_spills,
    qs.last_dop,
    qs.min_dop,
    qs.max_dop
FROM sys.dm_exec_query_stats qs
WHERE
    qs.plan_handle = @PlanHandle
OPTION (RECOMPILE);`

