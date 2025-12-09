// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// WaitTimeMetricsQuery contains the SQL query for collecting wait time statistics
const WaitTimeMetricsQuery = `SELECT 
    wait_type, 
    wait_time_ms AS wait_time, 
    waiting_tasks_count
FROM sys.dm_os_wait_stats 
WHERE wait_time_ms != 0
ORDER BY wait_time_ms DESC`

// LatchWaitTimeMetricsQuery contains the SQL query for collecting latch-specific wait time statistics
const LatchWaitTimeMetricsQuery = `SELECT 
    wait_type, 
    wait_time_ms AS wait_time, 
    waiting_tasks_count
    FROM sys.dm_os_wait_stats WITH (nolock) 
    WHERE wait_type LIKE 'LATCH_%' 
    AND wait_time_ms >= 0
    ORDER BY wait_time_ms DESC`
