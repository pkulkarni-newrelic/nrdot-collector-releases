// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstanceIDString_Nil(t *testing.T) {
	result := getInstanceIDString(nil)
	assert.Equal(t, "unknown", result)
}

func TestGetInstanceIDString_Int64(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "positive int64",
			input:    123456789,
			expected: "123456789",
		},
		{
			name:     "negative int64",
			input:    -987654321,
			expected: "-987654321",
		},
		{
			name:     "zero int64",
			input:    0,
			expected: "0",
		},
		{
			name:     "max int64",
			input:    9223372036854775807,
			expected: "9223372036854775807",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstanceIDString_Int32(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected string
	}{
		{
			name:     "positive int32",
			input:    12345,
			expected: "12345",
		},
		{
			name:     "negative int32",
			input:    -54321,
			expected: "-54321",
		},
		{
			name:     "zero int32",
			input:    0,
			expected: "0",
		},
		{
			name:     "max int32",
			input:    2147483647,
			expected: "2147483647",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstanceIDString_Int(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{
			name:     "positive int",
			input:    42,
			expected: "42",
		},
		{
			name:     "negative int",
			input:    -99,
			expected: "-99",
		},
		{
			name:     "zero int",
			input:    0,
			expected: "0",
		},
		{
			name:     "large int",
			input:    999999,
			expected: "999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstanceIDString_String(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "instance1",
			expected: "instance1",
		},
		{
			name:     "numeric string",
			input:    "123",
			expected: "123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with spaces",
			input:    "instance 1",
			expected: "instance 1",
		},
		{
			name:     "string with special chars",
			input:    "inst@nce#1",
			expected: "inst@nce#1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstanceIDString_ByteSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "simple byte slice",
			input:    []byte("instance1"),
			expected: "instance1",
		},
		{
			name:     "numeric byte slice",
			input:    []byte("456"),
			expected: "456",
		},
		{
			name:     "empty byte slice",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "byte slice with special chars",
			input:    []byte("test@123"),
			expected: "test@123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstanceIDString_DefaultCase(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "float64",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "float32",
			input:    float32(2.71),
			expected: "2.71",
		},
		{
			name:     "bool true",
			input:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			input:    false,
			expected: "false",
		},
		{
			name:     "struct",
			input:    struct{ ID int }{ID: 123},
			expected: "{123}",
		},
		{
			name:     "pointer",
			input:    new(int),
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.NotEmpty(t, result)
			if tt.name != "pointer" {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetInstanceIDString_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "nil interface",
			input: nil,
		},
		{
			name:  "nil pointer",
			input: (*int)(nil),
		},
		{
			name:  "nil slice",
			input: ([]byte)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.NotEmpty(t, result)
		})
	}
}

func TestGetInstanceIDString_LargeNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "large positive int64",
			input:    int64(9223372036854775807),
			expected: "9223372036854775807",
		},
		{
			name:     "large negative int64",
			input:    int64(-9223372036854775808),
			expected: "-9223372036854775808",
		},
		{
			name:     "large positive int32",
			input:    int32(2147483647),
			expected: "2147483647",
		},
		{
			name:     "large negative int32",
			input:    int32(-2147483648),
			expected: "-2147483648",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstanceIDString_UnicodeAndSpecialStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unicode characters",
			input:    "实例1",
			expected: "实例1",
		},
		{
			name:     "emoji",
			input:    "instance🚀1",
			expected: "instance🚀1",
		},
		{
			name:     "newline",
			input:    "instance\n1",
			expected: "instance\n1",
		},
		{
			name:     "tab",
			input:    "instance\t1",
			expected: "instance\t1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceIDString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
