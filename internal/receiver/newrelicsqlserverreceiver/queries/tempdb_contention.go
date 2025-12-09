// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// TempDBContentionQuery monitors TempDB page latch contention and allocation waits
// This query identifies TempDB bottlenecks using cumulative statistics for reliability
const TempDBContentionQuery = `
WITH TempDBMetrics AS (
    SELECT 
        @@SERVERNAME AS sql_hostname,
        -- Current waiters using cumulative stats (more reliable than real-time)
        (SELECT COUNT(*) 
         FROM sys.dm_os_wait_stats 
         WHERE wait_type IN ('PAGELATCH_IO', 'PAGELATCH_SH', 'PAGELATCH_EX', 'PAGELATCH_UP')
             AND waiting_tasks_count > 0) AS current_waiters,
        -- Total page latch wait time since server start
        (SELECT ISNULL(SUM(wait_time_ms), 0) 
         FROM sys.dm_os_wait_stats 
         WHERE wait_type IN ('PAGELATCH_IO', 'PAGELATCH_SH', 'PAGELATCH_EX', 'PAGELATCH_UP')) AS pagelatch_waits,
        -- Allocation-related waits (compatible with older SQL Server versions)
        (SELECT ISNULL(SUM(wait_time_ms), 0) 
         FROM sys.dm_os_wait_stats
         WHERE wait_type LIKE '%GAM%' OR wait_type LIKE '%ALLOCATION%' OR wait_type LIKE '%SGAM%' OR wait_type LIKE '%PFS%') AS allocation_waits,
        -- TempDB configuration info
        (SELECT COUNT(*) 
         FROM sys.master_files 
         WHERE database_id = 2 AND type = 0) AS tempdb_data_file_count,
        -- TempDB space usage
        (SELECT SUM(size) * 8.0 / 1024 
         FROM sys.master_files 
         WHERE database_id = 2 AND type = 0) AS tempdb_total_size_mb
)
SELECT 
    sql_hostname,
    CASE 
        WHEN current_waiters > 10 THEN 'CRITICAL: High TempDB Contention'
        WHEN current_waiters > 5 THEN 'WARNING: TempDB Contention'
        WHEN pagelatch_waits > 100000 THEN 'WARNING: Historical TempDB Pressure'
        ELSE 'HEALTHY'
    END AS tempdb_health_status,
    current_waiters,
    pagelatch_waits,
    allocation_waits,
    tempdb_data_file_count,
    tempdb_total_size_mb,
    CONVERT(VARCHAR(25), SWITCHOFFSET(SYSDATETIMEOFFSET(), '+00:00'), 127) + 'Z' AS collection_timestamp
FROM TempDBMetrics;
`
