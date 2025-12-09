// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package scrapers provides the query performance smoothing algorithm
// This file implements exponential smoothing for slow query metrics to prevent
// metric discontinuities and provide stable monitoring over time.
//
// Smoothing Algorithm:
//
// The smoothing algorithm prevents "spiky" or "noisy" metrics by:
// 1. Tracking query performance history across multiple collection intervals
// 2. Applying exponential smoothing to metric values
// 3. Implementing hysteresis to avoid sudden appearance/disappearance of queries
// 4. Natural decay for queries that are no longer slow
//
// Key Features:
// - Exponential Weighted Moving Average (EWMA) for metric smoothing
// - Configurable smoothing factor (0.0 to 1.0, where higher = more responsive)
// - Decay mechanism for queries no longer appearing in results
// - Memory-efficient with automatic cleanup of stale entries
//
// Usage:
//
//	smoother := NewSlowQuerySmoother(logger, smoothingFactor, decayThreshold, maxAge)
//	smoothedQueries := smoother.Smooth(currentQueries)
//
// Parameters:
// - smoothingFactor: Weight for new observations (0.0-1.0). Higher = more reactive.
//   - 0.3: Smooth, good for stable trending
//   - 0.5: Balanced between responsiveness and stability
//   - 0.7: Responsive, quick to react to changes
//
// - decayThreshold: Number of consecutive misses before removing query from history
// - maxAge: Maximum age of history entries before forced cleanup
//
// Example:
//
//	// Create smoother with 30% weight for new data, 3-cycle decay, 5-minute max age
//	smoother := NewSlowQuerySmoother(logger, 0.3, 3, 5*time.Minute)
//
//	// In each collection cycle:
//	rawQueries := fetchSlowQueriesFromDB()
//	smoothedQueries := smoother.Smooth(rawQueries)
//	emitMetrics(smoothedQueries)
package scrapers

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
)

// SlowQuerySmoother implements exponential smoothing for slow query metrics
// to prevent metric discontinuities and noise
type SlowQuerySmoother struct {
	logger          *zap.Logger
	history         map[string]*QueryHistory
	mutex           sync.RWMutex
	smoothingFactor float64       // Weight for new observations (0.0-1.0)
	decayThreshold  int           // Consecutive misses before removal
	maxAge          time.Duration // Maximum age before forced cleanup
}

// QueryHistory tracks the historical performance of a specific query
type QueryHistory struct {
	QueryID             string
	LastSeen            time.Time
	FirstSeen           time.Time
	ConsecutiveMisses   int // Number of collection cycles where query didn't appear
	SmoothedMetrics     *models.SlowQuery
	RawMetricsCount     int64 // Number of raw observations smoothed
	LastUpdateTimestamp time.Time
}

// NewSlowQuerySmoother creates a new slow query smoother with specified parameters
//
// Parameters:
// - logger: Logger instance for debugging
// - smoothingFactor: Weight for new data (0.0-1.0). Default: 0.3
//   - 0.0 = no smoothing (use only historical data)
//   - 1.0 = no smoothing (use only current data)
//   - 0.3 = recommended default (30% new, 70% historical)
//
// - decayThreshold: Consecutive misses before removal. Default: 3
// - maxAge: Maximum history age. Default: 5 minutes
func NewSlowQuerySmoother(logger *zap.Logger, smoothingFactor float64, decayThreshold int, maxAge time.Duration) *SlowQuerySmoother {
	// Validate and set defaults
	if smoothingFactor < 0.0 || smoothingFactor > 1.0 {
		logger.Warn("Invalid smoothing factor, using default 0.3", zap.Float64("provided", smoothingFactor))
		smoothingFactor = 0.3
	}
	if decayThreshold <= 0 {
		logger.Warn("Invalid decay threshold, using default 3", zap.Int("provided", decayThreshold))
		decayThreshold = 3
	}
	if maxAge <= 0 {
		logger.Warn("Invalid max age, using default 5 minutes", zap.Duration("provided", maxAge))
		maxAge = 5 * time.Minute
	}

	return &SlowQuerySmoother{
		logger:          logger,
		history:         make(map[string]*QueryHistory),
		smoothingFactor: smoothingFactor,
		decayThreshold:  decayThreshold,
		maxAge:          maxAge,
	}
}

// Smooth applies exponential smoothing to the provided queries
// Returns smoothed query list with continuity across collection intervals
func (s *SlowQuerySmoother) Smooth(rawQueries []models.SlowQuery) []models.SlowQuery {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	seenQueryIDs := make(map[string]bool)

	var smoothedQueries []models.SlowQuery

	// Process each raw query from current collection
	for _, rawQuery := range rawQueries {
		if rawQuery.QueryID == nil || rawQuery.QueryID.IsEmpty() {
			s.logger.Debug("Skipping query with nil or empty QueryID in smoother")
			continue
		}

		queryID := rawQuery.QueryID.String()
		seenQueryIDs[queryID] = true

		history, exists := s.history[queryID]
		if !exists {
			// New query - add to history with initial values
			history = &QueryHistory{
				QueryID:             queryID,
				FirstSeen:           now,
				LastSeen:            now,
				ConsecutiveMisses:   0,
				SmoothedMetrics:     copySlowQuery(&rawQuery),
				RawMetricsCount:     1,
				LastUpdateTimestamp: now,
			}
			s.history[queryID] = history

			s.logger.Debug("New query added to smoothing history",
				zap.String("query_id", queryID),
				zap.Float64("avg_elapsed_time_ms", getFloat64Value(rawQuery.AvgElapsedTimeMS)))
		} else {
			// Existing query - apply exponential smoothing
			history.LastSeen = now
			history.ConsecutiveMisses = 0
			history.RawMetricsCount++
			history.LastUpdateTimestamp = now

			// Apply EWMA: smoothed = α * current + (1 - α) * previous
			history.SmoothedMetrics = s.applyExponentialSmoothing(history.SmoothedMetrics, &rawQuery)

			s.logger.Debug("Query updated with smoothing",
				zap.String("query_id", queryID),
				zap.Float64("raw_avg_elapsed_ms", getFloat64Value(rawQuery.AvgElapsedTimeMS)),
				zap.Float64("smoothed_avg_elapsed_ms", getFloat64Value(history.SmoothedMetrics.AvgElapsedTimeMS)),
				zap.Int64("observation_count", history.RawMetricsCount))
		}

		// Add smoothed query to output
		smoothedQueries = append(smoothedQueries, *history.SmoothedMetrics)
	}

	// Process queries in history that weren't seen in current collection
	// Increment their consecutive misses and potentially remove them
	for queryID, history := range s.history {
		if !seenQueryIDs[queryID] {
			history.ConsecutiveMisses++

			// Check if query should be removed (decay threshold reached or too old)
			age := now.Sub(history.FirstSeen)
			shouldRemove := history.ConsecutiveMisses >= s.decayThreshold || age > s.maxAge

			if shouldRemove {
				delete(s.history, queryID)
				s.logger.Debug("Query removed from smoothing history",
					zap.String("query_id", queryID),
					zap.Int("consecutive_misses", history.ConsecutiveMisses),
					zap.Duration("age", age),
					zap.Bool("decay_threshold_reached", history.ConsecutiveMisses >= s.decayThreshold),
					zap.Bool("max_age_exceeded", age > s.maxAge))
			} else {
				// Query is decaying but still within threshold
				// Continue to emit it with last known smoothed values
				smoothedQueries = append(smoothedQueries, *history.SmoothedMetrics)

				s.logger.Debug("Query in decay phase, still emitting",
					zap.String("query_id", queryID),
					zap.Int("consecutive_misses", history.ConsecutiveMisses),
					zap.Int("decay_threshold", s.decayThreshold))
			}
		}
	}

	s.logger.Debug("Smoothing cycle completed",
		zap.Int("raw_query_count", len(rawQueries)),
		zap.Int("smoothed_query_count", len(smoothedQueries)),
		zap.Int("history_size", len(s.history)))

	return smoothedQueries
}

// applyExponentialSmoothing applies EWMA to smooth metric values
// Formula: smoothed = α * current + (1 - α) * previous
// where α is the smoothing factor
func (s *SlowQuerySmoother) applyExponentialSmoothing(previous *models.SlowQuery, current *models.SlowQuery) *models.SlowQuery {
	α := s.smoothingFactor
	β := 1.0 - α

	smoothed := copySlowQuery(current) // Start with current query structure

	// Apply EWMA to each numeric metric
	if current.AvgCPUTimeMS != nil && previous.AvgCPUTimeMS != nil {
		smoothed.AvgCPUTimeMS = smootherFloatPtr(α*(*current.AvgCPUTimeMS) + β*(*previous.AvgCPUTimeMS))
	}

	if current.AvgElapsedTimeMS != nil && previous.AvgElapsedTimeMS != nil {
		smoothed.AvgElapsedTimeMS = smootherFloatPtr(α*(*current.AvgElapsedTimeMS) + β*(*previous.AvgElapsedTimeMS))
	}

	if current.AvgDiskReads != nil && previous.AvgDiskReads != nil {
		smoothed.AvgDiskReads = smootherFloatPtr(α*(*current.AvgDiskReads) + β*(*previous.AvgDiskReads))
	}

	if current.AvgDiskWrites != nil && previous.AvgDiskWrites != nil {
		smoothed.AvgDiskWrites = smootherFloatPtr(α*(*current.AvgDiskWrites) + β*(*previous.AvgDiskWrites))
	}

	if current.AvgRowsProcessed != nil && previous.AvgRowsProcessed != nil {
		smoothed.AvgRowsProcessed = smootherFloatPtr(α*(*current.AvgRowsProcessed) + β*(*previous.AvgRowsProcessed))
	}

	if current.AvgLockWaitTimeMs != nil && previous.AvgLockWaitTimeMs != nil {
		smoothed.AvgLockWaitTimeMs = smootherFloatPtr(α*(*current.AvgLockWaitTimeMs) + β*(*previous.AvgLockWaitTimeMs))
	}

	// RCA Enhancement Fields
	if current.MinElapsedTimeMs != nil && previous.MinElapsedTimeMs != nil {
		smoothed.MinElapsedTimeMs = smootherFloatPtr(α*(*current.MinElapsedTimeMs) + β*(*previous.MinElapsedTimeMs))
	}

	if current.MaxElapsedTimeMs != nil && previous.MaxElapsedTimeMs != nil {
		smoothed.MaxElapsedTimeMs = smootherFloatPtr(α*(*current.MaxElapsedTimeMs) + β*(*previous.MaxElapsedTimeMs))
	}

	if current.LastElapsedTimeMs != nil && previous.LastElapsedTimeMs != nil {
		smoothed.LastElapsedTimeMs = smootherFloatPtr(α*(*current.LastElapsedTimeMs) + β*(*previous.LastElapsedTimeMs))
	}

	if current.LastGrantKB != nil && previous.LastGrantKB != nil {
		smoothed.LastGrantKB = smootherFloatPtr(α*(*current.LastGrantKB) + β*(*previous.LastGrantKB))
	}

	if current.LastUsedGrantKB != nil && previous.LastUsedGrantKB != nil {
		smoothed.LastUsedGrantKB = smootherFloatPtr(α*(*current.LastUsedGrantKB) + β*(*previous.LastUsedGrantKB))
	}

	if current.LastSpills != nil && previous.LastSpills != nil {
		smoothed.LastSpills = smootherFloatPtr(α*(*current.LastSpills) + β*(*previous.LastSpills))
	}

	if current.MaxSpills != nil && previous.MaxSpills != nil {
		smoothed.MaxSpills = smootherFloatPtr(α*(*current.MaxSpills) + β*(*previous.MaxSpills))
	}

	if current.LastDOP != nil && previous.LastDOP != nil {
		smoothed.LastDOP = smootherFloatPtr(α*(*current.LastDOP) + β*(*previous.LastDOP))
	}

	// For execution count, use simple addition (cumulative)
	if current.ExecutionCount != nil && previous.ExecutionCount != nil {
		smoothed.ExecutionCount = smootherInt64Ptr(*current.ExecutionCount + *previous.ExecutionCount)
	}

	return smoothed
}

// GetHistoryStats returns statistics about the smoothing history
func (s *SlowQuerySmoother) GetHistoryStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	totalQueries := len(s.history)
	decayingQueries := 0
	activeQueries := 0

	for _, history := range s.history {
		if history.ConsecutiveMisses > 0 {
			decayingQueries++
		} else {
			activeQueries++
		}
	}

	return map[string]interface{}{
		"total_queries":    totalQueries,
		"active_queries":   activeQueries,
		"decaying_queries": decayingQueries,
		"smoothing_factor": s.smoothingFactor,
		"decay_threshold":  s.decayThreshold,
		"max_age_minutes":  s.maxAge.Minutes(),
	}
}

// Reset clears all smoothing history (useful for testing or resets)
func (s *SlowQuerySmoother) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.history = make(map[string]*QueryHistory)
	s.logger.Info("Slow query smoothing history reset")
}

// Helper functions

func copySlowQuery(src *models.SlowQuery) *models.SlowQuery {
	if src == nil {
		return nil
	}

	dst := &models.SlowQuery{}

	// Copy all fields
	if src.QueryID != nil {
		dst.QueryID = src.QueryID
	}
	if src.PlanHandle != nil {
		dst.PlanHandle = src.PlanHandle
	}
	if src.QueryText != nil {
		dst.QueryText = smootherStringPtr(*src.QueryText)
	}
	if src.DatabaseName != nil {
		dst.DatabaseName = smootherStringPtr(*src.DatabaseName)
	}
	if src.SchemaName != nil {
		dst.SchemaName = smootherStringPtr(*src.SchemaName)
	}
	if src.LastExecutionTimestamp != nil {
		dst.LastExecutionTimestamp = smootherStringPtr(*src.LastExecutionTimestamp)
	}
	if src.ExecutionCount != nil {
		dst.ExecutionCount = smootherInt64Ptr(*src.ExecutionCount)
	}
	if src.AvgCPUTimeMS != nil {
		dst.AvgCPUTimeMS = smootherFloatPtr(*src.AvgCPUTimeMS)
	}
	if src.AvgElapsedTimeMS != nil {
		dst.AvgElapsedTimeMS = smootherFloatPtr(*src.AvgElapsedTimeMS)
	}
	if src.AvgDiskReads != nil {
		dst.AvgDiskReads = smootherFloatPtr(*src.AvgDiskReads)
	}
	if src.AvgDiskWrites != nil {
		dst.AvgDiskWrites = smootherFloatPtr(*src.AvgDiskWrites)
	}
	if src.AvgRowsProcessed != nil {
		dst.AvgRowsProcessed = smootherFloatPtr(*src.AvgRowsProcessed)
	}
	if src.AvgLockWaitTimeMs != nil {
		dst.AvgLockWaitTimeMs = smootherFloatPtr(*src.AvgLockWaitTimeMs)
	}
	if src.StatementType != nil {
		dst.StatementType = smootherStringPtr(*src.StatementType)
	}
	if src.CollectionTimestamp != nil {
		dst.CollectionTimestamp = smootherStringPtr(*src.CollectionTimestamp)
	}

	// RCA Enhancement Fields
	if src.MinElapsedTimeMs != nil {
		dst.MinElapsedTimeMs = smootherFloatPtr(*src.MinElapsedTimeMs)
	}
	if src.MaxElapsedTimeMs != nil {
		dst.MaxElapsedTimeMs = smootherFloatPtr(*src.MaxElapsedTimeMs)
	}
	if src.LastElapsedTimeMs != nil {
		dst.LastElapsedTimeMs = smootherFloatPtr(*src.LastElapsedTimeMs)
	}
	if src.LastGrantKB != nil {
		dst.LastGrantKB = smootherFloatPtr(*src.LastGrantKB)
	}
	if src.LastUsedGrantKB != nil {
		dst.LastUsedGrantKB = smootherFloatPtr(*src.LastUsedGrantKB)
	}
	if src.LastSpills != nil {
		dst.LastSpills = smootherFloatPtr(*src.LastSpills)
	}
	if src.MaxSpills != nil {
		dst.MaxSpills = smootherFloatPtr(*src.MaxSpills)
	}
	if src.LastDOP != nil {
		dst.LastDOP = smootherFloatPtr(*src.LastDOP)
	}

	return dst
}

func smootherFloatPtr(v float64) *float64 {
	return &v
}

func smootherInt64Ptr(v int64) *int64 {
	return &v
}

func smootherStringPtr(v string) *string {
	return &v
}

func getFloat64Value(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}
