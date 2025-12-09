// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

// ThreadPoolHealth represents SQL Server thread pool health and worker thread utilization metrics
type ThreadPoolHealth struct {
	SQLHostname                   *string  `db:"sql_hostname" metric_name:"sql_hostname" source_type:"attribute"`
	HealthStatus                  *string  `db:"health_status" metric_name:"health_status" source_type:"attribute"`
	CurrentlyWaitingForThreadpool *int64   `db:"currently_waiting_for_threadpool" metric_name:"sqlserver.threadpool.waiting_tasks" source_type:"gauge"`
	TotalWorkQueueCount           *int64   `db:"total_work_queue_count" metric_name:"sqlserver.threadpool.work_queue_count" source_type:"gauge"`
	RunningWorkers                *int64   `db:"running_workers" metric_name:"sqlserver.threadpool.running_workers" source_type:"gauge"`
	MaxConfiguredWorkers          *int64   `db:"max_configured_workers" metric_name:"sqlserver.threadpool.max_workers" source_type:"gauge"`
	WorkerUtilizationPercent      *float64 `db:"worker_utilization_percent" metric_name:"sqlserver.threadpool.utilization_percent" source_type:"gauge"`
	TotalCurrentTasks             *int64   `db:"total_current_tasks" metric_name:"sqlserver.threadpool.current_tasks" source_type:"gauge"`
	TotalRunnableTasks            *int64   `db:"total_runnable_tasks" metric_name:"sqlserver.threadpool.runnable_tasks" source_type:"gauge"`
	CollectionTimestamp           *string  `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
}

// TempDBContention represents TempDB page latch contention and allocation wait metrics
type TempDBContention struct {
	SQLHostname         *string  `db:"sql_hostname" metric_name:"sql_hostname" source_type:"attribute"`
	TempDBHealthStatus  *string  `db:"tempdb_health_status" metric_name:"tempdb_health_status" source_type:"attribute"`
	CurrentWaiters      *int64   `db:"current_waiters" metric_name:"sqlserver.tempdb.current_waiters" source_type:"gauge"`
	PagelatchWaits      *int64   `db:"pagelatch_waits" metric_name:"sqlserver.tempdb.pagelatch_waits_ms" source_type:"gauge"`
	AllocationWaits     *int64   `db:"allocation_waits" metric_name:"sqlserver.tempdb.allocation_waits_ms" source_type:"gauge"`
	TempDBDataFileCount *int64   `db:"tempdb_data_file_count" metric_name:"sqlserver.tempdb.data_file_count" source_type:"gauge"`
	TempDBTotalSizeMB   *float64 `db:"tempdb_total_size_mb" metric_name:"sqlserver.tempdb.total_size_mb" source_type:"gauge"`
	CollectionTimestamp *string  `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
}
