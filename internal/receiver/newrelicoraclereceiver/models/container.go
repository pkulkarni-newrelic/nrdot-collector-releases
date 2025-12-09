// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "database/sql"

// ContainerStatus represents container status information
type ContainerStatus struct {
	ConID         sql.NullInt64
	ContainerName sql.NullString
	OpenMode      sql.NullString
	Restricted    sql.NullString
	OpenTime      sql.NullTime
}

// PDBStatus represents pluggable database status
type PDBStatus struct {
	ConID       sql.NullInt64
	PDBName     sql.NullString
	CreationSCN sql.NullInt64
	OpenMode    sql.NullString
	Restricted  sql.NullString
	OpenTime    sql.NullTime
	TotalSize   sql.NullInt64
}

// CDBTablespaceUsage represents tablespace usage across containers
type CDBTablespaceUsage struct {
	ConID          sql.NullInt64
	TablespaceName sql.NullString
	UsedBytes      sql.NullInt64
	TotalBytes     sql.NullInt64
	UsedPercent    sql.NullFloat64
}

// CDBDataFile represents data file information across containers
type CDBDataFile struct {
	ConID          sql.NullInt64
	FileName       sql.NullString
	TablespaceName sql.NullString
	Bytes          sql.NullInt64
	Status         sql.NullString
	Autoextensible sql.NullString
	MaxBytes       sql.NullInt64
	UserBytes      sql.NullInt64
}

// CDBService represents service information across containers
type CDBService struct {
	ConID        sql.NullInt64
	ServiceName  sql.NullString
	NetworkName  sql.NullString
	CreationDate sql.NullTime
	PDB          sql.NullString
	Enabled      sql.NullString
}

// ContainerContext represents current container context
type ContainerContext struct {
	ContainerName sql.NullString
	ContainerID   sql.NullString
}
