// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

import (
	"fmt"
	"strings"
)

// GetSlowQueriesSQL returns SQL for slow queries with configurable time window
// Returns total_elapsed_time_ms for delta calculation in addition to avg_elapsed_time_ms
//
// IMPORTANT: TOP N filtering and threshold filtering are applied in Go code AFTER delta calculation
// This ensures we get enough candidates for interval-based (delta) averaging
//
// Parameters:
// - intervalSeconds: Time window to fetch queries (e.g., 60 = last 60 seconds)
//
// Note: This function signature previously accepted responseTimeThreshold and rowLimit parameters,
// but those are no longer used in the SQL query. Filtering and TOP N selection are now done in Go
// after delta calculation for accurate results.
func GetSlowQueriesSQL(intervalSeconds int) string {
	return fmt.Sprintf(`
		SELECT
			SYSTIMESTAMP AS COLLECTION_TIMESTAMP,
			d.name AS cdb_name,
			p.name AS database_name,
			sa.sql_id AS query_id,
			sa.parsing_schema_name AS schema_name,
			au.username AS user_name,
			sa.executions AS execution_count,
			sa.sql_text AS query_text,
			sa.cpu_time / DECODE(sa.executions, 0, 1, sa.executions) / 1000 AS avg_cpu_time_ms,
			sa.disk_reads / DECODE(sa.executions, 0, 1, sa.executions) AS avg_disk_reads,
			sa.direct_writes / DECODE(sa.executions, 0, 1, sa.executions) AS avg_disk_writes,
			sa.elapsed_time / DECODE(sa.executions, 0, 1, sa.executions) / 1000 AS avg_elapsed_time_ms,
			sa.buffer_gets / DECODE(sa.executions, 0, 1, sa.executions) AS rows_examined,
			sa.concurrency_wait_time / DECODE(sa.executions, 0, 1, sa.executions) / 1000 AS avg_lock_time_ms,
			sa.last_active_time AS last_active_time_ms,
			sa.elapsed_time / 1000 AS total_elapsed_time_ms
		FROM
			v$sqlarea sa
		INNER JOIN
			ALL_USERS au ON sa.parsing_user_id = au.user_id
		INNER JOIN
			v$pdbs p ON sa.con_id = p.con_id
		CROSS JOIN
			v$database d
		WHERE
			sa.executions > 0
			AND sa.sql_text NOT LIKE '%%ALL_USERS%%'
			AND sa.sql_text NOT LIKE '%%V$SQL_PLAN%%'
			AND sa.sql_text NOT LIKE '%%V$SQLAREA%%'
			AND sa.sql_text NOT LIKE '%%V$SESSION%%'
			AND sa.sql_text NOT LIKE '%%V$ACTIVE_SESSION_HISTORY%%'
			AND sa.sql_text NOT LIKE '%%gv$sqlarea%%'
			AND sa.sql_text NOT LIKE '%%v$lock%%'
			AND sa.sql_text NOT LIKE '%%gv$instance%%'
			AND au.username NOT IN ('SYS', 'SYSTEM', 'DBSNMP', 'SYSMAN', 'OUTLN', 'MDSYS', 'ORDSYS', 'EXFSYS', 'WMSYS', 'APPQOSSYS', 'APEX_030200', 'OWBSYS', 'GSMADMIN_INTERNAL', 'OLAPSYS', 'XDB', 'ANONYMOUS', 'CTXSYS', 'SI_INFORMTN_SCHEMA', 'ORDDATA', 'DVSYS', 'LBACSYS', 'OJVMSYS','C##JS_USER','C##OTEL_MONITOR')
			AND sa.last_active_time >= SYSDATE - INTERVAL '%d' SECOND
		ORDER BY
			sa.elapsed_time / DECODE(sa.executions, 0, 1, sa.executions) DESC`, intervalSeconds)
}

// GetWaitEventsAndBlockingSQL returns SQL for wait events with optional blocking information
// This combines both wait events and blocking queries into a single query to reduce overhead
// High cardinality mitigation:
// - FETCH FIRST limits total rows returned to configured rowLimit
// - Filters active sessions capturing both CPU activity and actual waits (status='ACTIVE', excludes Idle waits and BACKGROUND processes)
// - Optional SQL_ID filter (slowQuerySQLIDs) for database-level filtering
// - Orders by time in state (CPU or wait time) to get most impactful sessions first
//
// Key improvements:
// - Captures CPU activity (state != 'WAITING') in addition to wait events
// - Correctly labels activity as 'ON CPU' or actual wait event name
// - Uses proper timer: wait_time_micro for WAITING state, time_since_last_wait_micro for CPU
// - Excludes background processes to match ASH behavior
func GetWaitEventsAndBlockingSQL(rowLimit int, slowQuerySQLIDs []string) string {
	sqlIDFilter := ""
	if len(slowQuerySQLIDs) > 0 {
		quotedIDs := make([]string, len(slowQuerySQLIDs))
		for i, id := range slowQuerySQLIDs {
			quotedIDs[i] = fmt.Sprintf("'%s'", id)
		}
		sqlIDFilter = fmt.Sprintf("AND s.sql_id IN (%s)", strings.Join(quotedIDs, ", "))
	}
	return fmt.Sprintf(`
		SELECT
			SYSTIMESTAMP AS COLLECTION_TIMESTAMP,
			d.name AS cdb_name,
			p.name AS database_name,
			s.username,
			s.sid,
			s.serial#,
			s.status,
			s.state,
			s.sql_id,
			s.SQL_CHILD_NUMBER,
			s.wait_class,
			s.event,
			ROUND(
				CASE
					WHEN s.state = 'WAITING' THEN s.WAIT_TIME_MICRO
					ELSE s.TIME_SINCE_LAST_WAIT_MICRO
				END / 1000, 2
			) AS wait_time_ms,
			s.SQL_EXEC_START,
			s.SQL_EXEC_ID,
			s.PROGRAM,
			s.MACHINE,
			s.ROW_WAIT_OBJ#,
			o.OWNER,
			o.OBJECT_NAME,
			o.OBJECT_TYPE,
			s.ROW_WAIT_FILE#,
			s.ROW_WAIT_BLOCK#,
			s.BLOCKING_SESSION_STATUS,
			s.BLOCKING_SESSION AS immediate_blocker_sid,
			s.FINAL_BLOCKING_SESSION_STATUS,
			s.FINAL_BLOCKING_SESSION AS final_blocker_sid,
			final_blocker.username AS final_blocker_user,
			final_blocker.serial# AS final_blocker_serial,
			final_blocker.sql_id AS final_blocker_query_id,
			final_blocker_sql.sql_text AS final_blocker_query_text
		FROM
			v$session s
		LEFT JOIN
			DBA_OBJECTS o ON s.ROW_WAIT_OBJ# = o.OBJECT_ID
		LEFT JOIN
			v$session final_blocker ON s.FINAL_BLOCKING_SESSION = final_blocker.sid
		LEFT JOIN
			v$sqlarea final_blocker_sql ON final_blocker.sql_id = final_blocker_sql.sql_id
		INNER JOIN
    		v$pdbs p ON s.con_id = p.con_id
		CROSS JOIN
			v$database d
		WHERE
			s.status = 'ACTIVE'
			AND s.type != 'BACKGROUND'
			AND s.sql_id IS NOT NULL
			AND (
				s.state != 'WAITING'
				OR s.wait_class <> 'Idle'
			)
			%s
		ORDER BY
			CASE
				WHEN s.state = 'WAITING' THEN s.WAIT_TIME_MICRO
				ELSE s.TIME_SINCE_LAST_WAIT_MICRO
			END DESC
		FETCH FIRST %d ROWS ONLY`, sqlIDFilter, rowLimit)
}

// GetSpecificChildCursorQuery returns SQL to get a SPECIFIC child cursor by sql_id and child_number
func GetSpecificChildCursorQuery(sqlID string, childNumber int64) string {
	return fmt.Sprintf(`
		SELECT
			SYSTIMESTAMP AS COLLECTION_TIMESTAMP,
			d.name AS cdb_name,
			p.name AS database_name,
			s.sql_id,
			s.child_number,
			s.plan_hash_value,
			CASE WHEN s.executions > 0 THEN (s.cpu_time / s.executions) / 1000 ELSE 0 END AS avg_cpu_time_ms,
			CASE WHEN s.executions > 0 THEN (s.elapsed_time / s.executions) / 1000 ELSE 0 END AS avg_elapsed_time_ms,
			CASE WHEN s.executions > 0 THEN (s.user_io_wait_time / s.executions) / 1000 ELSE 0 END AS avg_io_wait_time_ms,
			CASE WHEN s.executions > 0 THEN s.disk_reads / s.executions ELSE 0 END AS avg_disk_reads,
			CASE WHEN s.executions > 0 THEN s.buffer_gets / s.executions ELSE 0 END AS avg_buffer_gets,
			s.executions,
			s.invalidations,
			s.first_load_time,
			s.last_load_time
		FROM
			v$sql s
		INNER JOIN
    		v$pdbs p ON s.con_id = p.con_id	
		CROSS JOIN
			v$database d
		WHERE
			s.sql_id = '%s'
			AND s.child_number = %d`, sqlID, childNumber)
}

// GetExecutionPlanForChildQuery returns SQL to get execution plan from V$SQL_PLAN for specific child number
func GetExecutionPlanForChildQuery(sqlID string, childNumber int64) string {
	return fmt.Sprintf(`
		SELECT
			SQL_ID,
			TIMESTAMP,
			TEMP_SPACE,
			ACCESS_PREDICATES,
			PROJECTION,
			TIME,
			FILTER_PREDICATES,
			CHILD_NUMBER,
			ID,
			PARENT_ID,
			DEPTH,
			OPERATION,
			OPTIONS,
			OBJECT_OWNER,
			OBJECT_NAME,
			POSITION,
			PLAN_HASH_VALUE,
			COST,
			CARDINALITY,
			BYTES,
			CPU_COST,
			IO_COST
		FROM
			V$SQL_PLAN
		WHERE
			SQL_ID = '%s'
			AND CHILD_NUMBER = %d`, sqlID, childNumber)
}
