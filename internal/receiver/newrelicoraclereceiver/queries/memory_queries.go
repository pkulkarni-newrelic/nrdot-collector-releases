// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queries

// Oracle memory-related queries (SGA, PGA, UGA, buffer pools)

// PGA Memory Queries
const (
	// PGAMetricsSQL returns PGA memory statistics
	PGAMetricsSQL = `
		SELECT 
			INST_ID, 
			NAME, 
			VALUE 
		FROM gv$pgastat 
		WHERE NAME IN (
			'total PGA inuse', 
			'total PGA allocated', 
			'total freeable PGA memory', 
			'global memory bound'
		)`
)

// SGA Memory Queries
const (
	// SGASQL returns SGA component information
	SGASQL = `
		SELECT 
			inst.inst_id, 
			sga.name, 
			sga.value
		FROM GV$SGA sga, GV$INSTANCE inst
		WHERE sga.inst_id = inst.inst_id 
			AND sga.name IN ('Fixed Size', 'Redo Buffers')`

	// SGAUGATotalMemorySQL returns total UGA memory usage
	SGAUGATotalMemorySQL = `
		SELECT 
			SUM(value) AS sum, 
			inst.inst_id
		FROM GV$sesstat, GV$statname, GV$INSTANCE inst
		WHERE name = 'session uga memory max'
			AND GV$sesstat.statistic# = GV$statname.statistic#
			AND GV$sesstat.inst_id = inst.inst_id
			AND GV$statname.inst_id = inst.inst_id
		GROUP BY inst.inst_id`

	// SGAHitRatioSQL returns SGA buffer hit ratio
	SGAHitRatioSQL = `
		SELECT 
			inst.inst_id,
			(1 - (phy.value - lob.value - dir.value)/ses.value) AS ratio
		FROM GV$SYSSTAT ses, GV$SYSSTAT lob, GV$SYSSTAT dir, GV$SYSSTAT phy, GV$INSTANCE inst
		WHERE ses.name = 'session logical reads'
			AND dir.name = 'physical reads direct'
			AND lob.name = 'physical reads direct (lob)'
			AND phy.name = 'physical reads'
			AND ses.inst_id = inst.inst_id
			AND lob.inst_id = inst.inst_id
			AND dir.inst_id = inst.inst_id
			AND phy.inst_id = inst.inst_id`
)

// Shared Pool Queries
const (
	// SGASharedPoolLibraryCacheShareableStatementSQL returns shareable statement memory
	SGASharedPoolLibraryCacheShareableStatementSQL = `
		SELECT 
			SUM(sqlarea.sharable_mem) AS sum, 
			inst.inst_id
		FROM GV$sqlarea sqlarea, GV$INSTANCE inst
		WHERE sqlarea.executions > 5
			AND inst.inst_id = sqlarea.inst_id
		GROUP BY inst.inst_id`

	// SGASharedPoolLibraryCacheShareableUserSQL returns shareable user memory
	SGASharedPoolLibraryCacheShareableUserSQL = `
		SELECT 
			SUM(250 * sqlarea.users_opening) AS sum, 
			inst.inst_id
		FROM GV$sqlarea sqlarea, GV$INSTANCE inst
		WHERE inst.inst_id = sqlarea.inst_id
		GROUP BY inst.inst_id`

	// SGASharedPoolLibraryCacheReloadRatioSQL returns library cache reload ratio
	SGASharedPoolLibraryCacheReloadRatioSQL = `
		SELECT 
			(SUM(libcache.reloads)/SUM(libcache.pins)) AS ratio, 
			inst.inst_id
		FROM GV$librarycache libcache, GV$INSTANCE inst
		WHERE inst.inst_id = libcache.inst_id
		GROUP BY inst.inst_id`

	// SGASharedPoolLibraryCacheHitRatioSQL returns library cache hit ratio
	SGASharedPoolLibraryCacheHitRatioSQL = `
		SELECT 
			libcache.gethitratio AS ratio, 
			inst.inst_id
		FROM GV$librarycache libcache, GV$INSTANCE inst
		WHERE namespace = 'SQL AREA'
			AND inst.inst_id = libcache.inst_id`

	// SGASharedPoolDictCacheMissRatioSQL returns dictionary cache miss ratio
	SGASharedPoolDictCacheMissRatioSQL = `
		SELECT 
			(SUM(rcache.getmisses)/SUM(rcache.gets)) AS ratio, 
			inst.inst_id
		FROM GV$rowcache rcache, GV$INSTANCE inst
		WHERE inst.inst_id = rcache.inst_id
		GROUP BY inst.inst_id`
)

// Buffer and Log Memory Queries
const (
	// SGALogBufferSpaceWaitsSQL returns log buffer space waits
	SGALogBufferSpaceWaitsSQL = `
		SELECT 
			COUNT(wait.inst_id) AS count, 
			inst.inst_id
		FROM GV$SESSION_WAIT wait, GV$INSTANCE inst
		WHERE wait.event LIKE 'log buffer space%'
			AND inst.inst_id = wait.inst_id
		GROUP BY inst.inst_id`

	// SGALogAllocRetriesSQL returns log allocation retry ratio
	SGALogAllocRetriesSQL = `
		SELECT 
			(rbar.value/re.value) AS ratio, 
			inst.inst_id
		FROM GV$SYSSTAT rbar, GV$SYSSTAT re, GV$INSTANCE inst
		WHERE rbar.name LIKE 'redo buffer allocation retries'
			AND re.name LIKE 'redo entries'
			AND re.inst_id = inst.inst_id 
			AND rbar.inst_id = inst.inst_id`
)
