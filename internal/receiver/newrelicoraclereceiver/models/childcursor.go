// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"database/sql"
	"time"
)

// ChildCursor represents a child cursor from V$SQL with average execution statistics
type ChildCursor struct {
	CollectionTimestamp sql.NullTime
	CDBName             sql.NullString // Container Database name
	DatabaseName        sql.NullString // PDB (Pluggable Database) name
	SQLID               sql.NullString
	ChildNumber         sql.NullInt64
	PlanHashValue       sql.NullInt64   // Plan hash value - identifies the execution plan
	AvgCPUTimeMs        sql.NullFloat64 // Average CPU time per execution (milliseconds) - can have decimals
	AvgElapsedTimeMs    sql.NullFloat64 // Average elapsed time per execution (milliseconds) - can have decimals
	AvgIOWaitTimeMs     sql.NullFloat64 // Average I/O wait time per execution (milliseconds) - can have decimals
	AvgDiskReads        sql.NullFloat64 // Average disk reads per execution - can have decimals
	AvgBufferGets       sql.NullFloat64 // Average buffer gets per execution - can have decimals
	Executions          sql.NullInt64   // Total number of executions
	Invalidations       sql.NullInt64
	FirstLoadTime       sql.NullString
	LastLoadTime        sql.NullString
}

// GetCollectionTimestamp returns the collection timestamp as time.Time
func (cc *ChildCursor) GetCollectionTimestamp() time.Time {
	if cc.CollectionTimestamp.Valid {
		return cc.CollectionTimestamp.Time
	}
	return time.Time{}
}

// GetCDBName returns the Container Database name as a string, empty if null
func (cc *ChildCursor) GetCDBName() string {
	if cc.CDBName.Valid {
		return cc.CDBName.String
	}
	return ""
}

// GetDatabaseName returns the database name as a string, empty if null
func (cc *ChildCursor) GetDatabaseName() string {
	if cc.DatabaseName.Valid {
		return cc.DatabaseName.String
	}
	return ""
}

// GetSQLID returns the SQL ID as a string, empty if null
func (cc *ChildCursor) GetSQLID() string {
	if cc.SQLID.Valid {
		return cc.SQLID.String
	}
	return ""
}

// GetChildNumber returns the child number as int64, 0 if null
func (cc *ChildCursor) GetChildNumber() int64 {
	if cc.ChildNumber.Valid {
		return cc.ChildNumber.Int64
	}
	return 0
}

// GetPlanHashValue returns the plan hash value as int64, 0 if null
func (cc *ChildCursor) GetPlanHashValue() int64 {
	if cc.PlanHashValue.Valid {
		return cc.PlanHashValue.Int64
	}
	return 0
}

// GetCPUTime returns the average CPU time in milliseconds as float64, 0 if null
func (cc *ChildCursor) GetCPUTime() float64 {
	if cc.AvgCPUTimeMs.Valid {
		return cc.AvgCPUTimeMs.Float64
	}
	return 0
}

// GetElapsedTime returns the average elapsed time in milliseconds as float64, 0 if null
func (cc *ChildCursor) GetElapsedTime() float64 {
	if cc.AvgElapsedTimeMs.Valid {
		return cc.AvgElapsedTimeMs.Float64
	}
	return 0
}

// GetUserIOWaitTime returns the average user IO wait time in milliseconds as float64, 0 if null
func (cc *ChildCursor) GetUserIOWaitTime() float64 {
	if cc.AvgIOWaitTimeMs.Valid {
		return cc.AvgIOWaitTimeMs.Float64
	}
	return 0
}

// GetExecutions returns the number of executions as int64, 0 if null
func (cc *ChildCursor) GetExecutions() int64 {
	if cc.Executions.Valid {
		return cc.Executions.Int64
	}
	return 0
}

// GetDiskReads returns the average number of disk reads as float64, 0 if null
func (cc *ChildCursor) GetDiskReads() float64 {
	if cc.AvgDiskReads.Valid {
		return cc.AvgDiskReads.Float64
	}
	return 0
}

// GetBufferGets returns the average number of buffer gets as float64, 0 if null
func (cc *ChildCursor) GetBufferGets() float64 {
	if cc.AvgBufferGets.Valid {
		return cc.AvgBufferGets.Float64
	}
	return 0
}

// GetInvalidations returns the number of invalidations as int64, 0 if null
func (cc *ChildCursor) GetInvalidations() int64 {
	if cc.Invalidations.Valid {
		return cc.Invalidations.Int64
	}
	return 0
}

// GetFirstLoadTime returns the first load time as a string, empty if null
func (cc *ChildCursor) GetFirstLoadTime() string {
	if cc.FirstLoadTime.Valid {
		return cc.FirstLoadTime.String
	}
	return ""
}

// GetLastLoadTime returns the last load time as a string, empty if null
func (cc *ChildCursor) GetLastLoadTime() string {
	if cc.LastLoadTime.Valid {
		return cc.LastLoadTime.String
	}
	return ""
}

// HasValidIdentifier checks if the child cursor has valid SQL_ID and child_number
func (cc *ChildCursor) HasValidIdentifier() bool {
	return cc.SQLID.Valid && cc.ChildNumber.Valid
}
