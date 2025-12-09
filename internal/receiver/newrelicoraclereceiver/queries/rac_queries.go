// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// Oracle Real Application Clusters (RAC) related queries

// RAC Detection and Status Queries
const (
	// RACDetectionSQL checks if Oracle is running in RAC mode
	RACDetectionSQL = `
		SELECT 
			VALUE 
		FROM V$PARAMETER 
		WHERE NAME = 'cluster_database'`

	// RACInstanceStatusSQL returns status of all RAC instances
	RACInstanceStatusSQL = `
		SELECT
			INST_ID,
			INSTANCE_NAME,
			HOST_NAME,
			STATUS,
			STARTUP_TIME,
			DATABASE_STATUS,
			ACTIVE_STATE,
			LOGINS,
			ARCHIVER,
			VERSION
		FROM GV$INSTANCE`

	// RACActiveServicesSQL returns active services across all RAC instances
	// Includes failover-related columns for monitoring service availability
	RACActiveServicesSQL = `
		SELECT
			INST_ID,
			NAME AS SERVICE_NAME,
			NETWORK_NAME,
			GOAL,
			CLB_GOAL,
			BLOCKED,                      -- Check for service being blocked (direct failover signal)
			AQ_HA_NOTIFICATION,           -- Check if FAN is enabled
			COMMIT_OUTCOME,               -- Check if Transaction Guard is enabled
			DRAIN_TIMEOUT,                -- Time reserved for session draining before stop
			REPLAY_INITIATION_TIMEOUT     -- Application Continuity Replay timeout
		FROM GV$ACTIVE_SERVICES
		ORDER BY NAME, INST_ID`
)

// ASM (Automatic Storage Management) Queries
const (
	// ASMDetectionSQL checks if ASM instance is available by querying the view directly
	// Returns row count if ASM is configured, otherwise throws error (caught by client)
	ASMDetectionSQL = `
		SELECT COUNT(*) AS ASM_COUNT
		FROM GV$ASM_DISKGROUP`

	// ASMDiskGroupSQL returns ASM disk group information
	ASMDiskGroupSQL = `
		SELECT
			NAME,
			TOTAL_MB,
			FREE_MB,
			OFFLINE_DISKS
		FROM GV$ASM_DISKGROUP`
)

// Cluster Performance and Wait Events
const (
	// ClusterWaitEventsSQL returns cluster-specific wait events including Global Cache waits
	// Filters for Cluster wait class events and global cache (gc) related events
	ClusterWaitEventsSQL = `
		SELECT
			INST_ID,
			EVENT,
			TOTAL_WAITS,
			TIME_WAITED_MICRO
		FROM GV$SYSTEM_EVENT
		WHERE WAIT_CLASS = 'Cluster'
		   OR EVENT LIKE 'gc%request'
		   OR EVENT LIKE 'gc%busy'
		   OR EVENT LIKE 'gc%retry'
		   OR EVENT LIKE 'gc%lost'`
)
