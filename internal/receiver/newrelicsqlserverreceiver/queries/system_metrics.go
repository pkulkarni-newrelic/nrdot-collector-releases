// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

const SystemInformationQuery = `
SET DEADLOCK_PRIORITY -10;

-- Handle different SQL Server editions
IF SERVERPROPERTY('EngineEdition') NOT IN (2,3,4,5,8) BEGIN /*NOT IN Standard,Enterprise,Express,Azure SQL Database, Azure SQL Managed Instance*/
	DECLARE @ErrorMessage AS nvarchar(500) = 'Connection string Server:'+ @@ServerName + ',Database:' + DB_NAME() +' is not a SQL Server Standard, Enterprise, Express, Azure SQL Database or Azure SQL Managed Instance. This query is only supported on these editions.';
	RAISERROR (@ErrorMessage,11,1)
	RETURN
END

SELECT
	 REPLACE(@@SERVERNAME,'\',':') AS [sql_instance]
	,HOST_NAME() AS [computer_name]
	,CASE 
		WHEN SERVERPROPERTY('EngineEdition') = 5 THEN 'Azure SQL Database' 
		WHEN SERVERPROPERTY('EngineEdition') = 8 THEN 'Azure SQL Managed Instance'
		ELSE 'SQL Server'
	END AS [service_name]
	,si.[cpu_count]
	,NULL AS [server_memory]  -- Not consistently available across all editions
	,NULL AS [available_server_memory]  -- Not consistently available across all editions
	,SERVERPROPERTY('Edition') AS [sku]
	,CAST(SERVERPROPERTY('EngineEdition') AS int) AS [engine_edition]
	,DATEDIFF(MINUTE,si.[sqlserver_start_time],GETDATE()) AS [uptime]
	,SERVERPROPERTY('ProductVersion') AS [sql_version]
	,SERVERPROPERTY('IsClustered') AS [instance_type]
	,SERVERPROPERTY('IsHadrEnabled') AS [is_hadr_enabled]
	,LEFT(@@VERSION,CHARINDEX(' - ',@@VERSION)) AS [sql_version_desc]
	,NULL AS [ForceEncryption]  -- Registry access not available in Azure SQL
	,CASE 
		WHEN SERVERPROPERTY('EngineEdition') IN (5,8) THEN '1433'  -- Standard Azure SQL port
		ELSE NULL  -- Unknown for on-premises without registry access
	END AS [Port]
	,CASE 
		WHEN SERVERPROPERTY('EngineEdition') IN (5,8) THEN 'Azure'
		ELSE 'Unknown'
	END AS [PortType]
	,(si.[ms_ticks]/1000) AS [computer_uptime]
	FROM sys.[dm_os_sys_info] AS si`
