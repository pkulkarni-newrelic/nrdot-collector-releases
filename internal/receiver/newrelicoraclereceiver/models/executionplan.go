// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "database/sql"

// ExecutionPlanRow represents a single row from V$SQL_PLAN
type ExecutionPlanRow struct {
	SQLID            sql.NullString
	Timestamp        sql.NullString // Date and time when the execution plan was generated (from V$SQL_PLAN.TIMESTAMP)
	TempSpace        sql.NullString // Changed to string to handle large values
	AccessPredicates sql.NullString
	Projection       sql.NullString
	Time             sql.NullString // Changed to string to handle large values
	FilterPredicates sql.NullString
	ChildNumber      sql.NullInt64
	ID               sql.NullInt64
	ParentID         sql.NullInt64
	Depth            sql.NullInt64
	Operation        sql.NullString
	Options          sql.NullString
	ObjectOwner      sql.NullString
	ObjectName       sql.NullString
	Position         sql.NullInt64
	PlanHashValue    sql.NullInt64
	Cost             sql.NullString // Changed to string to handle large values
	Cardinality      sql.NullString // Changed to string to handle large values
	Bytes            sql.NullString // Changed to string to handle large values
	CPUCost          sql.NullString // Changed to string to handle large values that exceed int64 max
	IOCost           sql.NullString // Changed to string to handle large values
}
