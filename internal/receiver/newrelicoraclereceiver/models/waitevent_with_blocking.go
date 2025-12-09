package models

import (
	"database/sql"
	"time"
)

// WaitEventWithBlocking represents a unified record combining wait events and blocking session information
// This model is populated by the combined GetWaitEventsAndBlockingSQL query
// Field order matches SQL column order from GetWaitEventsAndBlockingSQL
type WaitEventWithBlocking struct {
	// 1. COLLECTION_TIMESTAMP
	CollectionTimestamp sql.NullTime
	// 2. cdb_name (Container Database name)
	CDBName sql.NullString
	// 3. database_name (PDB name)
	DatabaseName sql.NullString

	// Session identification (from waiting/blocked session)
	// 4. username
	Username sql.NullString
	// 5. sid
	SID sql.NullInt64
	// 6. serial#
	Serial sql.NullInt64
	// 7. status
	Status sql.NullString
	// 8. state
	State sql.NullString

	// SQL identification
	// 9. sql_id
	SQLID sql.NullString
	// 10. SQL_CHILD_NUMBER
	SQLChildNumber sql.NullInt64

	// Wait event information
	// 11. wait_class
	WaitClass sql.NullString
	// 12. event
	Event sql.NullString
	// 13. wait_time_ms (milliseconds the session has been waiting - general wait time)
	WaitTimeMs sql.NullFloat64
	// SQL execution context
	// 14. SQL_EXEC_START
	SQLExecStart sql.NullTime
	// 15. SQL_EXEC_ID
	SQLExecID sql.NullInt64

	// Session context
	// 16. PROGRAM
	Program sql.NullString
	// 17. MACHINE
	Machine sql.NullString

	// Object being waited on
	// 18. ROW_WAIT_OBJ#
	RowWaitObj sql.NullInt64
	// 19. OWNER
	Owner sql.NullString
	// 20. OBJECT_NAME
	ObjectName sql.NullString
	// 21. OBJECT_TYPE
	ObjectType sql.NullString
	// 22. ROW_WAIT_FILE#
	RowWaitFile sql.NullInt64
	// 23. ROW_WAIT_BLOCK#
	RowWaitBlock sql.NullInt64

	// Blocking session context
	// 24. BLOCKING_SESSION_STATUS
	BlockingSessionStatus sql.NullString
	// 25. immediate_blocker_sid
	ImmediateBlockerSID sql.NullInt64
	// 26. FINAL_BLOCKING_SESSION_STATUS
	FinalBlockingSessionStatus sql.NullString
	// 27. final_blocker_sid
	FinalBlockerSID sql.NullInt64
	// Final blocker details
	// 28. final_blocker_user
	FinalBlockerUser sql.NullString
	// 29. final_blocker_serial
	FinalBlockerSerial sql.NullInt64
	// 30. final_blocker_query_id
	FinalBlockerQueryID sql.NullString
	// 31. final_blocker_query_text
	FinalBlockerQueryText sql.NullString
}

// ========================================
// Getter methods for wait event fields
// ========================================

func (w *WaitEventWithBlocking) GetUsername() string {
	if w.Username.Valid {
		return w.Username.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetStatus() string {
	if w.Status.Valid {
		return w.Status.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetState() string {
	if w.State.Valid {
		return w.State.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetQueryID() string {
	if w.SQLID.Valid {
		return w.SQLID.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetWaitCategory() string {
	if w.WaitClass.Valid {
		return w.WaitClass.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetWaitEventName() string {
	if w.Event.Valid {
		return w.Event.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetProgram() string {
	if w.Program.Valid {
		return w.Program.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetMachine() string {
	if w.Machine.Valid {
		return w.Machine.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetObjectOwner() string {
	if w.Owner.Valid {
		return w.Owner.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetObjectNameWaitedOn() string {
	if w.ObjectName.Valid {
		return w.ObjectName.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetObjectTypeWaitedOn() string {
	if w.ObjectType.Valid {
		return w.ObjectType.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetSID() int64 {
	if w.SID.Valid {
		return w.SID.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetSerial() int64 {
	if w.Serial.Valid {
		return w.Serial.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetSQLChildNumber() int64 {
	if w.SQLChildNumber.Valid {
		return w.SQLChildNumber.Int64
	}
	return -1 // Return -1 for NULL to distinguish from legitimate child_number 0
}

func (w *WaitEventWithBlocking) GetSQLExecID() int64 {
	if w.SQLExecID.Valid {
		return w.SQLExecID.Int64
	}
	return -1
}

// GetCurrentWaitMs returns wait time in milliseconds (general wait time)
func (w *WaitEventWithBlocking) GetCurrentWaitMs() float64 {
	if w.WaitTimeMs.Valid {
		return w.WaitTimeMs.Float64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetLockedObjectID() int64 {
	if w.RowWaitObj.Valid {
		return w.RowWaitObj.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetLockedFileID() int64 {
	if w.RowWaitFile.Valid {
		return w.RowWaitFile.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetLockedBlockID() int64 {
	if w.RowWaitBlock.Valid {
		return w.RowWaitBlock.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetSQLExecStart() time.Time {
	if w.SQLExecStart.Valid {
		return w.SQLExecStart.Time
	}
	return time.Time{}
}

func (w *WaitEventWithBlocking) GetCollectionTimestamp() time.Time {
	if w.CollectionTimestamp.Valid {
		return w.CollectionTimestamp.Time
	}
	return time.Time{}
}

func (w *WaitEventWithBlocking) GetDatabaseName() string {
	if w.DatabaseName.Valid {
		return w.DatabaseName.String
	}
	return ""
}

// GetCDBName returns the Container Database name
func (w *WaitEventWithBlocking) GetCDBName() string {
	if w.CDBName.Valid {
		return w.CDBName.String
	}
	return ""
}

// ========================================
// Getter methods for blocking fields
// ========================================

func (w *WaitEventWithBlocking) GetBlockingSessionStatus() string {
	if w.BlockingSessionStatus.Valid {
		return w.BlockingSessionStatus.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetImmediateBlockerSID() int64 {
	if w.ImmediateBlockerSID.Valid {
		return w.ImmediateBlockerSID.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetFinalBlockingSessionStatus() string {
	if w.FinalBlockingSessionStatus.Valid {
		return w.FinalBlockingSessionStatus.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetFinalBlockerSID() int64 {
	if w.FinalBlockerSID.Valid {
		return w.FinalBlockerSID.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetFinalBlockerUser() string {
	if w.FinalBlockerUser.Valid {
		return w.FinalBlockerUser.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetFinalBlockerSerial() int64 {
	if w.FinalBlockerSerial.Valid {
		return w.FinalBlockerSerial.Int64
	}
	return 0
}

func (w *WaitEventWithBlocking) GetFinalBlockerQueryID() string {
	if w.FinalBlockerQueryID.Valid {
		return w.FinalBlockerQueryID.String
	}
	return ""
}

func (w *WaitEventWithBlocking) GetFinalBlockerQueryText() string {
	if w.FinalBlockerQueryText.Valid {
		return w.FinalBlockerQueryText.String
	}
	return ""
}

// ========================================
// Validation methods
// ========================================

func (w *WaitEventWithBlocking) HasValidQueryID() bool {
	return w.SQLID.Valid
}

func (w *WaitEventWithBlocking) HasValidWaitEventName() bool {
	return w.Event.Valid
}

func (w *WaitEventWithBlocking) HasValidCurrentWaitSeconds() bool {
	return w.WaitTimeMs.Valid && w.WaitTimeMs.Float64 > 0
}

func (w *WaitEventWithBlocking) IsValidForMetrics() bool {
	return w.HasValidQueryID() && w.HasValidWaitEventName() && w.HasValidCurrentWaitSeconds()
}

// IsBlocked checks if this session is currently blocked by another session
func (w *WaitEventWithBlocking) IsBlocked() bool {
	return w.FinalBlockerSID.Valid && w.FinalBlockerSID.Int64 > 0
}

// HasBlockingInfo checks if blocking information is available
func (w *WaitEventWithBlocking) HasBlockingInfo() bool {
	return w.IsBlocked() && w.FinalBlockerQueryID.Valid
}
