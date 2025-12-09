// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"database/sql"
)

// RACDetection represents RAC mode detection result
type RACDetection struct {
	ClusterDB sql.NullString
}

// ASMDetection represents ASM availability detection result
type ASMDetection struct {
	ASMCount int
}

// ASMDiskGroup represents ASM disk group metrics
type ASMDiskGroup struct {
	Name         sql.NullString
	TotalMB      sql.NullFloat64
	FreeMB       sql.NullFloat64
	OfflineDisks sql.NullFloat64
}

// ClusterWaitEvent represents cluster wait event metrics
type ClusterWaitEvent struct {
	InstID          sql.NullString
	Event           sql.NullString
	TotalWaits      sql.NullFloat64
	TimeWaitedMicro sql.NullFloat64
}

// RACInstanceStatus represents RAC instance status information
type RACInstanceStatus struct {
	InstID         sql.NullString
	InstanceName   sql.NullString
	HostName       sql.NullString
	Status         sql.NullString
	StartupTime    sql.NullTime
	DatabaseStatus sql.NullString
	ActiveState    sql.NullString
	Logins         sql.NullString
	Archiver       sql.NullString
	Version        sql.NullString
}

// RACActiveService represents RAC active service information
// Includes failover and high availability configuration columns
type RACActiveService struct {
	InstID                  sql.NullString
	ServiceName             sql.NullString
	NetworkName             sql.NullString
	Goal                    sql.NullString
	ClbGoal                 sql.NullString
	Blocked                 sql.NullString
	AqHaNotification        sql.NullString
	CommitOutcome           sql.NullString
	DrainTimeout            sql.NullFloat64
	ReplayInitiationTimeout sql.NullFloat64
}
