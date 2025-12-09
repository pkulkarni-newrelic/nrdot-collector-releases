// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

// DiskIOMetrics represents disk I/O statistics from gv$filestat
type DiskIOMetrics struct {
	InstID              interface{} // Instance ID (can be int or string)
	PhysicalReads       int64       // Physical reads
	PhysicalWrites      int64       // Physical writes
	PhysicalBlockReads  int64       // Physical block reads
	PhysicalBlockWrites int64       // Physical block writes
	ReadTime            int64       // Read time in milliseconds
	WriteTime           int64       // Write time in milliseconds
}
