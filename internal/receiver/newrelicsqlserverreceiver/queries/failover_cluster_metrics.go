// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// replica performance metrics when running in high availability failover cluster environments.
//
// Failover Cluster Metrics Categories:
//
// 1. Database Replica Performance Metrics:
//   - Log bytes received per second from primary replica
//   - Transaction delay on secondary replica
//   - Flow control time for log record processing
//
// Query Sources:
// - sys.dm_os_performance_counters: Always On Database Replica performance counters
//
// Metric Collection Strategy:
// - Uses PIVOT operation to transform performance counter rows into columns
// - Filters for Database Replica object counters specific to Always On Availability Groups
// - Returns structured data for log replication performance monitoring
// - Provides insights into replication lag and flow control behavior
//
// Engine Support:
// - Default: Full failover cluster metrics for SQL Server with Always On Availability Groups
// - AzureSQLDatabase: Not applicable (no Always On AG support in single database)
// - AzureSQLManagedInstance: Limited support (managed service handles AG internally)
//
// Availability:
// - Only available on SQL Server instances configured with Always On Availability Groups
// - Returns empty result set on instances without Always On AG enabled
// - Requires appropriate permissions to query performance counter DMVs
package queries

// Source: sys.dm_os_performance_counters for Database Replica performance counters
//
// The query returns:
// - instance_name: Database instance name from performance counters
// - Log Bytes Received/sec: Rate of log records received by secondary replica from primary (bytes/sec)
// - Transaction Delay: Average delay for transactions on the secondary replica (milliseconds)
// - Flow Control Time (ms/sec): Time spent in flow control by log records (milliseconds/sec)
const FailoverClusterReplicaQuery = `SELECT
    instance_name,
    ISNULL(MAX(CASE WHEN counter_name = 'Log Bytes Received/sec' THEN cntr_value END), 0) AS [Log Bytes Received/sec],
    ISNULL(MAX(CASE WHEN counter_name = 'Transaction Delay' THEN cntr_value END), 0) AS [Transaction Delay],
    ISNULL(MAX(CASE WHEN counter_name = 'Flow Control Time (ms/sec)' THEN cntr_value END), 0) AS [Flow Control Time (ms/sec)]
FROM
    sys.dm_os_performance_counters
WHERE
    object_name LIKE '%Database Replica%'
    AND counter_name IN (
        'Log Bytes Received/sec',
        'Transaction Delay',
        'Flow Control Time (ms/sec)'
    )
GROUP BY
    instance_name;`

// detailed log send/redo queue metrics and synchronization state for each database in the availability group
//
// The query returns:
// - replica_server_name: Name of the server hosting the replica
// - database_name: Name of the database in the availability group
// - log_send_queue_kb: Amount of log records not yet sent to secondary replica (KB)
// - redo_queue_kb: Amount of log records waiting to be redone on secondary replica (KB)
// - redo_rate_kb_sec: Rate at which log records are being redone on secondary replica (KB/sec)
// - database_state_desc: Current state of the database replica
// - synchronization_state_desc: Synchronization state of the database replica
// - is_local: Whether this is a local replica (1) or remote (0)
// - is_primary_replica: Whether this is the primary replica (1) or secondary (0)
// - last_commit_time: Timestamp of the last transaction commit
// - last_sent_time: Timestamp when the last log block was sent
// - last_received_time: Timestamp when the last log block was received
// - last_hardened_time: Timestamp when the last log block was hardened
// - last_redone_lsn: Log sequence number of the last redone log record
// - suspend_reason_desc: Reason why the database is suspended (if applicable)
const FailoverClusterReplicaStateQuery = `SELECT
    ar.replica_server_name,
    d.name AS database_name,
    drs.log_send_queue_size AS log_send_queue_kb,
    drs.redo_queue_size AS redo_queue_kb,
    drs.redo_rate AS redo_rate_kb_sec,
    ISNULL(drs.database_state_desc, 'UNKNOWN') AS database_state_desc,
    ISNULL(drs.synchronization_state_desc, 'UNKNOWN') AS synchronization_state_desc,
    CAST(drs.is_local AS INT) AS is_local,
    CAST(drs.is_primary_replica AS INT) AS is_primary_replica,
    ISNULL(CONVERT(VARCHAR(23), drs.last_commit_time, 121), '') AS last_commit_time,
    ISNULL(CONVERT(VARCHAR(23), drs.last_sent_time, 121), '') AS last_sent_time,
    ISNULL(CONVERT(VARCHAR(23), drs.last_received_time, 121), '') AS last_received_time,
    ISNULL(CONVERT(VARCHAR(23), drs.last_hardened_time, 121), '') AS last_hardened_time,
    ISNULL(CONVERT(VARCHAR(25), drs.last_redone_lsn), '') AS last_redone_lsn,
    ISNULL(drs.suspend_reason_desc, 'NONE') AS suspend_reason_desc
FROM
    sys.dm_hadr_database_replica_states AS drs
JOIN
    sys.availability_replicas AS ar ON drs.replica_id = ar.replica_id
JOIN
    sys.databases AS d ON drs.database_id = d.database_id;`

// The query returns:
// - replica_server_name: Name of the server instance hosting the availability replica
// - role_desc: Current role of the replica (PRIMARY, SECONDARY)
// - synchronization_health_desc: Health of data synchronization (HEALTHY, PARTIALLY_HEALTHY, NOT_HEALTHY)
// - availability_mode_desc: Availability mode (SYNCHRONOUS_COMMIT, ASYNCHRONOUS_COMMIT)
// - failover_mode_desc: Failover mode (AUTOMATIC, MANUAL, EXTERNAL)
// - backup_priority: Backup priority setting (0-100)
// - endpoint_url: Database mirroring endpoint URL
// - read_only_routing_url: Read-only routing URL
// - connected_state_desc: Connection state (CONNECTED, DISCONNECTED)
// - operational_state_desc: Operational state (ONLINE, OFFLINE, etc.)
// - recovery_health_desc: Recovery health (ONLINE, OFFLINE, etc.)
const FailoverClusterAvailabilityGroupHealthQuery = `SELECT
    ISNULL(ar.replica_server_name, 'UNKNOWN') AS replica_server_name,
    ISNULL(ars.role_desc, 'UNKNOWN') AS role_desc,
    ISNULL(ars.synchronization_health_desc, 'UNKNOWN') AS synchronization_health_desc,
    ISNULL(ar.availability_mode_desc, 'UNKNOWN') AS availability_mode_desc,
    ISNULL(ar.failover_mode_desc, 'UNKNOWN') AS failover_mode_desc,
    ISNULL(ar.backup_priority, 0) AS backup_priority,
    ISNULL(ar.endpoint_url, '') AS endpoint_url,
    ISNULL(ar.read_only_routing_url, '') AS read_only_routing_url,
    ISNULL(ars.connected_state_desc, 'UNKNOWN') AS connected_state_desc,
    ISNULL(ars.operational_state_desc, 'UNKNOWN') AS operational_state_desc,
    ISNULL(ars.recovery_health_desc, 'UNKNOWN') AS recovery_health_desc
FROM
    sys.dm_hadr_availability_replica_states AS ars
INNER JOIN
    sys.availability_replicas AS ar ON ars.replica_id = ar.replica_id;`

// The query returns:
// - group_name: Name of the availability group
// - automated_backup_preference_desc: Backup preference setting (PRIMARY, SECONDARY_ONLY, SECONDARY, NONE)
// - failure_condition_level: Failure detection level (1-5)
// - health_check_timeout: Health check timeout value in milliseconds
// - cluster_type_desc: Cluster type (WSFC, EXTERNAL, NONE)
// - required_synchronized_secondaries_to_commit: Number of secondary replicas required to be synchronized
const FailoverClusterAvailabilityGroupQuery = `SELECT
    ISNULL(ag.name, 'UNKNOWN') AS group_name,
    ISNULL(ag.automated_backup_preference_desc, 'UNKNOWN') AS automated_backup_preference_desc,
    ISNULL(ag.failure_condition_level, 0) AS failure_condition_level,
    ISNULL(ag.health_check_timeout, 0) AS health_check_timeout,
    ISNULL(ag.cluster_type_desc, 'UNKNOWN') AS cluster_type_desc,
    ISNULL(ag.required_synchronized_secondaries_to_commit, 0) AS required_synchronized_secondaries_to_commit
FROM
    sys.availability_groups AS ag;`

// and redo performance in availability groups. Compatible with both Standard SQL Server
// and Azure SQL Managed Instance.
//
// The query returns:
// - replica_server_name: Name of the server hosting the replica
// - database_name: Name of the database in the availability group
// - log_send_queue_kb: Amount of log records not yet sent to secondary replica (KB)
// - redo_queue_kb: Amount of log records waiting to be redone on secondary replica (KB)
// - redo_rate_kb_sec: Rate at which log records are being redone on secondary replica (KB/sec)
const FailoverClusterRedoQueueQuery = `SELECT
    ar.replica_server_name,
    d.name AS database_name,
    drs.log_send_queue_size AS log_send_queue_kb,
    drs.redo_queue_size AS redo_queue_kb,
    drs.redo_rate AS redo_rate_kb_sec
FROM
    sys.dm_hadr_database_replica_states AS drs
JOIN
    sys.availability_replicas AS ar ON drs.replica_id = ar.replica_id
JOIN
    sys.databases AS d ON drs.database_id = d.database_id;`
