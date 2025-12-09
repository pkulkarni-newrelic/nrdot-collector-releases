// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "database/sql"

// PGAMetric represents PGA memory statistics per instance
type PGAMetric struct {
	InstID interface{} // Instance ID (can be int or string)
	Name   string      // Metric name
	Value  float64     // Metric value
}

// SGAUGATotalMemoryMetric represents total UGA memory per instance
type SGAUGATotalMemoryMetric struct {
	InstID interface{} // Instance ID (can be int or string)
	Sum    int64       // Total UGA memory
}

// SGASharedPoolLibraryCacheMetric represents shared pool library cache memory per instance
type SGASharedPoolLibraryCacheMetric struct {
	InstID interface{} // Instance ID (can be int or string)
	Sum    int64       // Total shareable statement memory
}

// SGASharedPoolLibraryCacheUserMetric represents shared pool library cache user memory per instance
type SGASharedPoolLibraryCacheUserMetric struct {
	InstID interface{} // Instance ID (can be int or string)
	Sum    int64       // Total shareable user memory
}

// SGAMetric represents SGA component information per instance
type SGAMetric struct {
	InstID interface{}   // Instance ID (can be int or string)
	Name   string        // SGA component name
	Value  sql.NullInt64 // Component value (can be null)
}
