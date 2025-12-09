// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// LockResourceQuery returns lock analysis focused on resource types (what's being locked)
const LockResourceQuery = `
SELECT 
    DB_NAME(tl.resource_database_id) AS database_name,
    COUNT(*) AS total_active_locks,
    -- Resource type locks (what's being locked)
    SUM(CASE WHEN tl.resource_type = 'OBJECT' THEN 1 ELSE 0 END) AS table_locks,
    SUM(CASE WHEN tl.resource_type = 'PAGE' THEN 1 ELSE 0 END) AS page_locks,
    SUM(CASE WHEN tl.resource_type = 'RID' THEN 1 ELSE 0 END) AS row_locks,
    SUM(CASE WHEN tl.resource_type = 'KEY' THEN 1 ELSE 0 END) AS key_locks,
    SUM(CASE WHEN tl.resource_type = 'EXTENT' THEN 1 ELSE 0 END) AS extent_locks,
    SUM(CASE WHEN tl.resource_type = 'DATABASE' THEN 1 ELSE 0 END) AS database_locks,
    SUM(CASE WHEN tl.resource_type = 'FILE' THEN 1 ELSE 0 END) AS file_locks,
    SUM(CASE WHEN tl.resource_type = 'APPLICATION' THEN 1 ELSE 0 END) AS application_locks,
    SUM(CASE WHEN tl.resource_type = 'METADATA' THEN 1 ELSE 0 END) AS metadata_locks,
    SUM(CASE WHEN tl.resource_type = 'HOBT' THEN 1 ELSE 0 END) AS hobt_locks,
    SUM(CASE WHEN tl.resource_type = 'ALLOCATION_UNIT' THEN 1 ELSE 0 END) AS allocation_unit_locks
FROM sys.dm_tran_locks tl WITH (NOLOCK)
WHERE tl.resource_database_id > 0  -- Exclude system locks
GROUP BY tl.resource_database_id
HAVING COUNT(*) > 0
ORDER BY total_active_locks DESC`

// LockModeQuery returns lock analysis focused on lock modes (how resources are being locked)
const LockModeQuery = `
SELECT 
    DB_NAME(tl.resource_database_id) AS database_name,
    COUNT(*) AS total_active_locks,
    -- Lock mode locks (how it's locked)
    SUM(CASE WHEN tl.request_mode IN ('S', 'IS') THEN 1 ELSE 0 END) AS shared_locks,
    SUM(CASE WHEN tl.request_mode IN ('X', 'IX') THEN 1 ELSE 0 END) AS exclusive_locks,
    SUM(CASE WHEN tl.request_mode IN ('U', 'IU', 'SIU', 'UIX') THEN 1 ELSE 0 END) AS update_locks,
    SUM(CASE WHEN tl.request_mode IN ('IS', 'IX', 'IU', 'SIU', 'SIX', 'UIX') THEN 1 ELSE 0 END) AS intent_locks,
    SUM(CASE WHEN tl.request_mode IN ('Sch-S', 'Sch-M') THEN 1 ELSE 0 END) AS schema_locks,
    SUM(CASE WHEN tl.request_mode = 'BU' THEN 1 ELSE 0 END) AS bulk_update_locks,
    SUM(CASE WHEN tl.request_mode = 'SIX' THEN 1 ELSE 0 END) AS shared_intent_exclusive_locks
FROM sys.dm_tran_locks tl WITH (NOLOCK)
WHERE tl.resource_database_id > 0  -- Exclude system locks
GROUP BY tl.resource_database_id
HAVING COUNT(*) > 0
ORDER BY total_active_locks DESC`
