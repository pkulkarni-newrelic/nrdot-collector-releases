// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// Oracle SQL queries for connection statistics and session monitoring

// Session Count Queries
const (
	// SessionCountSQL returns the count of user sessions
	SessionCountSQL = "SELECT COUNT(*) AS SESSION_COUNT FROM v$session WHERE type = 'USER'"

	// TotalSessionsSQL returns the total number of sessions
	TotalSessionsSQL = "SELECT COUNT(*) as TOTAL_SESSIONS FROM V$SESSION"

	// ActiveSessionsSQL returns the count of active sessions
	ActiveSessionsSQL = "SELECT COUNT(*) as ACTIVE_SESSIONS FROM V$SESSION WHERE STATUS = 'ACTIVE'"

	// InactiveSessionsSQL returns the count of inactive sessions
	InactiveSessionsSQL = "SELECT COUNT(*) as INACTIVE_SESSIONS FROM V$SESSION WHERE STATUS = 'INACTIVE'"

	// SessionStatusSQL returns session count breakdown by status
	SessionStatusSQL = `
		SELECT 
			STATUS,
			COUNT(*) AS SESSION_COUNT
		FROM V$SESSION 
		GROUP BY STATUS`

	// SessionTypeSQL returns session count breakdown by type
	SessionTypeSQL = `
		SELECT 
			TYPE,
			COUNT(*) AS SESSION_COUNT
		FROM V$SESSION 
		GROUP BY TYPE`

	// LogonsStatsSQL returns logon statistics
	LogonsStatsSQL = `
		SELECT 
			NAME, 
			VALUE 
		FROM V$SYSSTAT 
		WHERE NAME IN ('logons cumulative', 'logons current')`

	// SessionResourceConsumptionSQL returns detailed session resource usage
	SessionResourceConsumptionSQL = `
		SELECT 
			s.SID,
			s.USERNAME,
			s.STATUS,
			s.PROGRAM,
			s.MACHINE,
			s.OSUSER,
			s.LOGON_TIME,
			s.LAST_CALL_ET,
			ROUND(ss_cpu.VALUE/100, 2) AS CPU_USAGE_SECONDS,
			ss_pga.VALUE AS PGA_MEMORY_BYTES,
			ss_logical.VALUE AS LOGICAL_READS
		FROM V$SESSION s
		LEFT JOIN V$SESSTAT ss_cpu ON s.SID = ss_cpu.SID 
			AND ss_cpu.STATISTIC# = (SELECT STATISTIC# FROM V$STATNAME WHERE NAME = 'CPU used by this session')
		LEFT JOIN V$SESSTAT ss_pga ON s.SID = ss_pga.SID 
			AND ss_pga.STATISTIC# = (SELECT STATISTIC# FROM V$STATNAME WHERE NAME = 'session pga memory')
		LEFT JOIN V$SESSTAT ss_logical ON s.SID = ss_logical.SID 
			AND ss_logical.STATISTIC# = (SELECT STATISTIC# FROM V$STATNAME WHERE NAME = 'session logical reads')
		WHERE s.TYPE = 'USER'
		ORDER BY ss_cpu.VALUE DESC NULLS LAST
		FETCH FIRST 500 ROWS ONLY`
)

// Wait Events and Blocking Queries
const (

	// CurrentWaitEventsSQL returns currently waiting events (non-idle)
	CurrentWaitEventsSQL = `
		SELECT 
			SID,
			USERNAME,
			EVENT,
			WAIT_TIME,
			STATE,
			WAIT_TIME_MICRO,
			WAIT_CLASS
		FROM V$SESSION 
		WHERE STATUS = 'ACTIVE' 
			AND WAIT_CLASS != 'Idle'
			AND EVENT IS NOT NULL
		ORDER BY WAIT_TIME_MICRO DESC
		FETCH FIRST 200 ROWS ONLY`

	// BlockingSessionsSQL returns sessions that are blocking other sessions
	BlockingSessionsSQL = `
		SELECT 
			SID,
			SERIAL#,
			BLOCKING_SESSION,
			EVENT,
			USERNAME,
			PROGRAM,
			WAIT_TIME_MICRO
		FROM V$SESSION 
		WHERE BLOCKING_SESSION IS NOT NULL
		ORDER BY WAIT_TIME_MICRO DESC
		FETCH FIRST 100 ROWS ONLY`

	// WaitEventSummarySQL returns top wait events by time waited
	WaitEventSummarySQL = `
		SELECT 
			EVENT,
			TOTAL_WAITS,
			TIME_WAITED_MICRO,
			CASE 
				WHEN TOTAL_WAITS > 0 THEN TIME_WAITED_MICRO / TOTAL_WAITS 
				ELSE 0 
			END AS AVERAGE_WAIT_MICRO,
			WAIT_CLASS
		FROM V$SYSTEM_EVENT 
		WHERE WAIT_CLASS != 'Idle'
		ORDER BY TIME_WAITED_MICRO DESC 
		FETCH FIRST 20 ROWS ONLY`
)

// Connection Pool and Resource Queries
const (

	// ConnectionPoolMetricsSQL returns connection pool related metrics
	ConnectionPoolMetricsSQL = `
		SELECT 
			'shared_servers' AS METRIC_NAME,
			COUNT(*) AS VALUE
		FROM V$SHARED_SERVER
		UNION ALL
		SELECT 
			'dispatchers' AS METRIC_NAME,
			COUNT(*) AS VALUE
		FROM V$DISPATCHER
		UNION ALL
		SELECT 
			'circuits' AS METRIC_NAME,
			COUNT(*) AS VALUE
		FROM V$CIRCUIT
		WHERE STATUS = 'NORMAL'`

	// SessionLimitsSQL returns session and resource limits
	SessionLimitsSQL = `
		SELECT 
			RESOURCE_NAME,
			CURRENT_UTILIZATION,
			MAX_UTILIZATION,
			INITIAL_ALLOCATION,
			LIMIT_VALUE
		FROM V$RESOURCE_LIMIT 
		WHERE RESOURCE_NAME IN ('sessions', 'processes', 'enqueue_locks', 'enqueue_resources')`

	// ConnectionQualitySQL returns connection quality and performance metrics
	ConnectionQualitySQL = `
		SELECT 
			NAME,
			VALUE
		FROM V$SYSSTAT 
		WHERE NAME IN (
			'user commits',
			'user rollbacks', 
			'parse count (total)',
			'parse count (hard)',
			'execute count',
			'SQL*Net roundtrips to/from client',
			'bytes sent via SQL*Net to client',
			'bytes received via SQL*Net from client'
		)`
)
