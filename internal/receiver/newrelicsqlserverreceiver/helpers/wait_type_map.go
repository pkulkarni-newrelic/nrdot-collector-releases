// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

// getSQLServerWaitTypeMap returns a comprehensive map of SQL Server wait types to human-readable descriptions
// This includes the most common and useful wait types for performance monitoring and troubleshooting
func getSQLServerWaitTypeMap() map[string]string {
	return map[string]string{
		// ============================================================
		// PARALLELISM WAITS
		// ============================================================
		"CXPACKET":        "Parallel Query - Thread Synchronization",
		"CXSYNC_PORT":     "Parallel Query - Port Synchronization",
		"CXSYNC_CONSUMER": "Parallel Query - Consumer Thread Synchronization",
		"CXCONSUMER":      "Parallel Query - Consumer Wait",
		"CXROWSET_SYNC":   "Parallel Query - Range Scan Synchronization",
		"EXECSYNC":        "Parallel Query - Execution Sync",
		"EXCHANGE":        "Parallel Query - Data Exchange Iterator",

		// ============================================================
		// DISK I/O WAITS
		// ============================================================
		"PAGEIOLATCH_SH":      "Disk I/O - Reading Data Page (Shared)",
		"PAGEIOLATCH_EX":      "Disk I/O - Writing Data Page (Exclusive)",
		"PAGEIOLATCH_UP":      "Disk I/O - Updating Data Page",
		"PAGEIOLATCH_KP":      "Disk I/O - Keep Page Latch",
		"PAGEIOLATCH_DT":      "Disk I/O - Destroy Page Latch",
		"PAGEIOLATCH_NL":      "Disk I/O - No Latch",
		"WRITELOG":            "Transaction Log - Writing to Disk",
		"LOGBUFFER":           "Transaction Log - Buffer Wait",
		"LOGMGR_FLUSH":        "Transaction Log - Flush Wait",
		"LOGMGR":              "Transaction Log - Manager Wait",
		"LOGMGR_RESERVE_APPEND": "Transaction Log - Reserve Append Wait",
		"IO_COMPLETION":       "Disk I/O - Completion Wait",
		"ASYNC_IO_COMPLETION": "Disk I/O - Async Completion",
		"BACKUPIO":            "Backup - I/O Wait",
		"DISKIO_SUSPEND":      "Disk I/O - External Backup Active",

		// ============================================================
		// LOCK WAITS
		// ============================================================
		"LCK_M_X":      "Lock - Exclusive Lock Contention",
		"LCK_M_S":      "Lock - Shared Lock Contention",
		"LCK_M_U":      "Lock - Update Lock Contention",
		"LCK_M_IX":     "Lock - Intent Exclusive",
		"LCK_M_IS":     "Lock - Intent Shared",
		"LCK_M_IU":     "Lock - Intent Update",
		"LCK_M_SIX":    "Lock - Shared Intent Exclusive",
		"LCK_M_SIU":    "Lock - Shared Intent Update",
		"LCK_M_UIX":    "Lock - Update Intent Exclusive",
		"LCK_M_BU":     "Lock - Bulk Update",
		"LCK_M_RS_S":   "Lock - Range Shared-Shared",
		"LCK_M_RS_U":   "Lock - Range Shared-Update",
		"LCK_M_RX_S":   "Lock - Range Exclusive-Shared",
		"LCK_M_RX_U":   "Lock - Range Exclusive-Update",
		"LCK_M_RX_X":   "Lock - Range Exclusive-Exclusive",
		"LCK_M_RIn_NL": "Lock - Range Insert Null",
		"LCK_M_RIn_S":  "Lock - Range Insert Shared",
		"LCK_M_RIn_U":  "Lock - Range Insert Update",
		"LCK_M_RIn_X":  "Lock - Range Insert Exclusive",
		"LCK_M_SCH_M":  "Lock - Schema Modify",
		"LCK_M_SCH_S":  "Lock - Schema Share",

		// ============================================================
		// MEMORY WAITS
		// ============================================================
		"RESOURCE_SEMAPHORE":                              "Memory - Waiting for Memory Grant",
		"RESOURCE_SEMAPHORE_QUERY_COMPILE":                "Memory - Query Compilation Memory",
		"CMEMTHREAD":                                      "Memory - Thread Memory Wait",
		"PWAIT_RESOURCE_SEMAPHORE_FT_PARALLEL_QUERY_SYNC": "Memory - Full-Text Parallel Query Sync",
		"MEMORY_ALLOCATION_EXT":                           "Memory - Allocation from OS",

		// ============================================================
		// NETWORK WAITS
		// ============================================================
		"ASYNC_NETWORK_IO":   "Network - Client Not Consuming Results",
		"NETWORKIO":          "Network - General Network I/O",
		"NET_WAITFOR_PACKET": "Network - Waiting for Network Packet",

		// ============================================================
		// CPU WAITS
		// ============================================================
		"SOS_SCHEDULER_YIELD": "CPU - High CPU Pressure (Thread Yield)",
		"THREADPOOL":          "CPU - Thread Pool Starvation",
		"SQLTRACE_WAIT_ENTRIES": "CPU - SQL Trace Wait",

		// ============================================================
		// LATCH WAITS (In-Memory Contention)
		// ============================================================
		"PAGELATCH_SH": "Latch - Shared Page Latch (In-Memory Buffer Contention)",
		"PAGELATCH_EX": "Latch - Exclusive Page Latch (In-Memory Buffer Contention)",
		"PAGELATCH_UP": "Latch - Update Page Latch (In-Memory Buffer Contention)",
		"PAGELATCH_KP": "Latch - Keep Page Latch",
		"PAGELATCH_DT": "Latch - Destroy Page Latch",
		"LATCH_EX":     "Latch - Exclusive Latch Wait",
		"LATCH_SH":     "Latch - Shared Latch Wait",
		"LATCH_UP":     "Latch - Update Latch Wait",
		"LATCH_KP":     "Latch - Keep Latch Wait",
		"LATCH_DT":     "Latch - Destroy Latch Wait",

		// ============================================================
		// TRANSACTION WAITS
		// ============================================================
		"DTC":               "Transaction - Distributed Transaction Coordinator Wait",
		"XACTLOCKINFO":      "Transaction - Lock Info Wait",
		"TRANSACTION_MUTEX": "Transaction - Mutex Wait",

		// ============================================================
		// BACKUP/RESTORE WAITS
		// ============================================================
		"BACKUP":                            "Backup - Backup Operation Wait",
		"BACKUPBUFFER":                      "Backup - Buffer Wait",
		"BACKUPTHREAD":                      "Backup - Thread Wait",
		"RESTORE_FILEHANDLECACHE_ENTRYLOCK": "Restore - File Handle Cache Entry Lock",

		// ============================================================
		// ALWAYS ON AVAILABILITY GROUP WAITS
		// ============================================================
		"HADR_SYNC_COMMIT":            "AlwaysOn - Sync Commit Wait",
		"HADR_SYNCHRONIZING_THROTTLE": "AlwaysOn - Synchronizing Throttle",
		"HADR_LOGCAPTURE_WAIT":        "AlwaysOn - Log Capture Wait",
		"HADR_DATABASE_FLOW_CONTROL":  "AlwaysOn - Database Flow Control",
		"HADR_THROTTLE_LOG_RATE_MISMATCHED_SLO": "AlwaysOn - Log Rate Throttle (Mismatched SLO)",
		"HADR_AG_MUTEX":               "AlwaysOn - Availability Group Mutex",
		"HADR_WORK_QUEUE":             "AlwaysOn - Work Queue Wait",

		// ============================================================
		// PREEMPTIVE OS CALL WAITS
		// ============================================================
		"PREEMPTIVE_OS_QUERYREGISTRY":     "OS Call - Registry Query",
		"PREEMPTIVE_OS_FILEOPS":           "OS Call - File Operations",
		"PREEMPTIVE_OS_LIBRARYOPS":        "OS Call - Library Operations",
		"PREEMPTIVE_OS_CRYPTOPS":          "OS Call - Cryptographic Operations",
		"PREEMPTIVE_OS_AUTHENTICATIONOPS": "OS Call - Authentication",
		"PREEMPTIVE_OS_GENERICOPS":        "OS Call - Generic Operations",
		"PREEMPTIVE_OS_WRITEFILEGATHER":   "OS Call - Write File Gather",
		"PREEMPTIVE_OS_WRITEFILE":         "OS Call - Write File",

		// ============================================================
		// SERVICE BROKER WAITS
		// ============================================================
		"BROKER_RECEIVE_WAITFOR": "Service Broker - Receive Wait",
		"BROKER_TRANSMITTER":     "Service Broker - Transmitter Wait",
		"BROKER_TASK_STOP":       "Service Broker - Task Stop",
		"DBMIRROR_EVENTS_QUEUE":  "Database Mirroring - Event Queue",

		// ============================================================
		// SLEEP/IDLE WAITS
		// ============================================================
		"SLEEP_TASK":        "Sleep - Intentional Wait (WAITFOR command)",
		"SLEEP_BPOOL_STEAL": "Sleep - Buffer Pool Steal",
		"SLEEP_SYSTEMTASK":  "Sleep - System Task Wait",
		"LAZYWRITER_SLEEP":  "Sleep - Lazy Writer Wait",

		// ============================================================
		// CLR WAITS
		// ============================================================
		"CLR_AUTO_EVENT":   "CLR - Auto Event Wait",
		"CLR_MANUAL_EVENT": "CLR - Manual Event Wait",
		"CLR_SEMAPHORE":    "CLR - Semaphore Wait",
		"SQLCLR_APPDOMAIN": "CLR - Application Domain Startup",
		"SQLCLR_ASSEMBLY":  "CLR - Assembly List Access",

		// ============================================================
		// FULL-TEXT SEARCH WAITS
		// ============================================================
		"FT_IFTS_SCHEDULER_IDLE_WAIT": "Full-Text - Scheduler Idle",
		"MSSEARCH":                    "Full-Text - Search Wait",

		// ============================================================
		// BATCH MODE / COLUMNSTORE WAITS
		// ============================================================
		"HTBUILD":        "Batch Mode - Hash Table Build",
		"HTDELETE":       "Batch Mode - Hash Table Delete",
		"HTREINIT":       "Batch Mode - Hash Table Reinit",
		"HTREPARTITION":  "Batch Mode - Hash Table Repartition",
		"BMPALLOCATION":  "Batch Mode - Bitmap Filter Allocation",
		"BMPBUILD":       "Batch Mode - Bitmap Filter Build",
		"BMPREPARTITION": "Batch Mode - Bitmap Filter Repartition",

		// ============================================================
		// IN-MEMORY OLTP (HEKATON) WAITS
		// ============================================================
		"WAIT_XTP_GUEST":                "In-Memory OLTP - Guest Wait",
		"WAIT_XTP_HOST_WAIT":            "In-Memory OLTP - Host Wait",
		"WAIT_XTP_OFFLINE_CKPT_NEW_LOG": "In-Memory OLTP - Offline Checkpoint New Log",
		"WAIT_XTP_CKPT_CLOSE":           "In-Memory OLTP - Checkpoint Close",
		"XTPPROC_CACHE_ACCESS":          "In-Memory OLTP - Procedure Cache Access",

		// ============================================================
		// QUERY STORE WAITS
		// ============================================================
		"QDS_SHUTDOWN_QUEUE":                             "Query Store - Shutdown Queue",
		"QDS_CLEANUP_STALE_QUERIES_TASK_MAIN_LOOP_SLEEP": "Query Store - Cleanup Task Sleep",
		"QDS_PERSIST_TASK_MAIN_LOOP_SLEEP":               "Query Store - Persist Task Sleep",

		// ============================================================
		// MISCELLANEOUS COMMON WAITS
		// ============================================================
		"OLEDB":                      "External Data Access - OLE DB Provider Call",
		"MSQL_XP":                    "Extended Stored Procedure - Execution Wait",
		"WAITFOR":                    "Waitfor - User-Initiated Wait",
		"ONDEMAND_TASK_QUEUE":        "Background Task - Waiting for High Priority Requests",
		"TEMPOBJ":                    "Temp Object - Drop Synchronization",
		"TRACEWRITE":                 "Trace - Buffer Wait",
		"SQLTRACE_BUFFER_FLUSH":      "Trace - Buffer Flush",
		"DEADLOCK_ENUM_MUTEX":        "Deadlock - Detection Synchronization",
		"DEADLOCK_TASK_SEARCH":       "Deadlock - Task Search",
		"DBMIRROR_SEND":              "Database Mirroring - Send Wait",
		"DBMIRRORING_CMD":            "Database Mirroring - Command Wait",
		"ASYNC_DISKPOOL_LOCK":        "Async - Disk Pool Lock",
		"FCB_REPLICA_WRITE":          "Snapshot - Sparse File Write",
		"FCB_REPLICA_READ":           "Snapshot - Sparse File Read",
		"QRY_MEM_GRANT_INFO_MUTEX":   "Query - Memory Grant Info Mutex",
		"SOS_WORK_DISPATCHER":        "SOS - Work Dispatcher",
		"POOL_LOG_RATE_GOVERNOR":     "Log Rate - Pool Governor",
		"INSTANCE_LOG_RATE_GOVERNOR": "Log Rate - Instance Governor",
		"RBIO_RG_STORAGE":            "Hyperscale - Log Consumption by Page Servers",
		"RBIO_RG_DESTAGE":            "Hyperscale - Log Consumption by Long-Term Storage",

		// ============================================================
		// INTERNAL/INFORMATIONAL WAITS (Less Critical)
		// ============================================================
		"ASSEMBLY_LOAD":                    "Internal - Assembly Loading",
		"BROKER_EVENTHANDLER":              "Internal - Service Broker Event Handler",
		"CHECKPOINT_QUEUE":                 "Internal - Checkpoint Queue",
		"CHKPT":                            "Internal - Checkpoint Thread Startup",
		"DIRTY_PAGE_POLL":                  "Internal - Dirty Page Poll",
		"KSOURCE_WAKEUP":                   "Internal - Service Control Task",
		"LOGMGR_QUEUE":                     "Internal - Log Writer Task",
		"REQUEST_FOR_DEADLOCK_SEARCH":      "Internal - Deadlock Monitor",
		"SERVER_IDLE_CHECK":                "Internal - Server Idle Check",
		"SLEEP_DBSTARTUP":                  "Internal - Database Startup",
		"SLEEP_DCOMSTARTUP":                "Internal - DCOM Initialization",
		"SQLTRACE_INCREMENTAL_FLUSH_SLEEP": "Internal - SQL Trace Incremental Flush",
		"XE_DISPATCHER_WAIT":               "Internal - Extended Events Dispatcher",
		"XE_TIMER_EVENT":                   "Internal - Extended Events Timer",
	}
}
