// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "database/sql"

// SessionStatus represents session count by status
type SessionStatus struct {
	Status sql.NullString
	Count  sql.NullInt64
}

// SessionType represents session count by type
type SessionType struct {
	Type  sql.NullString
	Count sql.NullInt64
}

// LogonStat represents logon statistics
type LogonStat struct {
	Name  sql.NullString
	Value sql.NullFloat64
}

// SessionResource represents resource consumption for a session
type SessionResource struct {
	SID             sql.NullInt64
	Username        sql.NullString
	Status          sql.NullString
	Program         sql.NullString
	Machine         sql.NullString
	OSUser          sql.NullString
	LogonTime       sql.NullTime
	LastCallET      sql.NullInt64
	CPUUsageSeconds sql.NullFloat64
	PGAMemoryBytes  sql.NullInt64
	LogicalReads    sql.NullInt64
}

// CurrentWaitEvent represents a current wait event
type CurrentWaitEvent struct {
	SID           sql.NullInt64
	Username      sql.NullString
	Event         sql.NullString
	WaitTime      sql.NullInt64
	State         sql.NullString
	SecondsInWait sql.NullInt64
	WaitClass     sql.NullString
}

// BlockingSession represents a blocking session
type BlockingSession struct {
	SID             sql.NullInt64
	Serial          sql.NullInt64
	BlockingSession sql.NullInt64
	Event           sql.NullString
	Username        sql.NullString
	Program         sql.NullString
	SecondsInWait   sql.NullInt64
}

// WaitEventSummary represents wait event summary
type WaitEventSummary struct {
	Event            sql.NullString
	TotalWaits       sql.NullInt64
	TimeWaitedMicro  sql.NullInt64
	AverageWaitMicro sql.NullFloat64
	WaitClass        sql.NullString
}

// ConnectionPoolMetric represents connection pool metrics
type ConnectionPoolMetric struct {
	MetricName sql.NullString
	Value      sql.NullInt64
}

// SessionLimit represents session and resource limits
type SessionLimit struct {
	ResourceName       sql.NullString
	CurrentUtilization sql.NullInt64
	MaxUtilization     sql.NullInt64
	InitialAllocation  sql.NullString
	LimitValue         sql.NullString
}

// ConnectionQualityMetric represents connection quality metrics
type ConnectionQualityMetric struct {
	Name  sql.NullString
	Value sql.NullFloat64
}
