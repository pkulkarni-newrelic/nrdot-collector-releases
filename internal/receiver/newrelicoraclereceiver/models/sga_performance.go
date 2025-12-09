// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "database/sql"

// SGASharedPoolLibraryCacheReloadRatioMetric represents library cache reload ratio
type SGASharedPoolLibraryCacheReloadRatioMetric struct {
	Ratio  float64
	InstID interface{}
}

// SGASharedPoolLibraryCacheHitRatioMetric represents library cache hit ratio
type SGASharedPoolLibraryCacheHitRatioMetric struct {
	Ratio  float64
	InstID interface{}
}

// SGASharedPoolDictCacheMissRatioMetric represents dictionary cache miss ratio
type SGASharedPoolDictCacheMissRatioMetric struct {
	Ratio  float64
	InstID interface{}
}

// SGALogBufferSpaceWaitsMetric represents log buffer space waits
type SGALogBufferSpaceWaitsMetric struct {
	Count  int64
	InstID interface{}
}

// SGALogAllocRetriesMetric represents log allocation retry ratio
type SGALogAllocRetriesMetric struct {
	Ratio  sql.NullFloat64
	InstID interface{}
}

// SGAHitRatioMetric represents SGA buffer hit ratio
type SGAHitRatioMetric struct {
	InstID interface{}
	Ratio  sql.NullFloat64
}
