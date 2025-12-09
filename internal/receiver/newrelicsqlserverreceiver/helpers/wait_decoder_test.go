// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"
)

func TestDecodeWaitType(t *testing.T) {
	tests := []struct {
		name     string
		waitType string
		expected string
	}{
		{
			name:     "Empty wait type",
			waitType: "",
			expected: "Not Waiting",
		},
		{
			name:     "N/A wait type",
			waitType: "N/A",
			expected: "Not Waiting",
		},
		{
			name:     "Exclusive lock",
			waitType: "LCK_M_X",
			expected: "Lock - Exclusive Lock Contention",
		},
		{
			name:     "Shared lock",
			waitType: "LCK_M_S",
			expected: "Lock - Shared Lock Contention",
		},
		{
			name:     "Page IO latch shared",
			waitType: "PAGEIOLATCH_SH",
			expected: "Disk I/O - Reading Data Page (Shared)",
		},
		{
			name:     "Async network IO",
			waitType: "ASYNC_NETWORK_IO",
			expected: "Network - Client Not Consuming Results",
		},
		{
			name:     "Parallel query sync",
			waitType: "CXPACKET",
			expected: "Parallel Query - Thread Synchronization",
		},
		{
			name:     "Memory grant wait",
			waitType: "RESOURCE_SEMAPHORE",
			expected: "Memory - Waiting for Memory Grant",
		},
		{
			name:     "CPU pressure",
			waitType: "SOS_SCHEDULER_YIELD",
			expected: "CPU - High CPU Pressure (Thread Yield)",
		},
		{
			name:     "Write log",
			waitType: "WRITELOG",
			expected: "Transaction Log - Writing to Disk",
		},
		{
			name:     "Preemptive OS call",
			waitType: "PREEMPTIVE_OS_QUERYREGISTRY",
			expected: "OS Call - Registry Query",
		},
		{
			name:     "Unknown preemptive",
			waitType: "PREEMPTIVE_OS_CUSTOMOP",
			expected: "OS Call - OS_CUSTOMOP",
		},
		{
			name:     "AlwaysOn sync commit",
			waitType: "HADR_SYNC_COMMIT",
			expected: "AlwaysOn - Sync Commit Wait",
		},
		{
			name:     "Unknown wait type",
			waitType: "UNKNOWN_WAIT_TYPE",
			expected: "UNKNOWN_WAIT_TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeWaitType(tt.waitType)
			if result != tt.expected {
				t.Errorf("DecodeWaitType(%q) = %q, want %q", tt.waitType, result, tt.expected)
			}
		})
	}
}

func TestGetWaitTypeCategory(t *testing.T) {
	tests := []struct {
		name     string
		waitType string
		expected string
	}{
		{
			name:     "Empty wait type",
			waitType: "",
			expected: "None",
		},
		{
			name:     "Lock wait",
			waitType: "LCK_M_X",
			expected: "Locking",
		},
		{
			name:     "Disk I/O wait",
			waitType: "PAGEIOLATCH_SH",
			expected: "Disk I/O",
		},
		{
			name:     "Network wait",
			waitType: "ASYNC_NETWORK_IO",
			expected: "Network",
		},
		{
			name:     "Parallelism wait",
			waitType: "CXPACKET",
			expected: "Parallelism",
		},
		{
			name:     "Memory wait",
			waitType: "RESOURCE_SEMAPHORE",
			expected: "Memory",
		},
		{
			name:     "CPU wait",
			waitType: "SOS_SCHEDULER_YIELD",
			expected: "CPU",
		},
		{
			name:     "Latch wait",
			waitType: "LATCH_EX",
			expected: "Latch",
		},
		{
			name:     "OS call wait",
			waitType: "PREEMPTIVE_OS_FILEOPS",
			expected: "OS Call",
		},
		{
			name:     "AlwaysOn wait",
			waitType: "HADR_SYNC_COMMIT",
			expected: "AlwaysOn",
		},
		{
			name:     "Unknown wait",
			waitType: "UNKNOWN_WAIT",
			expected: "Other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWaitTypeCategory(tt.waitType)
			if result != tt.expected {
				t.Errorf("GetWaitTypeCategory(%q) = %q, want %q", tt.waitType, result, tt.expected)
			}
		})
	}
}

func TestDecodeWaitResource(t *testing.T) {
	tests := []struct {
		name         string
		waitResource string
		expectedType string
		expectedDesc string
	}{
		{
			name:         "Empty wait resource",
			waitResource: "",
			expectedType: "N/A",
			expectedDesc: "Not Applicable",
		},
		{
			name:         "KEY lock",
			waitResource: "KEY: 5:72057594049986560 (419e4517fb6a)",
			expectedType: "Key Lock",
			expectedDesc: "Database ID: 5 | HOBT: 72057594049986560 | Key Hash: 419e4517fb6a",
		},
		{
			name:         "PAGE lock",
			waitResource: "PAGE: 5:1:104",
			expectedType: "Page Lock",
			expectedDesc: "Database ID: 5 | File: 1 | Page: 104",
		},
		{
			name:         "RID lock",
			waitResource: "RID: 5:1:104:0",
			expectedType: "Row Lock",
			expectedDesc: "Database ID: 5 | File: 1 | Page: 104 | Row: 0",
		},
		{
			name:         "OBJECT lock",
			waitResource: "OBJECT: 5:245575913:0",
			expectedType: "Object Lock",
			expectedDesc: "Database ID: 5 | Object ID: 245575913 | Partition: 0",
		},
		{
			name:         "DATABASE lock",
			waitResource: "DATABASE: 5",
			expectedType: "Database Lock",
			expectedDesc: "Database ID: 5",
		},
		{
			name:         "FILE lock",
			waitResource: "FILE: 5:1",
			expectedType: "File Lock",
			expectedDesc: "Database ID: 5 | File: 1",
		},
		{
			name:         "EXTENT lock",
			waitResource: "EXTENT: 5:1:104",
			expectedType: "Extent Lock",
			expectedDesc: "Database ID: 5 | File: 1 | First Page: 104",
		},
		{
			name:         "HOBT lock",
			waitResource: "HOBT: 72057594049986560",
			expectedType: "Heap/B-Tree Lock",
			expectedDesc: "HOBT ID: 72057594049986560",
		},
		{
			name:         "APPLICATION lock",
			waitResource: "APPLICATION: 1:MyAppLock:(abc123)",
			expectedType: "Application Lock",
			expectedDesc: "Principal: 1 | Lock Name: MyAppLock | Hash: abc123",
		},
		{
			name:         "Unknown format",
			waitResource: "UNKNOWN_FORMAT: test",
			expectedType: "Unknown",
			expectedDesc: "UNKNOWN_FORMAT: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resType, resDesc := DecodeWaitResource(tt.waitResource)
			if resType != tt.expectedType {
				t.Errorf("DecodeWaitResource(%q) type = %q, want %q", tt.waitResource, resType, tt.expectedType)
			}
			if resDesc != tt.expectedDesc {
				t.Errorf("DecodeWaitResource(%q) desc = %q, want %q", tt.waitResource, resDesc, tt.expectedDesc)
			}
		})
	}
}
