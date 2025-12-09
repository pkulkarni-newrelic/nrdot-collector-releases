// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

// LockedAccountsMetric represents locked accounts count per instance
type LockedAccountsMetric struct {
	InstID         interface{} // Instance ID (can be int or string)
	LockedAccounts int64       // Number of locked accounts
}

// GlobalNameMetric represents global database name per instance
type GlobalNameMetric struct {
	InstID     interface{} // Instance ID (can be int or string)
	GlobalName string      // Global database name
}

// DBIDMetric represents database ID per instance
type DBIDMetric struct {
	InstID interface{} // Instance ID (can be int or string)
	DBID   string      // Database ID
}

// LongRunningQueriesMetric represents long-running queries count per instance
type LongRunningQueriesMetric struct {
	InstID interface{} // Instance ID (can be int or string)
	Total  int64       // Total count of long-running queries
}
