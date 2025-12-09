// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

const InstanceBufferPoolQuery = `SELECT
	Count_big(*) * (8*1024) AS instance_buffer_pool_size
	FROM sys.dm_os_buffer_descriptors WITH (nolock)
	WHERE database_id <> 32767 -- ResourceDB`

// InstanceMemoryDefinitions query for standard SQL Server instance memory metrics
const InstanceMemoryDefinitions = `SELECT
		Max(sys_mem.total_physical_memory_kb * 1024.0) AS total_physical_memory,
		Max(sys_mem.available_physical_memory_kb * 1024.0) AS available_physical_memory,
		(Max(proc_mem.physical_memory_in_use_kb) / (Max(sys_mem.total_physical_memory_kb) * 1.0)) * 100 AS memory_utilization
		FROM sys.dm_os_process_memory proc_mem,
		  sys.dm_os_sys_memory sys_mem,
		  sys.dm_os_performance_counters perf_count WHERE object_name LIKE '%:Memory Manager%'`

// InstanceStatsQuery returns comprehensive instance statistics
const InstanceStatsQuery = `SELECT
        t1.cntr_value AS sql_compilations,
        t2.cntr_value AS sql_recompilations,
        t3.cntr_value AS user_connections,
        t4.cntr_value AS lock_wait_time_ms,
        t5.cntr_value AS page_splits_sec,
        t6.cntr_value AS checkpoint_pages_sec,
        t7.cntr_value AS deadlocks_sec,
        t8.cntr_value AS user_errors,
        t9.cntr_value AS kill_connection_errors,
        t10.cntr_value AS batch_request_sec,
        (t11.cntr_value * 1000.0) AS page_life_expectancy_ms,
        t12.cntr_value AS transactions_sec,
        t13.cntr_value AS forced_parameterizations_sec
        FROM 
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'SQL Compilations/sec') t1,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'SQL Re-Compilations/sec') t2,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'User Connections') t3,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Lock Wait Time (ms)' AND instance_name = '_Total') t4,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Page Splits/sec') t5,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Checkpoint pages/sec') t6,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Number of Deadlocks/sec' AND instance_name = '_Total') t7,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE object_name LIKE '%SQL Errors%' AND instance_name = 'User Errors') t8,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE object_name LIKE '%SQL Errors%' AND instance_name LIKE 'Kill Connection Errors%') t9,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Batch Requests/sec') t10,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t11,
        (SELECT Sum(cntr_value) AS cntr_value FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Transactions/sec') t12,
        (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Forced Parameterizations/sec') t13`

const BufferPoolHitPercentMetricsQuery = `SELECT (a.cntr_value * 1.0 / b.cntr_value) * 100.0 AS buffer_pool_hit_percent
		FROM sys.dm_os_performance_counters 
		a JOIN (SELECT cntr_value, OBJECT_NAME FROM sys.dm_os_performance_counters WHERE counter_name = 'Buffer cache hit ratio base') 
		b ON  a.OBJECT_NAME = b.OBJECT_NAME 
		WHERE a.counter_name = 'Buffer cache hit ratio'`

const InstanceProcessCountMetricsQuery = `SELECT
		Max(CASE WHEN sessions.status = 'preconnect' THEN counts ELSE 0 END) AS preconnect,
		Max(CASE WHEN sessions.status = 'background' THEN counts ELSE 0 END) AS background,
		Max(CASE WHEN sessions.status = 'dormant' THEN counts ELSE 0 END) AS dormant,
		Max(CASE WHEN sessions.status = 'runnable' THEN counts ELSE 0 END) AS runnable,
		Max(CASE WHEN sessions.status = 'suspended' THEN counts ELSE 0 END) AS suspended,
		Max(CASE WHEN sessions.status = 'running' THEN counts ELSE 0 END) AS running,
		Max(CASE WHEN sessions.status = 'blocked' THEN counts ELSE 0 END) AS blocked,
		Max(CASE WHEN sessions.status = 'sleeping' THEN counts ELSE 0 END) AS sleeping
		FROM (SELECT status, Count(*) counts FROM (
			SELECT CASE WHEN req.status IS NOT NULL THEN
				CASE WHEN req.blocking_session_id <> 0 THEN 'blocked' ELSE req.status END
			  ELSE sess.status END status, req.blocking_session_id
			FROM sys.dm_exec_sessions sess
			LEFT JOIN sys.dm_exec_requests req
			ON sess.session_id = req.session_id
			WHERE sess.session_id > 50 ) statuses
		  GROUP BY status) sessions`

const RunnableTasksMetricsQuery = `SELECT Sum(runnable_tasks_count) AS runnable_tasks_count
		FROM sys.dm_os_schedulers
		WHERE   scheduler_id < 255 AND [status] = 'VISIBLE ONLINE'`

const InstanceActiveConnectionsMetricsQuery = `SELECT Count(dbid) AS instance_active_connections FROM sys.sysprocesses WITH (nolock) WHERE dbid > 0`

const InstanceDiskMetricsQuery = `SELECT Sum(total_bytes) AS total_disk_space FROM (
			SELECT DISTINCT
			dovs.volume_mount_point,
			dovs.available_bytes available_bytes,
			dovs.total_bytes total_bytes
			FROM sys.master_files mf WITH (nolock)
			CROSS apply sys.dm_os_volume_stats(mf.database_id, mf.file_id) dovs
			) drives`

// New instance metrics queries for enhanced performance monitoring

const InstanceTargetMemoryQuery = `SELECT 
    cntr_value AS target_server_memory_kb
    FROM sys.dm_os_performance_counters WITH (nolock) 
    WHERE counter_name = 'Target Server Memory (KB)'`

const InstancePerformanceRatiosQuery = `SELECT
    CASE WHEN t2.cntr_value > 0 THEN CAST(t1.cntr_value AS FLOAT) / CAST(t2.cntr_value AS FLOAT) ELSE 0 END AS compilations_per_batch,
    CASE WHEN t4.cntr_value > 0 THEN CAST(t3.cntr_value AS FLOAT) / CAST(t4.cntr_value AS FLOAT) ELSE 0 END AS page_splits_per_batch
    FROM 
    (SELECT cntr_value FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'SQL Compilations/sec') t1,
    (SELECT cntr_value FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Batch Requests/sec') t2,
    (SELECT cntr_value FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Page Splits/sec') t3,
    (SELECT cntr_value FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Batch Requests/sec') t4`

const InstanceIndexMetricsQuery = `SELECT
    cntr_value AS full_scans_per_sec
    FROM sys.dm_os_performance_counters WITH (nolock)
    WHERE counter_name = 'Full Scans/sec'`

// Provides lock timeouts per second - removed avg wait time due to incorrect calculation
const InstanceLockMetricsQuery = `SELECT
    cntr_value AS lock_timeouts_per_sec
    FROM sys.dm_os_performance_counters WITH (nolock) 
    WHERE counter_name = 'Lock Timeouts/sec'`

// Azure SQL Database doesn't have access to sys.master_files or sys.dm_os_volume_stats
// We use database-level disk information instead
const InstanceDiskMetricsQueryAzureSQL = `SELECT 
	CAST(SUM(size) * 8.0 * 1024 AS BIGINT) AS total_disk_space
FROM sys.database_files
WHERE state = 0`

// Azure SQL Managed Instance supports most sys.master_files functionality
const InstanceDiskMetricsQueryAzureMI = `SELECT Sum(total_bytes) AS total_disk_space FROM (
			SELECT DISTINCT
			dovs.volume_mount_point,
			dovs.available_bytes available_bytes,
			dovs.total_bytes total_bytes
			FROM sys.master_files mf WITH (nolock)
			CROSS apply sys.dm_os_volume_stats(mf.database_id, mf.file_id) dovs
			) drives`
