// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// Oracle Container Database (CDB) and Pluggable Database (PDB) related queries

// Container Status and Information
const (
	// ContainerStatusSQL returns status of all containers
	ContainerStatusSQL = `
		SELECT 
			CON_ID,
			NAME AS CONTAINER_NAME,
			OPEN_MODE,
			RESTRICTED,
			OPEN_TIME
		FROM GV$CONTAINERS
		WHERE ROWNUM <= 1000`

	// PDBStatusSQL returns status of all pluggable databases
	PDBStatusSQL = `
		SELECT 
			CON_ID,
			NAME AS PDB_NAME,
			CREATE_SCN,
			OPEN_MODE,
			RESTRICTED,
			OPEN_TIME,
			TOTAL_SIZE
		FROM GV$PDBS
		WHERE ROWNUM <= 1000`

	// CDBServicesSQL returns services across all containers
	CDBServicesSQL = `
		SELECT 
			con_id,
			name AS service_name,
			network_name,
			creation_date,
			pdb,
			enabled
		FROM CDB_SERVICES
		WHERE ROWNUM <= 1000`
)

// Container Metrics Queries
const (
	// ContainerSysMetricsSQL returns system metrics for containers
	ContainerSysMetricsSQL = `
		SELECT 
			con_id,
			metric_name,
			value,
			metric_unit,
			group_id
		FROM GV$CON_SYSMETRIC 
		WHERE group_id = 2
			AND ROWNUM <= 5000`

	// CDBSysMetricsSQL returns system metrics from CDB root
	CDBSysMetricsSQL = `
		SELECT 
			con_id,
			metric_name,
			value,
			metric_unit,
			group_id
		FROM GV$SYSMETRIC 
		WHERE group_id = 2
			AND ROWNUM <= 5000`
)
