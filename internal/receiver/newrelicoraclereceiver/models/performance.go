// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "database/sql"

// SysstatMetric represents system statistics from gv$sysstat
type SysstatMetric struct {
	InstID interface{}
	Name   string
	Value  sql.NullInt64
}

// RollbackSegmentsMetric represents rollback segment statistics
type RollbackSegmentsMetric struct {
	Gets   sql.NullInt64
	Waits  sql.NullInt64
	Ratio  sql.NullFloat64
	InstID interface{}
}

// RedoLogWaitsMetric represents redo log wait events
type RedoLogWaitsMetric struct {
	TotalWaits sql.NullInt64
	InstID     interface{}
	Event      string
}
