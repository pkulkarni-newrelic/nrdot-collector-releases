// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// SQLIdentifier represents a unique SQL statement identifier with its child cursor number
// and the timestamp when the query was captured (for correlation with execution plans)
type SQLIdentifier struct {
	SQLID       string
	ChildNumber int64
	Timestamp   time.Time // Timestamp when the query was captured (from wait event or slow query)
}
