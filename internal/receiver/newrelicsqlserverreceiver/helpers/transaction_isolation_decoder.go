// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

// DecodeTransactionIsolationLevel converts SQL Server numeric isolation level to human-readable string
// Based on sys.dm_exec_requests.transaction_isolation_level values
func DecodeTransactionIsolationLevel(level int64) string {
	isolationLevels := map[int64]string{
		0: "Unspecified",
		1: "READ UNCOMMITTED",
		2: "READ COMMITTED",
		3: "REPEATABLE READ",
		4: "SERIALIZABLE",
		5: "SNAPSHOT",
	}

	if description, exists := isolationLevels[level]; exists {
		return description
	}

	return "Unknown"
}

// GetIsolationLevelDescription returns a detailed description of the isolation level behavior
func GetIsolationLevelDescription(level int64) string {
	descriptions := map[int64]string{
		0: "Unspecified - No isolation level set",
		1: "READ UNCOMMITTED - Can read uncommitted data (dirty reads), lowest isolation level",
		2: "READ COMMITTED - Default isolation level, prevents dirty reads but allows non-repeatable reads",
		3: "REPEATABLE READ - Prevents dirty and non-repeatable reads but allows phantom reads",
		4: "SERIALIZABLE - Highest isolation level, prevents dirty reads, non-repeatable reads, and phantom reads",
		5: "SNAPSHOT - Uses row versioning for transactionally consistent reads without blocking",
	}

	if description, exists := descriptions[level]; exists {
		return description
	}

	return "Unknown isolation level"
}

// GetIsolationLevelLockingBehavior returns the locking behavior for the isolation level
func GetIsolationLevelLockingBehavior(level int64) string {
	lockingBehavior := map[int64]string{
		0: "N/A",
		1: "No shared locks, not blocked by exclusive locks (allows dirty reads)",
		2: "Uses shared locks, released after read completes (or uses row versioning if READ_COMMITTED_SNAPSHOT is ON)",
		3: "Holds shared locks until transaction completes",
		4: "Holds range locks until transaction completes",
		5: "Uses row versioning, no locks acquired for reads",
	}

	if behavior, exists := lockingBehavior[level]; exists {
		return behavior
	}

	return "Unknown locking behavior"
}

// IsBlockingIsolationLevel returns true if this isolation level can cause blocking
func IsBlockingIsolationLevel(level int64) bool {
	switch level {
	case 3, 4: // REPEATABLE READ, SERIALIZABLE
		return true
	case 2: // READ COMMITTED - depends on READ_COMMITTED_SNAPSHOT setting
		return true
	default:
		return false
	}
}

// GetIsolationLevelRisk returns a risk assessment for the isolation level
func GetIsolationLevelRisk(level int64) string {
	risks := map[int64]string{
		0: "Unknown",
		1: "High - Dirty reads possible, data consistency issues likely",
		2: "Low - Balanced between consistency and concurrency (default)",
		3: "Medium - Higher locking overhead, potential for blocking",
		4: "High - Highest locking overhead, significant blocking potential",
		5: "Low - Good concurrency but requires tempdb space for row versions",
	}

	if risk, exists := risks[level]; exists {
		return risk
	}

	return "Unknown risk level"
}
