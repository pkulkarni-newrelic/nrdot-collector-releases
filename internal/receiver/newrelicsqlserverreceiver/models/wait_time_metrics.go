// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package models

// WaitTimeMetricsModel represents SQL Server wait statistics
// This struct captures wait information from sys.dm_os_wait_stats
type WaitTimeMetricsModel struct {
	WaitType          *string `db:"wait_type" metric_name:"sqlserver.wait_stats.wait_type" source_type:"attribute" description:"SQL Server wait type" unit:"1"`
	WaitTimeMs        *int64  `db:"wait_time" metric_name:"sqlserver.wait_stats.wait_time_ms" source_type:"rate" description:"Total wait time in milliseconds" unit:"ms"`
	WaitingTasksCount *int64  `db:"waiting_tasks_count" metric_name:"sqlserver.wait_stats.waiting_tasks_count" source_type:"rate" description:"Number of tasks currently waiting" unit:"1"`
}

// LatchWaitTimeMetricsModel represents SQL Server latch-specific wait statistics
// This struct captures latch wait information from sys.dm_os_wait_stats filtered for latch waits
type LatchWaitTimeMetricsModel struct {
	WaitType          *string `db:"wait_type" metric_name:"sqlserver.wait_stats.latch.wait_type" source_type:"attribute" description:"Latch wait type" unit:"1"`
	WaitTimeMs        *int64  `db:"wait_time" metric_name:"sqlserver.wait_stats.latch.wait_time_ms" source_type:"rate" description:"Latch wait time in milliseconds" unit:"ms"`
	WaitingTasksCount *int64  `db:"waiting_tasks_count" metric_name:"sqlserver.wait_stats.latch.waiting_tasks_count" source_type:"rate" description:"Number of tasks waiting on latches" unit:"1"`
}
