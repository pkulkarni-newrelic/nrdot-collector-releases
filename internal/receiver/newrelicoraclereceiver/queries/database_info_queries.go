// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// Oracle database information, detection, and environment queries

// Database Feature Detection
const (
	// CheckCDBFeatureSQL checks if database is a Container Database
	CheckCDBFeatureSQL = `
		SELECT 
			CASE WHEN CDB = 'YES' THEN 1 ELSE 0 END AS IS_CDB
		FROM V$DATABASE`

	// CheckPDBCapabilitySQL checks if PDB functionality is available
	CheckPDBCapabilitySQL = `
		SELECT COUNT(*) AS PDB_COUNT
		FROM ALL_VIEWS
		WHERE VIEW_NAME = 'CDB_PDBS'
			AND OWNER = 'SYS'`

	// CheckCurrentContainerSQL returns current container context information
	CheckCurrentContainerSQL = `
		SELECT 
			SYS_CONTEXT('USERENV', 'CON_NAME') AS CONTAINER_NAME,
			SYS_CONTEXT('USERENV', 'CON_ID') AS CONTAINER_ID
		FROM DUAL`
)

// Database Version and Platform Information
const (
	// OptimizedDatabaseInfoSQL returns comprehensive database information
	OptimizedDatabaseInfoSQL = `
		SELECT 
			i.INST_ID,
			i.VERSION AS VERSION_FULL,
			i.HOST_NAME,
			d.NAME AS DATABASE_NAME,
			d.PLATFORM_NAME
		FROM GV$INSTANCE i, V$DATABASE d
		WHERE i.INST_ID = 1
			AND ROWNUM = 1`

	// DatabaseRoleSQL returns the database role (PRIMARY, PHYSICAL STANDBY, LOGICAL STANDBY, SNAPSHOT STANDBY)
	DatabaseRoleSQL = `
		SELECT 
			DATABASE_ROLE,
			OPEN_MODE,
			PROTECTION_MODE,
			PROTECTION_LEVEL
		FROM V$DATABASE`
)
