// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// Oracle system metrics and performance monitoring queries

// System Performance Metrics
const (
	// SystemSysMetricsSQL returns system metrics from gv$sysmetric
	SystemSysMetricsSQL = "SELECT INST_ID, METRIC_NAME, VALUE FROM gv$sysmetric"

	// PDBSysMetricsSQL returns PDB-specific system metrics
	PDBSysMetricsSQL = `
		SELECT
			csm.INST_ID,
			pd.NAME AS PDB_NAME,
			csm.METRIC_NAME,
			csm.VALUE
		FROM gv$con_sysmetric csm
		JOIN gv$pdbs pd ON csm.CON_ID = pd.CON_ID AND csm.INST_ID = pd.INST_ID`

	// ReadWriteMetricsSQL returns database I/O statistics
	ReadWriteMetricsSQL = `
		SELECT
			INST_ID,
			SUM(PHYRDS) AS PhysicalReads,
			SUM(PHYWRTS) AS PhysicalWrites,
			SUM(PHYBLKRD) AS PhysicalBlockReads,
			SUM(PHYBLKWRT) AS PhysicalBlockWrites,
			SUM(READTIM) * 10 AS ReadTime,
			SUM(WRITETIM) * 10 AS WriteTime
		FROM gv$filestat
		GROUP BY INST_ID`

	// LongRunningQueriesSQL returns count of long-running queries per instance
	// Returns count of ACTIVE non-BACKGROUND sessions that have been running for more than 60 seconds
	LongRunningQueriesSQL = `
		SELECT i.inst_id, COUNT(s.sid) AS total
		FROM gv$instance i
		LEFT JOIN gv$session s ON i.inst_id = s.inst_id
			AND s.status = 'ACTIVE'
			AND s.type <> 'BACKGROUND'
			AND s.last_call_et > 60
		GROUP BY i.inst_id`
)

// System Statistics Queries
const (
	// SysstatSQL returns specific system statistics
	SysstatSQL = `
		SELECT 
			inst.inst_id, 
			sysstat.name, 
			sysstat.value
		FROM GV$SYSSTAT sysstat, GV$INSTANCE inst
		WHERE sysstat.inst_id = inst.inst_id 
			AND sysstat.name IN (
				'redo buffer allocation retries', 
				'redo entries', 
				'sorts (memory)', 
				'sorts (disk)'
			)`

	// GlobalNameInstanceSQL returns instance info with global database name
	GlobalNameInstanceSQL = `
		SELECT
			t1.INST_ID,
			t2.GLOBAL_NAME
		FROM (SELECT INST_ID FROM gv$instance) t1,
			 (SELECT GLOBAL_NAME FROM global_name) t2`

	// DBIDInstanceSQL returns instance info with database ID
	DBIDInstanceSQL = `
		SELECT
			t1.INST_ID,
			t2.DBID
		FROM (SELECT INST_ID FROM gv$instance) t1,
			 (SELECT DBID FROM v$database) t2`
)

// Wait Events and Lock Queries
const (
	// RedoLogWaitsSQL returns redo log related wait events
	RedoLogWaitsSQL = `
		SELECT
			sysevent.total_waits,
			inst.inst_id,
			sysevent.event
		FROM GV$SYSTEM_EVENT sysevent, GV$INSTANCE inst
		WHERE sysevent.inst_id = inst.inst_id
			AND (sysevent.event LIKE '%log file parallel write%'
				OR sysevent.event LIKE '%log file switch completion%'
				OR sysevent.event LIKE '%log file switch (check%'
				OR sysevent.event LIKE '%log file switch (arch%'
				OR sysevent.event LIKE '%buffer busy waits%'
				OR sysevent.event LIKE '%freeBufferWaits%'
				OR sysevent.event LIKE '%free buffer inspected%')`

	// RollbackSegmentsSQL returns rollback segment statistics
	RollbackSegmentsSQL = `
		SELECT
			SUM(stat.gets) AS gets,
			SUM(stat.waits) AS waits,
			SUM(stat.waits)/SUM(stat.gets) AS ratio,
			inst.inst_id
		FROM GV$ROLLSTAT stat, GV$INSTANCE inst
		WHERE stat.inst_id = inst.inst_id
		GROUP BY inst.inst_id`
)

// User Account Metrics
const (
	// LockedAccountsSQL returns count of locked user accounts (CDB context)
	LockedAccountsSQL = `
		SELECT
			INST_ID, 
			LOCKED_ACCOUNTS
		FROM (
			SELECT COUNT(1) AS LOCKED_ACCOUNTS
			FROM cdb_users a, cdb_pdbs b
			WHERE a.con_id = b.con_id
				AND a.account_status != 'OPEN'
		) l,
		gv$instance i`

	// LockedAccountsCurrentContainerSQL returns locked accounts for current container
	LockedAccountsCurrentContainerSQL = `
		SELECT
			INST_ID, 
			LOCKED_ACCOUNTS
		FROM (
			SELECT COUNT(1) AS LOCKED_ACCOUNTS
			FROM dba_users
			WHERE account_status != 'OPEN'
		) l,
		gv$instance i`
)
