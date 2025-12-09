// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package scrapers provides the interval-based averaging calculator for slow query metrics
// This file implements delta-based interval calculations to address the problem of
// cumulative averages masking recent query optimizations.
//
// Problem Statement:
// When using sys.dm_exec_query_stats, metrics like avg_elapsed_time are cumulative
// averages since the query plan was cached. If a query is optimized after running
// slow for days, the cumulative average takes a very long time to reflect the improvement.
//
// Solution:
// Track previous scrape values (execution_count, total_elapsed_time) and calculate
// metrics based on deltas between intervals. This gives the average performance in
// the LAST interval (e.g., last 30 seconds) rather than cumulative average.
//
// Algorithm:
// 1. Maintain state cache with previous scrape values per query_hash
// 2. Calculate delta between current and previous scrape
// 3. interval_avg = delta_elapsed_time / delta_execution_count
// 4. Handle edge cases: first observation, no executions, plan cache reset, low samples
//
// Benefits:
// - Optimizations are immediately reflected in metrics (within one scrape interval)
// - Degradations are immediately detected
// - More accurate representation of current query performance
package scrapers

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
)

// QueryState tracks previous scrape data for delta calculation
type QueryState struct {
	// Previous values from DMV
	PrevExecutionCount     int64
	PrevTotalElapsedTimeUs int64 // microseconds from DMV
	PrevTotalCPUTimeUs     int64 // microseconds from DMV

	// Timestamps
	LastSeenTimestamp  time.Time
	FirstSeenTimestamp time.Time
}

// IntervalMetrics holds calculated interval-based metrics
type IntervalMetrics struct {
	// Primary metric - interval-based average
	IntervalAvgElapsedTimeMs float64
	IntervalAvgCPUTimeMs     float64
	IntervalExecutionCount   int64

	// Detection metadata
	DetectionMethod      string // "interval_avg", "last_time", "cumulative_avg", "initial_observation", "hybrid_low_sample", "plan_cache_reset", "stale"
	IsInitialObservation bool
	TimeSinceLastExecSec float64

	// For comparison/debugging - include cumulative average
	CumulativeAvgElapsedTimeMs float64
	CumulativeExecutionCount   int64
}

// IntervalCalculator implements interval-based averaging for slow query detection
type IntervalCalculator struct {
	logger *zap.Logger

	// State cache for tracking previous values
	stateCache      map[string]*QueryState
	stateCacheMutex sync.RWMutex

	// Configuration
	stateCacheTTL      time.Duration // Default: 10 minutes
	minSampleThreshold int64         // Minimum delta executions for pure interval avg (default: 2)
	lastCacheCleanup   time.Time
}

// NewIntervalCalculator creates a new interval-based averaging calculator
func NewIntervalCalculator(logger *zap.Logger, cacheTTL time.Duration, minSampleThreshold int64) *IntervalCalculator {
	if cacheTTL <= 0 {
		logger.Warn("Invalid cache TTL, using default 10 minutes", zap.Duration("provided", cacheTTL))
		cacheTTL = 10 * time.Minute
	}
	if minSampleThreshold <= 0 {
		logger.Warn("Invalid min sample threshold, using default 2", zap.Int64("provided", minSampleThreshold))
		minSampleThreshold = 2
	}

	return &IntervalCalculator{
		logger:             logger,
		stateCache:         make(map[string]*QueryState),
		stateCacheTTL:      cacheTTL,
		minSampleThreshold: minSampleThreshold,
		lastCacheCleanup:   time.Now(),
	}
}

// CalculateIntervalMetrics calculates interval-based metrics for a query
// Implements the 5-scenario decision tree from requirements.md
func (ic *IntervalCalculator) CalculateIntervalMetrics(query *models.SlowQuery, now time.Time) *IntervalMetrics {
	if query == nil || query.QueryID == nil || query.QueryID.IsEmpty() {
		ic.logger.Debug("Skipping query with nil or empty QueryID in interval calculator")
		return nil
	}

	queryID := query.QueryID.String()

	// Get current values from query
	currentExecCount := getInt64ValueFromQuery(query.ExecutionCount)
	currentTotalElapsedUs := int64(getFloat64ValueFromQuery(query.AvgElapsedTimeMS) * float64(currentExecCount) * 1000.0) // Convert ms to μs
	currentTotalCPUUs := int64(getFloat64ValueFromQuery(query.AvgCPUTimeMS) * float64(currentExecCount) * 1000.0)         // Convert ms to μs

	// Last execution time (for staleness check)
	lastExecTime := query.LastExecutionTimestamp

	// Lock for state cache access
	ic.stateCacheMutex.Lock()
	defer ic.stateCacheMutex.Unlock()

	// Check if query exists in state cache
	state, exists := ic.stateCache[queryID]

	// SCENARIO 1: Initial Observation (First Time Seeing Query)
	if !exists {
		ic.logger.Debug("Scenario 1: Initial observation",
			zap.String("query_id", queryID),
			zap.Int64("execution_count", currentExecCount))

		// Create new state
		ic.stateCache[queryID] = &QueryState{
			PrevExecutionCount:     currentExecCount,
			PrevTotalElapsedTimeUs: currentTotalElapsedUs,
			PrevTotalCPUTimeUs:     currentTotalCPUUs,
			FirstSeenTimestamp:     now,
			LastSeenTimestamp:      now,
		}

		// Use last_elapsed_time as proxy (most recent execution)
		lastElapsedMs := getFloat64ValueFromQuery(query.LastElapsedTimeMs)
		cumulativeAvgMs := getFloat64ValueFromQuery(query.AvgElapsedTimeMS)

		return &IntervalMetrics{
			IntervalAvgElapsedTimeMs:   lastElapsedMs,
			IntervalAvgCPUTimeMs:       getFloat64ValueFromQuery(query.AvgCPUTimeMS),
			IntervalExecutionCount:     currentExecCount,
			DetectionMethod:            "initial_observation",
			IsInitialObservation:       true,
			TimeSinceLastExecSec:       0,
			CumulativeAvgElapsedTimeMs: cumulativeAvgMs,
			CumulativeExecutionCount:   currentExecCount,
		}
	}

	// SCENARIO 4: Plan Cache Reset / SQL Server Restart
	// Condition: current_execution_count < prev_execution_count
	if currentExecCount < state.PrevExecutionCount {
		ic.logger.Debug("Scenario 4: Plan cache reset detected",
			zap.String("query_id", queryID),
			zap.Int64("current_exec_count", currentExecCount),
			zap.Int64("prev_exec_count", state.PrevExecutionCount))

		// Reset state - treat as new observation
		ic.stateCache[queryID] = &QueryState{
			PrevExecutionCount:     currentExecCount,
			PrevTotalElapsedTimeUs: currentTotalElapsedUs,
			PrevTotalCPUTimeUs:     currentTotalCPUUs,
			FirstSeenTimestamp:     now, // Reset first seen
			LastSeenTimestamp:      now,
		}

		lastElapsedMs := getFloat64ValueFromQuery(query.LastElapsedTimeMs)
		cumulativeAvgMs := getFloat64ValueFromQuery(query.AvgElapsedTimeMS)

		return &IntervalMetrics{
			IntervalAvgElapsedTimeMs:   lastElapsedMs,
			IntervalAvgCPUTimeMs:       getFloat64ValueFromQuery(query.AvgCPUTimeMS),
			IntervalExecutionCount:     currentExecCount,
			DetectionMethod:            "plan_cache_reset",
			IsInitialObservation:       true, // Treat as initial
			TimeSinceLastExecSec:       0,
			CumulativeAvgElapsedTimeMs: cumulativeAvgMs,
			CumulativeExecutionCount:   currentExecCount,
		}
	}

	// Calculate deltas
	deltaExecCount := currentExecCount - state.PrevExecutionCount
	deltaElapsedUs := currentTotalElapsedUs - state.PrevTotalElapsedTimeUs
	deltaCPUUs := currentTotalCPUUs - state.PrevTotalCPUTimeUs

	// SCENARIO 2: No New Executions in Interval
	// Condition: delta_execution_count == 0
	if deltaExecCount == 0 {
		// Calculate time since last execution
		timeSinceLastExec := 0.0
		if lastExecTime != nil {
			if lastExecTimeParsed, err := time.Parse(time.RFC3339, *lastExecTime); err == nil {
				timeSinceLastExec = now.Sub(lastExecTimeParsed).Seconds()
			}
		}

		ic.logger.Debug("Scenario 2: No new executions",
			zap.String("query_id", queryID),
			zap.Float64("time_since_last_exec_sec", timeSinceLastExec))

		// Update last seen timestamp
		state.LastSeenTimestamp = now

		// Return stale metric (caller can decide whether to emit)
		return &IntervalMetrics{
			IntervalAvgElapsedTimeMs:   0,
			IntervalAvgCPUTimeMs:       0,
			IntervalExecutionCount:     0,
			DetectionMethod:            "stale",
			IsInitialObservation:       false,
			TimeSinceLastExecSec:       timeSinceLastExec,
			CumulativeAvgElapsedTimeMs: getFloat64ValueFromQuery(query.AvgElapsedTimeMS),
			CumulativeExecutionCount:   currentExecCount,
		}
	}

	// SCENARIO 3: Very Few Executions (1-2 executions)
	// Condition: 0 < delta_execution_count < min_threshold
	if deltaExecCount > 0 && deltaExecCount < ic.minSampleThreshold {
		ic.logger.Debug("Scenario 3: Few executions - hybrid approach",
			zap.String("query_id", queryID),
			zap.Int64("delta_exec_count", deltaExecCount))

		// Calculate interval average
		intervalAvgElapsedMs := 0.0
		if deltaExecCount > 0 {
			intervalAvgElapsedMs = float64(deltaElapsedUs) / float64(deltaExecCount) / 1000.0 // Convert μs to ms
		}

		intervalAvgCPUMs := 0.0
		if deltaExecCount > 0 {
			intervalAvgCPUMs = float64(deltaCPUUs) / float64(deltaExecCount) / 1000.0 // Convert μs to ms
		}

		// Use hybrid approach: weighted average of interval_avg and last_elapsed_time
		lastElapsedMs := getFloat64ValueFromQuery(query.LastElapsedTimeMs)
		hybridWeight := 0.6 // 60% interval avg, 40% last elapsed
		hybridAvgElapsedMs := hybridWeight*intervalAvgElapsedMs + (1-hybridWeight)*lastElapsedMs

		// Update state
		state.PrevExecutionCount = currentExecCount
		state.PrevTotalElapsedTimeUs = currentTotalElapsedUs
		state.PrevTotalCPUTimeUs = currentTotalCPUUs
		state.LastSeenTimestamp = now

		return &IntervalMetrics{
			IntervalAvgElapsedTimeMs:   hybridAvgElapsedMs,
			IntervalAvgCPUTimeMs:       intervalAvgCPUMs,
			IntervalExecutionCount:     deltaExecCount,
			DetectionMethod:            "hybrid_low_sample",
			IsInitialObservation:       false,
			TimeSinceLastExecSec:       now.Sub(state.LastSeenTimestamp).Seconds(),
			CumulativeAvgElapsedTimeMs: getFloat64ValueFromQuery(query.AvgElapsedTimeMS),
			CumulativeExecutionCount:   currentExecCount,
		}
	}

	// SCENARIO 5: Normal Flow (Sufficient Executions)
	// Condition: delta_execution_count >= min_threshold
	ic.logger.Debug("Scenario 5: Normal flow - pure interval average",
		zap.String("query_id", queryID),
		zap.Int64("delta_exec_count", deltaExecCount))

	// Calculate pure interval average
	intervalAvgElapsedMs := float64(deltaElapsedUs) / float64(deltaExecCount) / 1000.0 // Convert μs to ms
	intervalAvgCPUMs := float64(deltaCPUUs) / float64(deltaExecCount) / 1000.0         // Convert μs to ms

	// Update state
	state.PrevExecutionCount = currentExecCount
	state.PrevTotalElapsedTimeUs = currentTotalElapsedUs
	state.PrevTotalCPUTimeUs = currentTotalCPUUs
	state.LastSeenTimestamp = now

	return &IntervalMetrics{
		IntervalAvgElapsedTimeMs:   intervalAvgElapsedMs,
		IntervalAvgCPUTimeMs:       intervalAvgCPUMs,
		IntervalExecutionCount:     deltaExecCount,
		DetectionMethod:            "interval_avg",
		IsInitialObservation:       false,
		TimeSinceLastExecSec:       now.Sub(state.LastSeenTimestamp).Seconds(),
		CumulativeAvgElapsedTimeMs: getFloat64ValueFromQuery(query.AvgElapsedTimeMS),
		CumulativeExecutionCount:   currentExecCount,
	}
}

// CleanupStaleEntries removes entries from state cache that exceed TTL
func (ic *IntervalCalculator) CleanupStaleEntries(now time.Time) {
	ic.stateCacheMutex.Lock()
	defer ic.stateCacheMutex.Unlock()

	// Only run cleanup periodically (e.g., every 5 minutes)
	if now.Sub(ic.lastCacheCleanup) < 5*time.Minute {
		return
	}

	ic.lastCacheCleanup = now
	removedCount := 0

	for queryID, state := range ic.stateCache {
		age := now.Sub(state.LastSeenTimestamp)
		if age > ic.stateCacheTTL {
			delete(ic.stateCache, queryID)
			removedCount++
		}
	}

	if removedCount > 0 {
		ic.logger.Debug("Cleaned up stale state cache entries",
			zap.Int("removed_count", removedCount),
			zap.Int("remaining_count", len(ic.stateCache)))
	}
}

// GetCacheStats returns statistics about the state cache
func (ic *IntervalCalculator) GetCacheStats() map[string]interface{} {
	ic.stateCacheMutex.RLock()
	defer ic.stateCacheMutex.RUnlock()

	return map[string]interface{}{
		"total_queries":        len(ic.stateCache),
		"cache_ttl_minutes":    ic.stateCacheTTL.Minutes(),
		"min_sample_threshold": ic.minSampleThreshold,
	}
}

// Reset clears all state cache (useful for testing)
func (ic *IntervalCalculator) Reset() {
	ic.stateCacheMutex.Lock()
	defer ic.stateCacheMutex.Unlock()

	ic.stateCache = make(map[string]*QueryState)
	ic.logger.Info("Interval calculator state cache reset")
}

// Helper functions

func getInt64ValueFromQuery(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func getFloat64ValueFromQuery(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}
