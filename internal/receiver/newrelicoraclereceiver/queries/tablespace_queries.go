// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

import (
	"fmt"
	"strings"
)

// Oracle tablespace and storage related queries

// Container Database Tablespace Queries
const (
	// CDBDataFilesSQL returns datafile information across all containers
	CDBDataFilesSQL = `
		SELECT 
			con_id,
			file_name,
			tablespace_name,
			bytes,
			status,
			autoextensible,
			maxbytes,
			user_bytes
		FROM CDB_DATA_FILES
		WHERE ROWNUM <= 10000`
)

// Dynamic query builders with tablespace filtering

// BuildTablespaceWhereClause builds a WHERE clause for tablespace filtering
// Equivalent to your agent's inWhitelist("a.TABLESPACE_NAME", false, true) functionality
func BuildTablespaceWhereClause(tablespaceColumn string, includeTablespaces, excludeTablespaces []string) string {
	var conditions []string

	// Include tablespaces (whitelist)
	if len(includeTablespaces) > 0 {
		var inList []string
		for _, ts := range includeTablespaces {
			inList = append(inList, fmt.Sprintf("'%s'", strings.ReplaceAll(ts, "'", "''"))) // SQL injection protection
		}
		conditions = append(conditions, fmt.Sprintf("%s IN (%s)", tablespaceColumn, strings.Join(inList, ",")))
	}

	// Exclude tablespaces (blacklist)
	if len(excludeTablespaces) > 0 {
		var notInList []string
		for _, ts := range excludeTablespaces {
			notInList = append(notInList, fmt.Sprintf("'%s'", strings.ReplaceAll(ts, "'", "''"))) // SQL injection protection
		}
		conditions = append(conditions, fmt.Sprintf("%s NOT IN (%s)", tablespaceColumn, strings.Join(notInList, ",")))
	}

	return strings.Join(conditions, " AND ")
}

// BuildCDBTablespaceUsageSQL builds the CDB tablespace usage query with optional filtering
func BuildCDBTablespaceUsageSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT 
			con_id,
			tablespace_name,
			used_space AS used_bytes,
			tablespace_size AS total_bytes,
			used_percent
		FROM CDB_TABLESPACE_USAGE_METRICS
		WHERE ROWNUM <= 5000`

	whereClause := BuildTablespaceWhereClause("tablespace_name", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
	}

	return baseQuery
}

// BuildPDBDatafilesOfflineSQL builds the PDB datafiles offline query with tablespace filtering
// Based on your agent's oraclePDBDatafilesOffline metric
func BuildPDBDatafilesOfflineSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT
			SUM(CASE WHEN ONLINE_STATUS IN ('ONLINE','SYSTEM','RECOVER') THEN 0 ELSE 1 END) AS PDB_DATAFILES_OFFLINE,
			a.TABLESPACE_NAME
		FROM cdb_data_files a, cdb_pdbs b
		WHERE a.con_id = b.con_id`

	whereClause := BuildTablespaceWhereClause("a.TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
	}

	baseQuery += `
		GROUP BY a.TABLESPACE_NAME`

	return baseQuery
}

// BuildTablespaceUsageSQL builds the tablespace usage query with optional filtering
func BuildTablespaceUsageSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT 
			a.TABLESPACE_NAME,
			a.USED_PERCENT,
			a.USED_SPACE * b.BLOCK_SIZE AS "USED",
			a.TABLESPACE_SIZE * b.BLOCK_SIZE AS "SIZE",
			b.TABLESPACE_OFFLINE AS "OFFLINE"
		FROM DBA_TABLESPACE_USAGE_METRICS a
		JOIN (
			SELECT
				TABLESPACE_NAME,
				BLOCK_SIZE,
				MAX(CASE WHEN status = 'OFFLINE' THEN 1 ELSE 0 END) AS "TABLESPACE_OFFLINE"
			FROM DBA_TABLESPACES
			GROUP BY TABLESPACE_NAME, BLOCK_SIZE
		) b
		ON a.TABLESPACE_NAME = b.TABLESPACE_NAME`

	whereClause := BuildTablespaceWhereClause("a.TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " WHERE " + whereClause
	}

	return baseQuery
}

// BuildGlobalNameTablespaceSQL builds the global name tablespace query with optional filtering
func BuildGlobalNameTablespaceSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT
			t1.TABLESPACE_NAME,
			t2.GLOBAL_NAME
		FROM (SELECT TABLESPACE_NAME FROM DBA_TABLESPACES) t1,
			 (SELECT GLOBAL_NAME FROM global_name) t2`

	whereClause := BuildTablespaceWhereClause("t1.TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " WHERE " + whereClause
	}

	return baseQuery
}

// BuildDBIDTablespaceSQL builds the DBID tablespace query with optional filtering
func BuildDBIDTablespaceSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT
			t1.TABLESPACE_NAME,
			t2.DBID
		FROM (SELECT TABLESPACE_NAME FROM DBA_TABLESPACES) t1,
			 (SELECT DBID FROM v$database) t2`

	whereClause := BuildTablespaceWhereClause("t1.TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " WHERE " + whereClause
	}

	return baseQuery
}

// BuildCDBDatafilesOfflineSQL builds the CDB datafiles offline query with tablespace filtering
func BuildCDBDatafilesOfflineSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT
			SUM(CASE WHEN ONLINE_STATUS IN ('ONLINE', 'SYSTEM','RECOVER') THEN 0 ELSE 1 END) AS CDB_DATAFILES_OFFLINE,
			TABLESPACE_NAME
		FROM dba_data_files
		WHERE 1=1`

	whereClause := BuildTablespaceWhereClause("TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
	}

	baseQuery += `
		GROUP BY TABLESPACE_NAME`

	return baseQuery
}

// BuildPDBDatafilesOfflineCurrentContainerSQL builds PDB datafiles offline query for current container with filtering
func BuildPDBDatafilesOfflineCurrentContainerSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT
			SUM(CASE WHEN ONLINE_STATUS IN ('ONLINE','SYSTEM','RECOVER') THEN 0 ELSE 1 END) AS PDB_DATAFILES_OFFLINE,
			TABLESPACE_NAME
		FROM dba_data_files
		WHERE 1=1`

	whereClause := BuildTablespaceWhereClause("TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
	}

	baseQuery += `
		GROUP BY TABLESPACE_NAME`

	return baseQuery
}

// BuildPDBNonWriteSQL builds PDB non-write mode query with tablespace filtering
func BuildPDBNonWriteSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT 
			TABLESPACE_NAME, 
			SUM(CASE WHEN ONLINE_STATUS IN ('ONLINE','SYSTEM','RECOVER') THEN 0 ELSE 1 END) AS PDB_NON_WRITE_MODE
		FROM cdb_data_files a, cdb_pdbs b
		WHERE a.con_id = b.con_id`

	whereClause := BuildTablespaceWhereClause("a.TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
	}

	baseQuery += `
		GROUP BY TABLESPACE_NAME`

	return baseQuery
}

// BuildPDBNonWriteCurrentContainerSQL builds PDB non-write mode query for current container with filtering
func BuildPDBNonWriteCurrentContainerSQL(includeTablespaces, excludeTablespaces []string) string {
	baseQuery := `
		SELECT 
			TABLESPACE_NAME, 
			SUM(CASE WHEN ONLINE_STATUS IN ('ONLINE','SYSTEM','RECOVER') THEN 0 ELSE 1 END) AS PDB_NON_WRITE_MODE
		FROM dba_data_files
		WHERE 1=1`

	whereClause := BuildTablespaceWhereClause("TABLESPACE_NAME", includeTablespaces, excludeTablespaces)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
	}

	baseQuery += `
		GROUP BY TABLESPACE_NAME`

	return baseQuery
}
