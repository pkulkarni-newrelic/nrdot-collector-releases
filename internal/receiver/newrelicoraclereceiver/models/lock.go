package models

import "database/sql"

// LockCount represents lock count metrics by type and mode
type LockCount struct {
	LockType  sql.NullString
	LockMode  sql.NullString
	LockCount sql.NullInt64
}

// GetLockType returns the lock type as a string, empty if null
func (lc *LockCount) GetLockType() string {
	if lc.LockType.Valid {
		return lc.LockType.String
	}
	return "unknown"
}

// GetLockMode returns the lock mode as a string, empty if null
func (lc *LockCount) GetLockMode() string {
	if lc.LockMode.Valid {
		return lc.LockMode.String
	}
	return "unknown"
}

// GetCount returns the lock count, 0 if null
func (lc *LockCount) GetCount() int64 {
	if lc.LockCount.Valid {
		return lc.LockCount.Int64
	}
	return 0
}

// LockSessionCount represents session count holding locks by type
type LockSessionCount struct {
	LockType     sql.NullString
	SessionCount sql.NullInt64
}

// GetLockType returns the lock type as a string, empty if null
func (lsc *LockSessionCount) GetLockType() string {
	if lsc.LockType.Valid {
		return lsc.LockType.String
	}
	return "unknown"
}

// GetSessionCount returns the session count, 0 if null
func (lsc *LockSessionCount) GetSessionCount() int64 {
	if lsc.SessionCount.Valid {
		return lsc.SessionCount.Int64
	}
	return 0
}

// LockedObjectCount represents count of locked objects by type
type LockedObjectCount struct {
	LockType    sql.NullString
	ObjectType  sql.NullString
	ObjectCount sql.NullInt64
}

// GetLockType returns the lock type as a string, empty if null
func (loc *LockedObjectCount) GetLockType() string {
	if loc.LockType.Valid {
		return loc.LockType.String
	}
	return "unknown"
}

// GetObjectType returns the object type as a string, empty if null
func (loc *LockedObjectCount) GetObjectType() string {
	if loc.ObjectType.Valid {
		return loc.ObjectType.String
	}
	return "unknown"
}

// GetObjectCount returns the object count, 0 if null
func (loc *LockedObjectCount) GetObjectCount() int64 {
	if loc.ObjectCount.Valid {
		return loc.ObjectCount.Int64
	}
	return 0
}

// DetailedLockInfo represents detailed lock information for monitoring
type DetailedLockInfo struct {
	SID         sql.NullInt64
	Username    sql.NullString
	LockType    sql.NullString
	LockMode    sql.NullString
	LockRequest sql.NullString
	Owner       sql.NullString
	ObjectName  sql.NullString
	ObjectType  sql.NullString
	Block       sql.NullInt64
}

// IsBlocking returns true if this lock is blocking others
func (dli *DetailedLockInfo) IsBlocking() bool {
	return dli.Block.Valid && dli.Block.Int64 > 0
}

// GetUsername returns the username as a string, empty if null
func (dli *DetailedLockInfo) GetUsername() string {
	if dli.Username.Valid {
		return dli.Username.String
	}
	return "unknown"
}

// GetLockType returns the lock type as a string, empty if null
func (dli *DetailedLockInfo) GetLockType() string {
	if dli.LockType.Valid {
		return dli.LockType.String
	}
	return "unknown"
}

// GetLockMode returns the lock mode as a string, empty if null
func (dli *DetailedLockInfo) GetLockMode() string {
	if dli.LockMode.Valid {
		return dli.LockMode.String
	}
	return "unknown"
}

// GetObjectType returns the object type as a string, empty if null
func (dli *DetailedLockInfo) GetObjectType() string {
	if dli.ObjectType.Valid {
		return dli.ObjectType.String
	}
	return "unknown"
}
