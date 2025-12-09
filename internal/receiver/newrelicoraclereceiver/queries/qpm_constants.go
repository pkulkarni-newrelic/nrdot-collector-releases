// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// QPM (Query Performance Monitoring) configuration constants
const (
	// Default values used when configuration is not specified
	DefaultQueryMonitoringResponseTimeThreshold = 100 // milliseconds
	DefaultQueryMonitoringCountThreshold        = 20  

	// Validation ranges for configuration values
	MinQueryMonitoringResponseTimeThreshold = 0  
	MinQueryMonitoringCountThreshold        = 20 
	MaxQueryMonitoringCountThreshold        = 50
)
