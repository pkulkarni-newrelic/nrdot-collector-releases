// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Security Metrics Categories:
//
// 1. Server Principals Monitoring:
//   - Total server principals count (sys.server_principals)
//   - Authentication type breakdown (SQL vs Windows)
//   - Principal status tracking (enabled/disabled)
//   - System vs user-defined principals
//
// 2. Server Role Membership Monitoring:
//   - Total role members count (sys.server_role_members)
//   - Role membership distribution by role type
//   - Administrative privilege tracking
//   - Security hierarchy analysis
//
// 3. Authentication Activity:
//   - Login success/failure rates (sys.dm_exec_sessions)
//   - Session authentication patterns
//   - Active authenticated connections
//   - Authentication method usage
//
// Engine Support:
// - Standard SQL Server: Full security metrics from all DMVs
// - Azure SQL Database: Limited (no server-level roles, restricted access)
// - Azure SQL Managed Instance: Most metrics available with some limitations
//
// Security Considerations:
// - Queries only count principals/members, no sensitive data exposure
// - Compatible with SQL Server security best practices
// - Follows principle of least privilege for monitoring accounts
// - No exposure of usernames, passwords, or security tokens
package queries

const SecurityPrincipalsQuery = `SELECT COUNT(*) AS server_principals_count FROM sys.server_principals WITH (nolock)`

const SecurityRoleMembersQuery = `SELECT COUNT(*) AS server_role_members_count FROM sys.server_role_members WITH (nolock)`
