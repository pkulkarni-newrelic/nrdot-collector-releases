// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ComputeQueryHash computes a consistent hash identifier from anonymized query text.
// This enables correlation between:
// - Slow queries from dm_exec_query_stats (which have query_hash)
// - Active running queries from dm_exec_requests (which don't have query_hash)
//
// The approach:
// 1. Anonymize the query text (replace literals with placeholders)
// 2. Normalize whitespace and casing
// 3. Compute SHA256 hash
// 4. Return first 16 hex characters as identifier (64 bits, matching SQL Server's query_hash size)
//
// This hash becomes the common identifier to correlate slow queries with their active running instances.
func ComputeQueryHash(queryText string) string {
	if queryText == "" {
		return ""
	}

	// Step 1: Anonymize the query text to normalize it
	// This removes literals, making queries with different values but same structure match
	anonymized := AnonymizeQueryText(queryText)

	// Step 2: Normalize the anonymized text
	// - Convert to uppercase for case-insensitive matching
	// - Trim whitespace
	// - Collapse multiple spaces to single space
	normalized := normalizeQueryText(anonymized)

	// Step 3: Compute SHA256 hash
	hash := sha256.Sum256([]byte(normalized))

	// Step 4: Return first 16 hex characters (64 bits, matching SQL Server query_hash size)
	// Format as 0x prefix to match SQL Server convention
	return "0x" + hex.EncodeToString(hash[:8])
}

// normalizeQueryText normalizes whitespace and casing for consistent hashing
func normalizeQueryText(text string) string {
	// Convert to uppercase
	text = strings.ToUpper(text)

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	// Collapse multiple spaces/tabs/newlines to single space
	normalized := strings.Builder{}
	prevSpace := false

	for _, char := range text {
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			if !prevSpace {
				normalized.WriteRune(' ')
				prevSpace = true
			}
		} else {
			normalized.WriteRune(char)
			prevSpace = false
		}
	}

	return normalized.String()
}

// ComputeQueryHashFromBytes computes a query hash from raw byte data
// Useful when query text is already in byte format
func ComputeQueryHashFromBytes(queryBytes []byte) string {
	return ComputeQueryHash(string(queryBytes))
}
