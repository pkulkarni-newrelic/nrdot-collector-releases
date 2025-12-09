// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeQueryHash(t *testing.T) {
	tests := []struct {
		name        string
		query1      string
		query2      string
		shouldMatch bool
		description string
	}{
		{
			name:        "identical_queries_same_hash",
			query1:      "SELECT * FROM users WHERE id = 123",
			query2:      "SELECT * FROM users WHERE id = 123",
			shouldMatch: true,
			description: "Identical queries should produce the same hash",
		},
		{
			name:        "same_structure_different_values_same_hash",
			query1:      "SELECT * FROM users WHERE id = 123",
			query2:      "SELECT * FROM users WHERE id = 456",
			shouldMatch: true,
			description: "Queries with same structure but different literal values should match (after anonymization)",
		},
		{
			name:        "different_whitespace_same_hash",
			query1:      "SELECT * FROM users WHERE id = 123",
			query2:      "SELECT    *    FROM    users    WHERE    id    =    123",
			shouldMatch: true,
			description: "Queries with different whitespace should produce the same hash",
		},
		{
			name:        "different_case_same_hash",
			query1:      "SELECT * FROM users WHERE id = 123",
			query2:      "select * from users where id = 123",
			shouldMatch: true,
			description: "Queries with different case should produce the same hash",
		},
		{
			name:        "different_structure_different_hash",
			query1:      "SELECT * FROM users WHERE id = 123",
			query2:      "SELECT * FROM orders WHERE id = 123",
			shouldMatch: false,
			description: "Queries with different structure should produce different hashes",
		},
		{
			name:        "string_literals_anonymized",
			query1:      "SELECT * FROM users WHERE name = 'John'",
			query2:      "SELECT * FROM users WHERE name = 'Jane'",
			shouldMatch: true,
			description: "String literals should be anonymized, producing same hash",
		},
		{
			name:        "multiple_parameters_same_hash",
			query1:      "SELECT * FROM users WHERE id = 123 AND age = 25",
			query2:      "SELECT * FROM users WHERE id = 999 AND age = 30",
			shouldMatch: true,
			description: "Multiple numeric parameters should be anonymized",
		},
		{
			name:        "empty_query",
			query1:      "",
			query2:      "",
			shouldMatch: true,
			description: "Empty queries should produce empty hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := ComputeQueryHash(tt.query1)
			hash2 := ComputeQueryHash(tt.query2)

			if tt.shouldMatch {
				assert.Equal(t, hash1, hash2, tt.description)
			} else {
				assert.NotEqual(t, hash1, hash2, tt.description)
			}

			// Verify hash format (should start with 0x and be 18 characters: "0x" + 16 hex chars)
			if tt.query1 != "" {
				assert.True(t, len(hash1) == 18, "Hash should be 18 characters (0x + 16 hex)")
				assert.True(t, hash1[:2] == "0x", "Hash should start with 0x")
			}
		})
	}
}

func TestComputeQueryHash_Format(t *testing.T) {
	query := "SELECT * FROM users WHERE id = 123"
	hash := ComputeQueryHash(query)

	// Should start with 0x
	assert.True(t, hash[:2] == "0x", "Hash should start with 0x prefix")

	// Should be 18 characters total (0x + 16 hex chars = 64 bits)
	assert.Equal(t, 18, len(hash), "Hash should be 18 characters (0x + 16 hex)")

	// Should be valid hex after 0x prefix
	_, err := hex.DecodeString(hash[2:])
	assert.NoError(t, err, "Hash should contain valid hex characters")
}

func TestComputeQueryHash_Consistency(t *testing.T) {
	query := "SELECT * FROM orders WHERE customer_id = 456 AND total > 100.50"

	// Compute hash multiple times
	hash1 := ComputeQueryHash(query)
	hash2 := ComputeQueryHash(query)
	hash3 := ComputeQueryHash(query)

	// All should be identical
	assert.Equal(t, hash1, hash2, "Hash should be consistent across multiple calls")
	assert.Equal(t, hash2, hash3, "Hash should be consistent across multiple calls")
}

func TestComputeQueryHash_EmptyString(t *testing.T) {
	hash := ComputeQueryHash("")
	assert.Equal(t, "", hash, "Empty query should produce empty hash")
}

func TestComputeQueryHashFromBytes(t *testing.T) {
	query := "SELECT * FROM products WHERE price = 99.99"
	queryBytes := []byte(query)

	hash1 := ComputeQueryHash(query)
	hash2 := ComputeQueryHashFromBytes(queryBytes)

	assert.Equal(t, hash1, hash2, "Hash from string and bytes should match")
}

func TestNormalizeQueryText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase_conversion",
			input:    "select * from users",
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "collapse_multiple_spaces",
			input:    "SELECT    *    FROM    users",
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "collapse_tabs_and_newlines",
			input:    "SELECT\t*\nFROM\r\nusers",
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "trim_whitespace",
			input:    "  SELECT * FROM users  ",
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "mixed_whitespace",
			input:    "  SELECT  \t  *  \n  FROM  \r\n  users  ",
			expected: "SELECT * FROM USERS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeQueryText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestQueryHashCorrelation verifies that the hash enables correlation between
// slow queries and active running queries with the same structure
func TestQueryHashCorrelation(t *testing.T) {
	// Simulate slow query from dm_exec_query_stats
	slowQuery := "SELECT customer_id, SUM(total) FROM orders WHERE order_date > '2024-01-01' GROUP BY customer_id"

	// Simulate active running query from dm_exec_requests (same structure, different date)
	activeQuery := "SELECT customer_id, SUM(total) FROM orders WHERE order_date > '2024-11-20' GROUP BY customer_id"

	slowQueryHash := ComputeQueryHash(slowQuery)
	activeQueryHash := ComputeQueryHash(activeQuery)

	// These should match because they have the same structure
	assert.Equal(t, slowQueryHash, activeQueryHash,
		"Slow query and active query with same structure should have matching hash for correlation")
}
