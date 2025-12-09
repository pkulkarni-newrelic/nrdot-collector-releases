// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0


package models

// SecurityPrincipalsModel represents server principals count metrics for SQL Server security monitoring
type SecurityPrincipalsModel struct {
	ServerPrincipalsCount *int64 `db:"server_principals_count" metric_name:"sqlserver.security.server_principals_count" source_type:"gauge" description:"Total number of server principals (logins)" unit:"1"`
}

// SecurityRoleMembersModel represents server role membership metrics for SQL Server security monitoring
type SecurityRoleMembersModel struct {
	ServerRoleMembersCount *int64 `db:"server_role_members_count" metric_name:"sqlserver.security.server_role_members_count" source_type:"gauge" description:"Total number of server role members" unit:"1"`
}
