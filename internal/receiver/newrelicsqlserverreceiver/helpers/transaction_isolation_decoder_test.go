// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"
)

func TestDecodeTransactionIsolationLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    int64
		expected string
	}{
		{
			name:     "Unspecified",
			level:    0,
			expected: "Unspecified",
		},
		{
			name:     "READ UNCOMMITTED",
			level:    1,
			expected: "READ UNCOMMITTED",
		},
		{
			name:     "READ COMMITTED",
			level:    2,
			expected: "READ COMMITTED",
		},
		{
			name:     "REPEATABLE READ",
			level:    3,
			expected: "REPEATABLE READ",
		},
		{
			name:     "SERIALIZABLE",
			level:    4,
			expected: "SERIALIZABLE",
		},
		{
			name:     "SNAPSHOT",
			level:    5,
			expected: "SNAPSHOT",
		},
		{
			name:     "Unknown level",
			level:    99,
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeTransactionIsolationLevel(tt.level)
			if result != tt.expected {
				t.Errorf("DecodeTransactionIsolationLevel(%d) = %s; want %s", tt.level, result, tt.expected)
			}
		})
	}
}

func TestGetIsolationLevelDescription(t *testing.T) {
	tests := []struct {
		name     string
		level    int64
		contains string // Check if result contains this string
	}{
		{
			name:     "READ UNCOMMITTED description",
			level:    1,
			contains: "dirty reads",
		},
		{
			name:     "READ COMMITTED description",
			level:    2,
			contains: "Default",
		},
		{
			name:     "REPEATABLE READ description",
			level:    3,
			contains: "non-repeatable reads",
		},
		{
			name:     "SERIALIZABLE description",
			level:    4,
			contains: "Highest isolation level",
		},
		{
			name:     "SNAPSHOT description",
			level:    5,
			contains: "row versioning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIsolationLevelDescription(tt.level)
			if result == "" {
				t.Errorf("GetIsolationLevelDescription(%d) returned empty string", tt.level)
			}
		})
	}
}

func TestGetIsolationLevelLockingBehavior(t *testing.T) {
	tests := []struct {
		name     string
		level    int64
		contains string
	}{
		{
			name:     "READ UNCOMMITTED locking",
			level:    1,
			contains: "No shared locks",
		},
		{
			name:     "READ COMMITTED locking",
			level:    2,
			contains: "shared locks",
		},
		{
			name:     "REPEATABLE READ locking",
			level:    3,
			contains: "transaction completes",
		},
		{
			name:     "SERIALIZABLE locking",
			level:    4,
			contains: "range locks",
		},
		{
			name:     "SNAPSHOT locking",
			level:    5,
			contains: "row versioning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIsolationLevelLockingBehavior(tt.level)
			if result == "" {
				t.Errorf("GetIsolationLevelLockingBehavior(%d) returned empty string", tt.level)
			}
		})
	}
}

func TestIsBlockingIsolationLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    int64
		expected bool
	}{
		{
			name:     "Unspecified - not blocking",
			level:    0,
			expected: false,
		},
		{
			name:     "READ UNCOMMITTED - not blocking",
			level:    1,
			expected: false,
		},
		{
			name:     "READ COMMITTED - can block",
			level:    2,
			expected: true,
		},
		{
			name:     "REPEATABLE READ - blocking",
			level:    3,
			expected: true,
		},
		{
			name:     "SERIALIZABLE - blocking",
			level:    4,
			expected: true,
		},
		{
			name:     "SNAPSHOT - not blocking",
			level:    5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlockingIsolationLevel(tt.level)
			if result != tt.expected {
				t.Errorf("IsBlockingIsolationLevel(%d) = %v; want %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestGetIsolationLevelRisk(t *testing.T) {
	tests := []struct {
		name  string
		level int64
		want  string
	}{
		{
			name:  "READ UNCOMMITTED - High risk",
			level: 1,
			want:  "High - Dirty reads possible, data consistency issues likely",
		},
		{
			name:  "READ COMMITTED - Low risk",
			level: 2,
			want:  "Low - Balanced between consistency and concurrency (default)",
		},
		{
			name:  "REPEATABLE READ - Medium risk",
			level: 3,
			want:  "Medium - Higher locking overhead, potential for blocking",
		},
		{
			name:  "SERIALIZABLE - High risk",
			level: 4,
			want:  "High - Highest locking overhead, significant blocking potential",
		},
		{
			name:  "SNAPSHOT - Low risk",
			level: 5,
			want:  "Low - Good concurrency but requires tempdb space for row versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIsolationLevelRisk(tt.level)
			if result != tt.want {
				t.Errorf("GetIsolationLevelRisk(%d) = %s; want %s", tt.level, result, tt.want)
			}
		})
	}
}
