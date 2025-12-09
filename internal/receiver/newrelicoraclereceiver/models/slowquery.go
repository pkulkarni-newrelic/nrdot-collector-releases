package models

import "database/sql"

// SlowQuery represents a slow query record from Oracle V$SQL view
type SlowQuery struct {
	CollectionTimestamp sql.NullString  // Timestamp when query was collected from Oracle
	CDBName             sql.NullString  // Container Database (CDB) name
	DatabaseName        sql.NullString
	QueryID             sql.NullString
	SchemaName          sql.NullString
	UserName            sql.NullString // NEW: The user who parsed the statement
	ExecutionCount      sql.NullInt64
	QueryText           sql.NullString
	AvgCPUTimeMs        sql.NullFloat64
	AvgDiskReads        sql.NullFloat64
	AvgDiskWrites       sql.NullFloat64
	AvgElapsedTimeMs    sql.NullFloat64
	AvgRowsExamined     sql.NullFloat64
	AvgLockTimeMs       sql.NullFloat64
	LastActiveTime      sql.NullString
	HasFullTableScan    sql.NullString
	TotalElapsedTimeMS  sql.NullFloat64 // Used for precise delta calculation only

	// Interval-based delta metrics (calculated in-memory, not from DB)
	// These are populated by the OracleIntervalCalculator
	// NOTE: Only elapsed time has delta calculation, CPU uses historical average from DB
	IntervalAvgElapsedTimeMS *float64 // Average elapsed time in the last interval (milliseconds)
	IntervalExecutionCount   *int64   // Number of executions in the last interval
}

// GetCollectionTimestamp returns the collection timestamp as a string, empty if null
func (sq *SlowQuery) GetCollectionTimestamp() string {
	if sq.CollectionTimestamp.Valid {
		return sq.CollectionTimestamp.String
	}
	return ""
}

// GetCDBName returns the CDB name as a string, empty if null
func (sq *SlowQuery) GetCDBName() string {
	if sq.CDBName.Valid {
		return sq.CDBName.String
	}
	return ""
}

// GetDatabaseName returns the database name as a string, empty if null
func (sq *SlowQuery) GetDatabaseName() string {
	if sq.DatabaseName.Valid {
		return sq.DatabaseName.String
	}
	return ""
}

// GetQueryID returns the query ID as a string, empty if null
func (sq *SlowQuery) GetQueryID() string {
	if sq.QueryID.Valid {
		return sq.QueryID.String
	}
	return ""
}

// GetSchemaName returns the schema name as a string, empty if null
func (sq *SlowQuery) GetSchemaName() string {
	if sq.SchemaName.Valid {
		return sq.SchemaName.String
	}
	return ""
}

// GetQueryText returns the query text as a string, empty if null
func (sq *SlowQuery) GetQueryText() string {
	if sq.QueryText.Valid {
		return sq.QueryText.String
	}
	return ""
}

// GetUserName returns the username as a string, empty if null
func (sq *SlowQuery) GetUserName() string {
	if sq.UserName.Valid {
		return sq.UserName.String
	}
	return ""
}

// GetLastActiveTime returns the last active time as a string, empty if null
func (sq *SlowQuery) GetLastActiveTime() string {
	if sq.LastActiveTime.Valid {
		return sq.LastActiveTime.String
	}
	return ""
}

// GetHasFullTableScan returns the full table scan flag as a string, empty if null
func (sq *SlowQuery) GetHasFullTableScan() string {
	if sq.HasFullTableScan.Valid {
		return sq.HasFullTableScan.String
	}
	return ""
}

// HasValidQueryID checks if the query has a valid query ID
func (sq *SlowQuery) HasValidQueryID() bool {
	return sq.QueryID.Valid
}

// HasValidElapsedTime checks if the query has a valid elapsed time
func (sq *SlowQuery) HasValidElapsedTime() bool {
	return sq.AvgElapsedTimeMs.Valid
}

// IsValidForMetrics checks if the slow query has the minimum required fields for metrics
func (sq *SlowQuery) IsValidForMetrics() bool {
	return sq.HasValidQueryID() && sq.HasValidElapsedTime()
}
