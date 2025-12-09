// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

// LockResourceSummary represents what resources are being locked (what's being locked)
// This gives visibility into lock granularity patterns across different resource types
type LockResourceSummary struct {
	DatabaseName        *string `db:"database_name" metric_name:"sqlserver.lock.resource.database" source_type:"attribute"`
	TotalActiveLocks    *int64  `db:"total_active_locks" metric_name:"sqlserver.lock.resource.total" source_type:"gauge" unit:"1"`
	TableLocks          *int64  `db:"table_locks" metric_name:"sqlserver.lock.resource.table" source_type:"gauge" unit:"1"`
	PageLocks           *int64  `db:"page_locks" metric_name:"sqlserver.lock.resource.page" source_type:"gauge" unit:"1"`
	RowLocks            *int64  `db:"row_locks" metric_name:"sqlserver.lock.resource.row" source_type:"gauge" unit:"1"`
	KeyLocks            *int64  `db:"key_locks" metric_name:"sqlserver.lock.resource.key" source_type:"gauge" unit:"1"`
	ExtentLocks         *int64  `db:"extent_locks" metric_name:"sqlserver.lock.resource.extent" source_type:"gauge" unit:"1"`
	DatabaseLocks       *int64  `db:"database_locks" metric_name:"sqlserver.lock.resource.database_level" source_type:"gauge" unit:"1"`
	FileLocks           *int64  `db:"file_locks" metric_name:"sqlserver.lock.resource.file" source_type:"gauge" unit:"1"`
	ApplicationLocks    *int64  `db:"application_locks" metric_name:"sqlserver.lock.resource.application" source_type:"gauge" unit:"1"`
	MetadataLocks       *int64  `db:"metadata_locks" metric_name:"sqlserver.lock.resource.metadata" source_type:"gauge" unit:"1"`
	HobtLocks           *int64  `db:"hobt_locks" metric_name:"sqlserver.lock.resource.hobt" source_type:"gauge" unit:"1"`
	AllocationUnitLocks *int64  `db:"allocation_unit_locks" metric_name:"sqlserver.lock.resource.allocation_unit" source_type:"gauge" unit:"1"`
}

// LockModeSummary represents how resources are being locked (lock modes)
// This gives visibility into lock behavior patterns and contention types
type LockModeSummary struct {
	DatabaseName               *string `db:"database_name" metric_name:"sqlserver.lock.mode.database" source_type:"attribute"`
	TotalActiveLocks           *int64  `db:"total_active_locks" metric_name:"sqlserver.lock.mode.total" source_type:"gauge" unit:"1"`
	SharedLocks                *int64  `db:"shared_locks" metric_name:"sqlserver.lock.mode.shared" source_type:"gauge" unit:"1"`
	ExclusiveLocks             *int64  `db:"exclusive_locks" metric_name:"sqlserver.lock.mode.exclusive" source_type:"gauge" unit:"1"`
	UpdateLocks                *int64  `db:"update_locks" metric_name:"sqlserver.lock.mode.update" source_type:"gauge" unit:"1"`
	IntentLocks                *int64  `db:"intent_locks" metric_name:"sqlserver.lock.mode.intent" source_type:"gauge" unit:"1"`
	SchemaLocks                *int64  `db:"schema_locks" metric_name:"sqlserver.lock.mode.schema" source_type:"gauge" unit:"1"`
	BulkUpdateLocks            *int64  `db:"bulk_update_locks" metric_name:"sqlserver.lock.mode.bulk_update" source_type:"gauge" unit:"1"`
	SharedIntentExclusiveLocks *int64  `db:"shared_intent_exclusive_locks" metric_name:"sqlserver.lock.mode.shared_intent_exclusive" source_type:"gauge" unit:"1"`
}
