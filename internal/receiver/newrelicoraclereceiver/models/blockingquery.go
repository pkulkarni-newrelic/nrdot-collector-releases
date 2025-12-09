package models

import (
	"database/sql"
	"time"
)

// BlockingQuery represents a blocking query record from Oracle V$SESSION views
type BlockingQuery struct {
	CollectionTimestamp sql.NullTime
	SessionID           sql.NullInt64
	BlockedSerial       sql.NullInt64
	BlockedUser         sql.NullString
	BlockedWaitSec      sql.NullFloat64
	QueryID             sql.NullString
	SQLExecID           sql.NullInt64
	BlockingQueryText   sql.NullString
	BlockingSID         sql.NullInt64
	BlockingSerial      sql.NullInt64
	BlockingUser        sql.NullString
	BlockingQueryID     sql.NullString
	DatabaseName        sql.NullString
}

// GetBlockedUser returns the blocked user as a string, empty if null
func (bq *BlockingQuery) GetBlockedUser() string {
	if bq.BlockedUser.Valid {
		return bq.BlockedUser.String
	}
	return ""
}

// GetBlockingQueryText returns the blocking query text as a string, empty if null
func (bq *BlockingQuery) GetBlockingQueryText() string {
	if bq.BlockingQueryText.Valid {
		return bq.BlockingQueryText.String
	}
	return ""
}

// GetBlockingUser returns the blocking user as a string, empty if null
func (bq *BlockingQuery) GetBlockingUser() string {
	if bq.BlockingUser.Valid {
		return bq.BlockingUser.String
	}
	return ""
}

// GetQueryID returns the query ID as a string, empty if null
func (bq *BlockingQuery) GetQueryID() string {
	if bq.QueryID.Valid {
		return bq.QueryID.String
	}
	return ""
}

// GetBlockingQueryID returns the blocking query ID as a string, empty if null
func (bq *BlockingQuery) GetBlockingQueryID() string {
	if bq.BlockingQueryID.Valid {
		return bq.BlockingQueryID.String
	}
	return ""
}

// GetSQLExecID returns the blocked SQL execution ID as int64, -1 if null
func (bq *BlockingQuery) GetSQLExecID() int64 {
	if bq.SQLExecID.Valid {
		return bq.SQLExecID.Int64
	}
	return -1
}

// GetDatabaseName returns the database name as a string, empty if null
func (bq *BlockingQuery) GetDatabaseName() string {
	if bq.DatabaseName.Valid {
		return bq.DatabaseName.String
	}
	return ""
}

// HasValidWaitTime checks if the query has a valid wait time
func (bq *BlockingQuery) HasValidWaitTime() bool {
	return bq.BlockedWaitSec.Valid && bq.BlockedWaitSec.Float64 >= 0
}

// HasValidQueryID checks if the query has a valid query ID
func (bq *BlockingQuery) HasValidQueryID() bool {
	return bq.QueryID.Valid
}

// IsValidForMetrics checks if the blocking query has the minimum required fields for metrics
func (bq *BlockingQuery) IsValidForMetrics() bool {
	return bq.HasValidQueryID() && bq.HasValidWaitTime()
}

// GetCollectionTimestamp returns the collection timestamp, or zero time if null
func (bq *BlockingQuery) GetCollectionTimestamp() time.Time {
	if bq.CollectionTimestamp.Valid {
		return bq.CollectionTimestamp.Time
	}
	return time.Time{}
}
