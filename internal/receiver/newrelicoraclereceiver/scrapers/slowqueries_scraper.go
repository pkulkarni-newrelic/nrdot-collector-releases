package scrapers

import (
	"context"
	"sort"
	"time"

	commonutils "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/common-utils"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

// SlowQueriesScraper contains the scraper for slow queries metrics
type SlowQueriesScraper struct {
	client                               client.OracleClient
	mb                                   *metadata.MetricsBuilder
	logger                               *zap.Logger
	instanceName                         string
	metricsBuilderConfig                 metadata.MetricsBuilderConfig
	queryMonitoringResponseTimeThreshold int
	queryMonitoringCountThreshold        int
	queryMonitoringIntervalSeconds       int                       // Time window for fetching queries (in seconds)
	intervalCalculator                   *OracleIntervalCalculator // Delta-based interval calculator
}

func NewSlowQueriesScraper(oracleClient client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, metricsBuilderConfig metadata.MetricsBuilderConfig, responseTimeThreshold, countThreshold, intervalSeconds int, enableIntervalCalculator bool, intervalCalculatorCacheTTLMinutes int) *SlowQueriesScraper {
	var intervalCalc *OracleIntervalCalculator

	// Initialize interval-based calculator if enabled
	if enableIntervalCalculator {
		cacheTTL := time.Duration(intervalCalculatorCacheTTLMinutes) * time.Minute
		intervalCalc = NewOracleIntervalCalculator(logger, cacheTTL)
		logger.Info("Oracle interval-based delta calculator enabled",
			zap.Int("cache_ttl_minutes", intervalCalculatorCacheTTLMinutes))
	} else {
		logger.Info("Oracle interval-based delta calculator disabled - using cumulative averages")
	}

	return &SlowQueriesScraper{
		client:                               oracleClient,
		mb:                                   mb,
		logger:                               logger,
		instanceName:                         instanceName,
		metricsBuilderConfig:                 metricsBuilderConfig,
		queryMonitoringResponseTimeThreshold: responseTimeThreshold,
		queryMonitoringCountThreshold:        countThreshold,
		queryMonitoringIntervalSeconds:       intervalSeconds,
		intervalCalculator:                   intervalCalc,
	}
}

func (s *SlowQueriesScraper) ScrapeSlowQueries(ctx context.Context) ([]string, []error) {
	var scrapeErrors []error
	var queryIDs []string

	// Fetch queries from the database with time window filter
	// Note: intervalSeconds determines the time window (e.g., 60 = last 60 seconds)
	//       This is critical for delta calculation to work correctly
	// The value comes from config and is typically set to collection_interval
	slowQueries, err := s.client.QuerySlowQueries(ctx, s.queryMonitoringIntervalSeconds, s.queryMonitoringResponseTimeThreshold, s.queryMonitoringCountThreshold)
	if err != nil {
		return nil, []error{err}
	}

	s.logger.Debug("Raw slow query metrics fetched",
		zap.Int("raw_result_count", len(slowQueries)),
		zap.Int("interval_seconds", s.queryMonitoringIntervalSeconds),
		zap.Bool("interval_calc_enabled", s.intervalCalculator != nil))

	// Apply interval-based delta calculation if enabled
	var queriesToProcess []models.SlowQuery
	if s.intervalCalculator != nil {
		now := time.Now()

		// Pre-allocate slice capacity to avoid reallocations
		queriesToProcess = make([]models.SlowQuery, 0, len(slowQueries))

		// Calculate interval metrics for each query
		for _, slowQuery := range slowQueries {
			metrics := s.intervalCalculator.CalculateMetrics(&slowQuery, now)

			if metrics == nil {
				s.logger.Debug("Skipping query with nil metrics")
				continue
			}

			// Skip queries with no new executions
			if !metrics.HasNewExecutions {
				s.logger.Debug("Skipping query with no new executions",
					zap.String("query_id", slowQuery.GetQueryID()),
					zap.Float64("time_since_last_exec_sec", metrics.TimeSinceLastExecSec))
				continue
			}

			// Apply interval-based threshold filtering
			// First scrape: interval = historical (no baseline), filter still applies
			// Subsequent scrapes: interval = delta, filter on delta performance
			if metrics.IntervalAvgElapsedTimeMs < float64(s.queryMonitoringResponseTimeThreshold) {
				s.logger.Debug("Skipping query below interval threshold",
					zap.String("query_id", slowQuery.GetQueryID()),
					zap.Float64("interval_avg_ms", metrics.IntervalAvgElapsedTimeMs),
					zap.Float64("historical_avg_ms", metrics.HistoricalAvgElapsedTimeMs),
					zap.Int("threshold_ms", s.queryMonitoringResponseTimeThreshold),
					zap.Bool("is_first_scrape", metrics.IsFirstScrape))
				continue
			}

			// Store interval metrics in new fields (preserve historical values)
			// Historical values remain unchanged from DB
			slowQuery.IntervalAvgElapsedTimeMS = &metrics.IntervalAvgElapsedTimeMs
			slowQuery.IntervalExecutionCount = &metrics.IntervalExecutionCount

			queriesToProcess = append(queriesToProcess, slowQuery)
		}

		// Cleanup stale entries periodically (TTL-based only)
		s.intervalCalculator.CleanupStaleEntries(now)

		s.logger.Debug("Interval-based delta calculation applied",
			zap.Int("raw_count", len(slowQueries)),
			zap.Int("processed_count", len(queriesToProcess)))

		// Log calculator statistics
		stats := s.intervalCalculator.GetCacheStats()
		s.logger.Debug("Interval calculator statistics", zap.Any("stats", stats))

		// Apply Go-level sorting and top-N selection AFTER delta calculation and filtering
		// This ensures we get the slowest queries based on interval (delta) averages, not historical
		if len(queriesToProcess) > 0 {
			// Sort by interval average elapsed time (delta) - slowest first
			sort.Slice(queriesToProcess, func(i, j int) bool {
				// Handle nil cases - queries without interval metrics go to the end
				if queriesToProcess[i].IntervalAvgElapsedTimeMS == nil {
					return false
				}
				if queriesToProcess[j].IntervalAvgElapsedTimeMS == nil {
					return true
				}
				// Sort descending (highest delta average first)
				return *queriesToProcess[i].IntervalAvgElapsedTimeMS > *queriesToProcess[j].IntervalAvgElapsedTimeMS
			})

			s.logger.Debug("Sorted queries by interval average elapsed time (delta)",
				zap.Int("total_count", len(queriesToProcess)))

			// Take top N queries after sorting
			if len(queriesToProcess) > s.queryMonitoringCountThreshold {
				s.logger.Debug("Applying top N selection",
					zap.Int("before_count", len(queriesToProcess)),
					zap.Int("top_n", s.queryMonitoringCountThreshold))
				queriesToProcess = queriesToProcess[:s.queryMonitoringCountThreshold]
			}
		}
	} else {
		// No interval calculator - use raw results
		queriesToProcess = slowQueries
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, slowQuery := range queriesToProcess {
		if !slowQuery.IsValidForMetrics() {
			continue
		}

		collectionTimestamp := slowQuery.GetCollectionTimestamp()
		cdbName := slowQuery.GetCDBName()
		dbName := slowQuery.GetDatabaseName()
		qID := slowQuery.GetQueryID()
		qText := commonutils.AnonymizeAndNormalize(slowQuery.GetQueryText())
		userName := slowQuery.GetUserName()
		schName := slowQuery.GetSchemaName()
		lastActiveTime := slowQuery.GetLastActiveTime()

		s.recordMetrics(now, &slowQuery, collectionTimestamp, cdbName, dbName, qID, qText, userName, schName, lastActiveTime)

		if slowQuery.QueryID.Valid {
			queryIDs = append(queryIDs, slowQuery.QueryID.String)
		}
	}

	s.logger.Debug("Slow queries scrape completed",
		zap.Int("rows", len(slowQueries)),
		zap.Int("processed_rows", len(queriesToProcess)),
		zap.Int("query_ids", len(queryIDs)),
		zap.Int("errors", len(scrapeErrors)))

	return queryIDs, scrapeErrors
}

func (s *SlowQueriesScraper) recordMetrics(now pcommon.Timestamp, slowQuery *models.SlowQuery, collectionTimestamp, cdbName, dbName, qID, qText, userName, schName, lastActiveTime string) {
	if slowQuery.ExecutionCount.Valid {
		s.mb.RecordNewrelicoracledbSlowQueriesExecutionCountDataPoint(
			now,
			float64(slowQuery.ExecutionCount.Int64),
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	if slowQuery.AvgCPUTimeMs.Valid {
		s.mb.RecordNewrelicoracledbSlowQueriesAvgCPUTimeDataPoint(
			now,
			slowQuery.AvgCPUTimeMs.Float64,
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	if slowQuery.AvgDiskReads.Valid {
		s.mb.RecordNewrelicoracledbSlowQueriesAvgDiskReadsDataPoint(
			now,
			slowQuery.AvgDiskReads.Float64,
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	if slowQuery.AvgDiskWrites.Valid {
		s.mb.RecordNewrelicoracledbSlowQueriesAvgDiskWritesDataPoint(
			now,
			slowQuery.AvgDiskWrites.Float64,
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	// Record historical (cumulative) average elapsed time
	s.mb.RecordNewrelicoracledbSlowQueriesAvgElapsedTimeDataPoint(
		now,
		slowQuery.AvgElapsedTimeMs.Float64,
		collectionTimestamp,
		dbName,
		qID,
		userName,
	)

	// Record interval-based average elapsed time if available (delta metric)
	if slowQuery.IntervalAvgElapsedTimeMS != nil {
		s.mb.RecordNewrelicoracledbSlowQueriesIntervalAvgElapsedTimeDataPoint(
			now,
			*slowQuery.IntervalAvgElapsedTimeMS,
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	// Record interval execution count if available (delta metric)
	if slowQuery.IntervalExecutionCount != nil {
		s.mb.RecordNewrelicoracledbSlowQueriesIntervalExecutionCountDataPoint(
			now,
			float64(*slowQuery.IntervalExecutionCount),
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	if slowQuery.AvgRowsExamined.Valid {
		s.mb.RecordNewrelicoracledbSlowQueriesAvgRowsExaminedDataPoint(
			now,
			slowQuery.AvgRowsExamined.Float64,
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	if slowQuery.AvgLockTimeMs.Valid {
		s.mb.RecordNewrelicoracledbSlowQueriesAvgLockTimeDataPoint(
			now,
			slowQuery.AvgLockTimeMs.Float64,
			collectionTimestamp,
			dbName,
			qID,
			userName,
		)
	}

	s.mb.RecordNewrelicoracledbSlowQueriesQueryDetailsDataPoint(
		now,
		1,
		collectionTimestamp,
		cdbName,
		dbName,
		qID,
		qText,
		schName,
		userName,
		lastActiveTime,
	)
}

// GetSlowQueryIDs returns only the query IDs without emitting metrics
// This is used by the logs scraper to get query IDs for execution plans
// without duplicating the slow query metrics already emitted by the metrics scraper
func (s *SlowQueriesScraper) GetSlowQueryIDs(ctx context.Context) ([]string, []error) {
	var queryIDs []string

	slowQueries, err := s.client.QuerySlowQueries(ctx, s.queryMonitoringIntervalSeconds, s.queryMonitoringResponseTimeThreshold, s.queryMonitoringCountThreshold)
	if err != nil {
		return nil, []error{err}
	}

	for _, slowQuery := range slowQueries {
		if !slowQuery.IsValidForMetrics() {
			continue
		}

		if slowQuery.QueryID.Valid {
			queryIDs = append(queryIDs, slowQuery.QueryID.String)
		}
	}

	s.logger.Debug("Fetched slow query IDs (no metrics emitted)",
		zap.Int("rows", len(slowQueries)),
		zap.Int("query_ids", len(queryIDs)))

	return queryIDs, nil
}
