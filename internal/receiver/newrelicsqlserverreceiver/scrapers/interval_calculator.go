// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package scrapers provides simplified interval-based delta calculator for slow query detection
//
// Problem Statement:
// Cumulative averages from DB performance schemas (total_elapsed_time / execution_count)
// change very slowly when queries are optimized or degrade, causing:
// - False positives (slow queries still flagged after optimization)
// - Delayed detection of new performance issues
// - Inefficient resource usage
//
// Solution:
// Interval-based delta calculations that show what happened in the LAST polling interval,
// not cumulative since plan cache. This provides immediate visibility into changes.
//
// Algorithm:
// 1. First Scrape: Use historical (cumulative) average, filter by threshold
// 2. Subsequent Scrapes: Calculate delta between current and previous values
//   - interval_avg = (current_total_elapsed - prev_total_elapsed) / (current_exec_count - prev_exec_count)
//
// 3. Emit both interval average AND historical average
// 4. Only emit metrics if interval average > threshold (memory efficient)
// 5. Eviction: Only TTL-based (inactive queries), NOT threshold-based
//   - This preserves delta calculation ability when queries oscillate
//
// Example (threshold = 100ms):
// Scrape 1: 100 execs, 500ms total → cumulative avg = 5ms → not slow
// Scrape 2: 120 execs, 700ms total → interval avg = (700-500)/(120-100) = 10ms → not slow
// Scrape 5: 180 execs, 21,100ms total → interval avg = 20,000/20 = 1,000ms → SLOW! (immediate detection)
// Scrape 7: 220 execs, 43,100ms total → interval avg = 2,000/20 = 100ms → at threshold
// Scrape 9: 241 execs, 45,600ms total → interval avg = 500/1 = 500ms → SLOW! (outlier detected)
package scrapers

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
)

// SimplifiedQueryState tracks previous scrape data for delta calculation
type SimplifiedQueryState struct {
	// Previous cumulative values from DB DMV
	PrevExecutionCount     int64
	PrevTotalElapsedTimeMs float64 // milliseconds (from SQL Server directly, no precision loss)

	// Timestamps for TTL-based cleanup
	LastSeenTimestamp  time.Time
	FirstSeenTimestamp time.Time
}

// SimplifiedIntervalMetrics holds both interval and historical metrics
type SimplifiedIntervalMetrics struct {
	// Interval-based metrics (delta calculation) - ONLY for elapsed time
	IntervalAvgElapsedTimeMs float64
	IntervalExecutionCount   int64

	// Historical metrics (cumulative from DB) - all metrics
	HistoricalAvgElapsedTimeMs float64
	HistoricalAvgCPUTimeMs     float64
	HistoricalExecutionCount   int64

	// Metadata
	IsFirstScrape        bool    // True if this is the first time seeing this query
	HasNewExecutions     bool    // True if delta_exec_count > 0
	TimeSinceLastExecSec float64 // Time since last execution from DB (for staleness detection)
}

// SimplifiedIntervalCalculator implements simplified delta-based calculation
type SimplifiedIntervalCalculator struct {
	logger *zap.Logger

	// State cache - tracks ALL queries, not just slow ones
	// Eviction is ONLY based on TTL (inactivity), NOT threshold
	stateCache      map[string]*SimplifiedQueryState
	stateCacheMutex sync.RWMutex

	// Configuration
	cacheTTL         time.Duration // TTL for inactive queries (default: 10 minutes)
	lastCacheCleanup time.Time
}

// NewSimplifiedIntervalCalculator creates a simplified interval calculator
func NewSimplifiedIntervalCalculator(logger *zap.Logger, cacheTTL time.Duration) *SimplifiedIntervalCalculator {
	if cacheTTL <= 0 {
		logger.Warn("Invalid cache TTL, using default 10 minutes", zap.Duration("provided", cacheTTL))
		cacheTTL = 10 * time.Minute
	}

	return &SimplifiedIntervalCalculator{
		logger:           logger,
		stateCache:       make(map[string]*SimplifiedQueryState),
		cacheTTL:         cacheTTL,
		lastCacheCleanup: time.Now(),
	}
}

// CalculateMetrics calculates both interval and historical metrics
// Returns nil if query should be skipped (e.g., nil QueryID)
func (sic *SimplifiedIntervalCalculator) CalculateMetrics(query *models.SlowQuery, now time.Time) *SimplifiedIntervalMetrics {
	if query == nil || query.QueryID == nil || query.QueryID.IsEmpty() {
		return nil
	}

	queryID := query.QueryID.String()

	// IMPORTANT: Cache key MUST include both query_hash AND plan_handle
	// Per SQL Server documentation: dm_exec_query_stats returns "one row per query statement within the cached plan"
	// Stats are tracked per plan_handle, not per query_hash
	// Same query (query_hash) can have multiple plans (plan_handles) with independent counters
	var cacheKey string
	if query.PlanHandle != nil && !query.PlanHandle.IsEmpty() {
		cacheKey = queryID + "|" + query.PlanHandle.String()
	} else {
		// Fallback: if plan_handle is missing, use query_hash only (shouldn't happen in normal operation)
		cacheKey = queryID
		sic.logger.Warn("Missing plan_handle for query - using query_hash only as cache key",
			zap.String("query_id", queryID))
	}

	// Extract current cumulative values from query
	currentExecCount := getInt64FromPtr(query.ExecutionCount)
	// Use total_elapsed_time_ms directly from SQL Server (no precision loss from reconstruction)
	currentTotalElapsedMs := getFloat64FromPtr(query.TotalElapsedTimeMS)

	// Historical (cumulative) averages
	historicalAvgElapsedMs := getFloat64FromPtr(query.AvgElapsedTimeMS)
	historicalAvgCPUMs := getFloat64FromPtr(query.AvgCPUTimeMS)

	// Calculate time since last execution from DB timestamp
	timeSinceLastExec := 0.0
	if query.LastExecutionTimestamp != nil {
		if lastExecTime, err := time.Parse(time.RFC3339, *query.LastExecutionTimestamp); err == nil {
			timeSinceLastExec = now.Sub(lastExecTime).Seconds()
		}
	}

	// Lock for state cache access
	sic.stateCacheMutex.Lock()
	defer sic.stateCacheMutex.Unlock()

	// Check if query exists in cache (using query_hash + plan_handle as key)
	state, exists := sic.stateCache[cacheKey]

	// FIRST SCRAPE: Use historical (cumulative) average
	if !exists {
		// Handle edge case: execution_count = 0 from DMV
		// This shouldn't happen due to SQL filter (qs.execution_count > 0), but be defensive
		if currentExecCount == 0 {
			sic.logger.Warn("Query with zero execution count - skipping",
				zap.String("query_id", queryID),
				zap.String("plan_handle", query.PlanHandle.String()),
				zap.Float64("total_elapsed_ms", currentTotalElapsedMs),
				zap.Float64("historical_avg_ms", historicalAvgElapsedMs))
			return nil
		}

		sic.logger.Debug("First scrape for query - using historical average",
			zap.String("query_id", queryID),
			zap.String("plan_handle", query.PlanHandle.String()),
			zap.Int64("execution_count", currentExecCount),
			zap.Float64("historical_avg_ms", historicalAvgElapsedMs),
			zap.Float64("total_elapsed_ms", currentTotalElapsedMs))

		// Add to cache for next scrape (using query_hash + plan_handle as key)
		sic.stateCache[cacheKey] = &SimplifiedQueryState{
			PrevExecutionCount:     currentExecCount,
			PrevTotalElapsedTimeMs: currentTotalElapsedMs,
			FirstSeenTimestamp:     now,
			LastSeenTimestamp:      now,
		}

		// For first scrape, use historical average as interval average
		// Caller will filter by threshold
		return &SimplifiedIntervalMetrics{
			IntervalAvgElapsedTimeMs:   historicalAvgElapsedMs,
			IntervalExecutionCount:     currentExecCount,
			HistoricalAvgElapsedTimeMs: historicalAvgElapsedMs,
			HistoricalAvgCPUTimeMs:     historicalAvgCPUMs,
			HistoricalExecutionCount:   currentExecCount,
			IsFirstScrape:              true,
			HasNewExecutions:           true,
			TimeSinceLastExecSec:       timeSinceLastExec,
		}
	}

	// SUBSEQUENT SCRAPES: Calculate delta

	// Calculate deltas
	deltaExecCount := currentExecCount - state.PrevExecutionCount
	deltaElapsedMs := currentTotalElapsedMs - state.PrevTotalElapsedTimeMs

	// Handle no new executions
	if deltaExecCount == 0 {
		sic.logger.Debug("No new executions for query",
			zap.String("query_id", queryID),
			zap.String("plan_handle", query.PlanHandle.String()),
			zap.Float64("time_since_last_exec_sec", timeSinceLastExec))

		// Update last seen timestamp (query is still in DB results)
		state.LastSeenTimestamp = now

		// Return metrics but flag as no new executions
		// Caller can decide whether to emit
		return &SimplifiedIntervalMetrics{
			IntervalAvgElapsedTimeMs:   0,
			IntervalExecutionCount:     0,
			HistoricalAvgElapsedTimeMs: historicalAvgElapsedMs,
			HistoricalAvgCPUTimeMs:     historicalAvgCPUMs,
			HistoricalExecutionCount:   currentExecCount,
			IsFirstScrape:              false,
			HasNewExecutions:           false,
			TimeSinceLastExecSec:       timeSinceLastExec,
		}
	}

	// Handle plan cache reset (execution count decreased) OR stats corruption (negative delta elapsed)
	// Fix #1: Validate for negative deltas
	// Fix #4: Preserve FirstSeenTimestamp on reset
	if deltaExecCount < 0 || deltaElapsedMs < 0 {
		if deltaExecCount < 0 {
			sic.logger.Warn("Plan cache reset detected - execution count decreased",
				zap.String("query_id", queryID),
				zap.String("plan_handle", query.PlanHandle.String()),
				zap.Int64("current_exec_count", currentExecCount),
				zap.Int64("prev_exec_count", state.PrevExecutionCount))
		}
		if deltaElapsedMs < 0 {
			sic.logger.Warn("Negative delta elapsed time detected - possible stats corruption",
				zap.String("query_id", queryID),
				zap.String("plan_handle", query.PlanHandle.String()),
				zap.Float64("current_total_ms", currentTotalElapsedMs),
				zap.Float64("prev_total_ms", state.PrevTotalElapsedTimeMs),
				zap.Float64("delta_ms", deltaElapsedMs))
		}

		// Reset state - treat as first scrape but PRESERVE FirstSeenTimestamp
		sic.stateCache[cacheKey] = &SimplifiedQueryState{
			PrevExecutionCount:     currentExecCount,
			PrevTotalElapsedTimeMs: currentTotalElapsedMs,
			FirstSeenTimestamp:     state.FirstSeenTimestamp, // FIX #4: Preserve original timestamp
			LastSeenTimestamp:      now,
		}

		return &SimplifiedIntervalMetrics{
			IntervalAvgElapsedTimeMs:   historicalAvgElapsedMs,
			IntervalExecutionCount:     currentExecCount,
			HistoricalAvgElapsedTimeMs: historicalAvgElapsedMs,
			HistoricalAvgCPUTimeMs:     historicalAvgCPUMs,
			HistoricalExecutionCount:   currentExecCount,
			IsFirstScrape:              true, // Treat as first scrape after reset
			HasNewExecutions:           true,
			TimeSinceLastExecSec:       timeSinceLastExec,
		}
	}

	// NORMAL CASE: Calculate interval average (ONLY for elapsed time, no CPU)
	// Fix #2: Use milliseconds directly (no μs conversion, no precision loss)
	intervalAvgElapsedMs := deltaElapsedMs / float64(deltaExecCount)

	// Warn if we have executions but zero elapsed time (possible data corruption or sub-microsecond queries)
	if deltaElapsedMs == 0 && deltaExecCount > 0 {
		sic.logger.Warn("Zero elapsed time with non-zero executions - possible data issue or extremely fast query",
			zap.String("query_id", queryID),
			zap.String("plan_handle", query.PlanHandle.String()),
			zap.Int64("delta_exec_count", deltaExecCount),
			zap.Float64("delta_elapsed_ms", deltaElapsedMs),
			zap.Float64("current_total_ms", currentTotalElapsedMs),
			zap.Float64("prev_total_ms", state.PrevTotalElapsedTimeMs))
	}

	sic.logger.Debug("Delta calculation for query",
		zap.String("query_id", queryID),
		zap.String("plan_handle", query.PlanHandle.String()),
		zap.Int64("delta_exec_count", deltaExecCount),
		zap.Float64("delta_elapsed_ms", deltaElapsedMs),
		zap.Float64("interval_avg_elapsed_ms", intervalAvgElapsedMs),
		zap.Float64("historical_avg_elapsed_ms", historicalAvgElapsedMs))

	// Update state for next scrape
	// IMPORTANT: Always update, regardless of threshold
	// This ensures we can continue delta calculation even if query goes fast→slow→fast
	state.PrevExecutionCount = currentExecCount
	state.PrevTotalElapsedTimeMs = currentTotalElapsedMs
	state.LastSeenTimestamp = now

	return &SimplifiedIntervalMetrics{
		IntervalAvgElapsedTimeMs:   intervalAvgElapsedMs,
		IntervalExecutionCount:     deltaExecCount,
		HistoricalAvgElapsedTimeMs: historicalAvgElapsedMs,
		HistoricalAvgCPUTimeMs:     historicalAvgCPUMs,
		HistoricalExecutionCount:   currentExecCount,
		IsFirstScrape:              false,
		HasNewExecutions:           true,
		TimeSinceLastExecSec:       timeSinceLastExec, // Fix #3: Use DB timestamp, not scrape interval
	}
}

// CleanupStaleEntries removes entries based on TTL (inactivity) ONLY
// NOT based on threshold - we need to preserve state for delta calculation
func (sic *SimplifiedIntervalCalculator) CleanupStaleEntries(now time.Time) {
	sic.stateCacheMutex.Lock()
	defer sic.stateCacheMutex.Unlock()

	// Only run cleanup periodically
	if now.Sub(sic.lastCacheCleanup) < 5*time.Minute {
		return
	}

	sic.lastCacheCleanup = now
	removedCount := 0

	for queryID, state := range sic.stateCache {
		age := now.Sub(state.LastSeenTimestamp)
		if age > sic.cacheTTL {
			delete(sic.stateCache, queryID)
			removedCount++
		}
	}

	if removedCount > 0 {
		sic.logger.Info("Cleaned up stale queries from cache",
			zap.Int("removed_count", removedCount),
			zap.Int("remaining_count", len(sic.stateCache)),
			zap.Duration("cache_ttl", sic.cacheTTL))
	}
}

// GetCacheStats returns cache statistics
func (sic *SimplifiedIntervalCalculator) GetCacheStats() map[string]interface{} {
	sic.stateCacheMutex.RLock()
	defer sic.stateCacheMutex.RUnlock()

	return map[string]interface{}{
		"total_queries_tracked": len(sic.stateCache),
		"cache_ttl_minutes":     sic.cacheTTL.Minutes(),
	}
}

// Reset clears all state (useful for testing)
func (sic *SimplifiedIntervalCalculator) Reset() {
	sic.stateCacheMutex.Lock()
	defer sic.stateCacheMutex.Unlock()

	sic.stateCache = make(map[string]*SimplifiedQueryState)
	sic.logger.Info("Simplified interval calculator cache reset")
}

// Helper functions

func getInt64FromPtr(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func getFloat64FromPtr(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}
