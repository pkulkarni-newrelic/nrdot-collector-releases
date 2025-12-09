// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

type TablespaceUsage struct {
	TablespaceName string
	UsedPercent    float64
	Used           float64
	Size           float64
	Offline        float64
}

type TablespaceGlobalName struct {
	TablespaceName string
	GlobalName     string
}

type TablespaceDBID struct {
	TablespaceName string
	DBID           int64
}

type TablespaceCDBDatafilesOffline struct {
	TablespaceName string
	OfflineCount   int64
}

type TablespacePDBDatafilesOffline struct {
	TablespaceName string
	OfflineCount   int64
}

type TablespacePDBNonWrite struct {
	TablespaceName string
	NonWriteCount  int64
}
