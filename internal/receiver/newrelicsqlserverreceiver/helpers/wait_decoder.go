// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"regexp"
	"strings"
)

// DecodeWaitType returns a human-readable description for SQL Server wait types
func DecodeWaitType(waitType string) string {
	if waitType == "" || waitType == "N/A" {
		return "Not Waiting"
	}

	waitTypeMap := getSQLServerWaitTypeMap()

	if description, exists := waitTypeMap[waitType]; exists {
		return description
	}

	// Handle prefix patterns
	if strings.HasPrefix(waitType, "PREEMPTIVE_") {
		return fmt.Sprintf("OS Call - %s", strings.TrimPrefix(waitType, "PREEMPTIVE_"))
	}
	if strings.HasPrefix(waitType, "HADR_") {
		return fmt.Sprintf("AlwaysOn - %s", strings.TrimPrefix(waitType, "HADR_"))
	}
	if strings.HasPrefix(waitType, "LCK_M_") {
		return fmt.Sprintf("Lock - %s", strings.TrimPrefix(waitType, "LCK_M_"))
	}
	if strings.HasPrefix(waitType, "PAGEIOLATCH_") {
		return fmt.Sprintf("Disk I/O - Page Latch %s", strings.TrimPrefix(waitType, "PAGEIOLATCH_"))
	}

	// Return original if unknown
	return waitType
}

// GetWaitTypeCategory returns the high-level category for a wait type
func GetWaitTypeCategory(waitType string) string {
	if waitType == "" || waitType == "N/A" {
		return "None"
	}

	switch {
	case strings.HasPrefix(waitType, "CXPACKET") || strings.HasPrefix(waitType, "CXSYNC") || strings.HasPrefix(waitType, "CXCONSUMER") || waitType == "EXECSYNC" || waitType == "EXCHANGE":
		return "Parallelism"
	case strings.HasPrefix(waitType, "PAGEIOLATCH") || strings.HasPrefix(waitType, "WRITELOG") || strings.HasPrefix(waitType, "LOGBUFFER") || strings.HasPrefix(waitType, "IO_"):
		return "Disk I/O"
	case strings.HasPrefix(waitType, "LCK_M_"):
		return "Locking"
	case strings.HasPrefix(waitType, "RESOURCE_SEMAPHORE") || strings.HasPrefix(waitType, "CMEMTHREAD"):
		return "Memory"
	case strings.HasPrefix(waitType, "ASYNC_NETWORK_IO") || strings.HasPrefix(waitType, "NETWORK"):
		return "Network"
	case waitType == "SOS_SCHEDULER_YIELD" || waitType == "THREADPOOL":
		return "CPU"
	case strings.HasPrefix(waitType, "LATCH_") || strings.HasPrefix(waitType, "PAGELATCH_"):
		return "Latch"
	case strings.HasPrefix(waitType, "PREEMPTIVE_"):
		return "OS Call"
	case strings.HasPrefix(waitType, "HADR_"):
		return "AlwaysOn"
	case strings.HasPrefix(waitType, "BACKUP"):
		return "Backup"
	case strings.HasPrefix(waitType, "DTC") || strings.HasPrefix(waitType, "XACT"):
		return "Transaction"
	case waitType == "SLEEP_TASK" || waitType == "SLEEP_BPOOL_STEAL":
		return "Sleep"
	default:
		return "Other"
	}
}

// DecodeWaitResource parses and returns human-readable wait resource information
// Based on sys.dm_tran_locks resource_description format
// DecodeWaitResource parses wait_resource and returns resource type and description
// Only decodes table names when OBJECT identifier is available
func DecodeWaitResource(waitResource string) (resourceType string, description string) {
	if waitResource == "" {
		return "", ""
	}

	// KEY: <database_id>:<hobt_id> (<hash_value>)
	// Example: KEY: 5:72057594049986560 (419e4517fb6a)
	keyPattern := regexp.MustCompile(`^KEY:\s*(\d+):(\d+)\s*\(([a-f0-9]+)\)`)
	if matches := keyPattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		hobtID := matches[2]
		hashValue := matches[3]
		desc := fmt.Sprintf("Database ID: %s | HOBT: %s | Key Hash: %s", dbID, hobtID, hashValue)
		return "Key Lock", desc
	}

	// PAGE: <database_id>:<file_id>:<page_id>
	// Example: PAGE: 5:1:104
	pagePattern := regexp.MustCompile(`^PAGE:\s*(\d+):(\d+):(\d+)`)
	if matches := pagePattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		fileID := matches[2]
		pageID := matches[3]
		desc := fmt.Sprintf("Database ID: %s | File: %s | Page: %s", dbID, fileID, pageID)
		return "Page Lock", desc
	}

	// RID: <database_id>:<file_id>:<page_id>:<row_on_page>
	// Example: RID: 5:1:104:0
	ridPattern := regexp.MustCompile(`^RID:\s*(\d+):(\d+):(\d+):(\d+)`)
	if matches := ridPattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		fileID := matches[2]
		pageID := matches[3]
		rowSlot := matches[4]
		desc := fmt.Sprintf("Database ID: %s | File: %s | Page: %s | Row: %s", dbID, fileID, pageID, rowSlot)
		return "Row Lock", desc
	}

	// OBJECT: <database_id>:<object_id>:<lock_partition>
	// Example: OBJECT: 5:245575913:0
	objectPattern := regexp.MustCompile(`^OBJECT:\s*(\d+):(\d+):(\d+)`)
	if matches := objectPattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		objectID := matches[2]
		partition := matches[3]
		desc := fmt.Sprintf("Database ID: %s | Object ID: %s | Partition: %s", dbID, objectID, partition)
		return "Object Lock", desc
	}

	// DATABASE: <database_id>
	// Example: DATABASE: 5
	databasePattern := regexp.MustCompile(`^DATABASE:\s*(\d+)`)
	if matches := databasePattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		desc := fmt.Sprintf("Database ID: %s", dbID)
		return "Database Lock", desc
	}

	// FILE: <database_id>:<file_id>
	// Example: FILE: 5:1
	filePattern := regexp.MustCompile(`^FILE:\s*(\d+):(\d+)`)
	if matches := filePattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		fileID := matches[2]
		desc := fmt.Sprintf("Database ID: %s | File: %s", dbID, fileID)
		return "File Lock", desc
	}

	// EXTENT: <database_id>:<file_id>:<page_id>
	// Example: EXTENT: 5:1:104
	extentPattern := regexp.MustCompile(`^EXTENT:\s*(\d+):(\d+):(\d+)`)
	if matches := extentPattern.FindStringSubmatch(waitResource); matches != nil {
		dbID := matches[1]
		fileID := matches[2]
		pageID := matches[3]
		desc := fmt.Sprintf("Database ID: %s | File: %s | First Page: %s", dbID, fileID, pageID)
		return "Extent Lock", desc
	}

	// HOBT: Heap or B-tree
	// Example: HOBT: 72057594049986560
	hobtPattern := regexp.MustCompile(`^HOBT:\s*(\d+)`)
	if matches := hobtPattern.FindStringSubmatch(waitResource); matches != nil {
		hobtID := matches[1]
		desc := fmt.Sprintf("HOBT ID: %s", hobtID)
		return "Heap/B-Tree Lock", desc
	}

	// APPLICATION: <DbPrincipalId>:<up to 32 characters>:(<hash_value>)
	// Example: APPLICATION: 1:MyAppLock:(abc123)
	applicationPattern := regexp.MustCompile(`^APPLICATION:\s*(\d+):([^:]+):\(([a-f0-9]+)\)`)
	if matches := applicationPattern.FindStringSubmatch(waitResource); matches != nil {
		principalID := matches[1]
		lockName := matches[2]
		hashValue := matches[3]
		return "Application Lock", fmt.Sprintf("Principal: %s | Lock Name: %s | Hash: %s", principalID, lockName, hashValue)
	}

	// Synchronization Object: 0x<hex_value>
	// Example: 0x82f2de703704dd7c (used in parallel query coordination)
	syncObjectPattern := regexp.MustCompile(`^0x([a-f0-9]+)$`)
	if matches := syncObjectPattern.FindStringSubmatch(waitResource); matches != nil {
		hexValue := matches[1]
		return "Synchronization Object", fmt.Sprintf("Parallel Query Sync Object: 0x%s", hexValue)
	}

	// METADATA: <metadata_class>:<metadata_id>
	// Example: METADATA: 21:5
	metadataPattern := regexp.MustCompile(`^METADATA:\s*(\d+):(\d+)`)
	if matches := metadataPattern.FindStringSubmatch(waitResource); matches != nil {
		metadataClass := matches[1]
		metadataID := matches[2]

		// Decode metadata class to human-readable name
		className := getMetadataClassName(metadataClass)
		desc := fmt.Sprintf("Metadata Class: %s (%s) | ID: %s", metadataClass, className, metadataID)

		return "Metadata Lock", desc
	}

	// ALLOCATION_UNIT: <allocation_unit_id>
	// Example: ALLOCATION_UNIT: 72057594044612608
	allocationUnitPattern := regexp.MustCompile(`^ALLOCATION_UNIT:\s*(\d+)`)
	if matches := allocationUnitPattern.FindStringSubmatch(waitResource); matches != nil {
		auID := matches[1]
		desc := fmt.Sprintf("Allocation Unit ID: %s", auID)
		return "Allocation Unit Lock", desc
	}

	// Unknown format - return as-is
	return "Unknown", waitResource
}

// getMetadataClassName returns human-readable name for metadata class ID
// Based on sys.dm_tran_locks documentation
func getMetadataClassName(classID string) string {
	metadataClasses := map[string]string{
		"0":  "DATABASE",
		"1":  "OBJECT",
		"2":  "INDEX",
		"3":  "STAT",
		"4":  "COLUMN",
		"5":  "KEY",
		"6":  "USER",
		"7":  "ROLE",
		"8":  "PARTITION_FUNCTION",
		"9":  "PARTITION_SCHEME",
		"10": "FILEGROUP",
		"11": "XML_SCHEMA_COLLECTION",
		"12": "SCHEMA",
		"13": "ASSEMBLY",
		"14": "DEFAULT_CONSTRAINT",
		"15": "CHECK_CONSTRAINT",
		"16": "FOREIGN_KEY_CONSTRAINT",
		"17": "PRIMARY_KEY_CONSTRAINT",
		"18": "UNIQUE_CONSTRAINT",
		"19": "SERVICE",
		"20": "QUEUE",
		"21": "DATABASE_DDL_TRIGGER",
		"22": "BROKER_PRIORITY",
		"23": "FULL_TEXT_CATALOG",
		"24": "FULL_TEXT_STOPLIST",
		"25": "SEARCH_PROPERTY_LIST",
		"26": "SEQUENCE",
	}

	if name, ok := metadataClasses[classID]; ok {
		return name
	}
	return ""
}

// ParseHOBTIDFromWaitResource extracts the HOBT ID from a KEY lock wait_resource string
// KEY wait_resource format: "KEY: <db_id>:<hobt_id> (<key_hash>)"
// Example: "KEY: 5:72057594042908672 (e5e11ab44f5d)"
// Returns 0 if parsing fails or format is invalid
func ParseHOBTIDFromWaitResource(waitResource string) int64 {
	if !strings.HasPrefix(waitResource, "KEY:") {
		return 0
	}

	// Use regex to extract HOBT ID: "KEY: <db_id>:<hobt_id> ("
	// Pattern: KEY: \d+:(\d+) \(
	keyRegex := regexp.MustCompile(`KEY:\s*\d+:(\d+)\s*\(`)
	matches := keyRegex.FindStringSubmatch(waitResource)

	if len(matches) < 2 {
		return 0
	}

	// Parse HOBT ID from captured group
	var hobtID int64
	_, err := fmt.Sscanf(matches[1], "%d", &hobtID)
	if err != nil {
		return 0
	}

	return hobtID
}
