// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// ThreadPoolHealthQuery monitors SQL Server thread pool exhaustion and worker thread utilization
// This query combines multiple DMVs to provide real-time health status of the thread pool
const ThreadPoolHealthQuery = `
WITH CurrentWaits AS (
    -- Counts tasks ACTUALLY waiting for a thread RIGHT NOW
    SELECT
        COUNT(*) AS currently_waiting_for_threadpool
    FROM sys.dm_os_waiting_tasks
    WHERE wait_type = 'THREADPOOL'
),
SchedulerStats AS (
    -- Checks the current state of the schedulers, where work is queued
    SELECT
        SUM(work_queue_count) AS total_work_queue_count,
        SUM(runnable_tasks_count) AS total_runnable_tasks,
        SUM(current_tasks_count) AS total_current_tasks
    FROM sys.dm_os_schedulers
    WHERE scheduler_id < 1048576 -- Filters out internal/hidden schedulers
),
WorkerAndConfigStats AS (
    -- Gets the current state of all workers and the configured server limit
    SELECT
        (SELECT max_workers_count FROM sys.dm_os_sys_info) AS max_configured_workers,
        COUNT(*) AS total_workers,
        SUM(CASE WHEN state = 'RUNNING' THEN 1 ELSE 0 END) AS running_workers,
        SUM(CASE WHEN state NOT IN ('RUNNING', 'RUNNABLE') THEN 1 ELSE 0 END) AS suspended_or_sleeping_workers
    FROM sys.dm_os_workers
)
SELECT
    @@SERVERNAME AS sql_hostname,
    -- Health Status: The most important column.
    CASE
        WHEN cw.currently_waiting_for_threadpool > 0 OR ss.total_work_queue_count > 0
        THEN 'CRITICAL: Thread Exhaustion Detected'
        WHEN wc.running_workers >= (wc.max_configured_workers * 0.9)
        THEN 'WARNING: Approaching Max Worker Limit'
        ELSE 'HEALTHY'
    END AS health_status,
    -- Real-time Exhaustion Metrics (The most critical indicators)
    cw.currently_waiting_for_threadpool,
    ss.total_work_queue_count,
    -- Utilization and Load Metrics
    wc.running_workers,
    wc.max_configured_workers,
    CAST((wc.running_workers * 100.0) / wc.max_configured_workers AS DECIMAL(5, 2)) AS worker_utilization_percent,
    ss.total_current_tasks, -- All tasks currently assigned to schedulers
    ss.total_runnable_tasks, -- Tasks ready to run but waiting for CPU time on the scheduler
    CONVERT(VARCHAR(25), SWITCHOFFSET(SYSDATETIMEOFFSET(), '+00:00'), 127) + 'Z' AS collection_timestamp
FROM CurrentWaits cw
CROSS JOIN SchedulerStats ss
CROSS JOIN WorkerAndConfigStats wc;
`
