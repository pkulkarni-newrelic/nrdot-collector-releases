package scrapers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	commonutils "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/common-utils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
)

// WaitEventBlockingScraper collects both Oracle wait events and blocking query metrics
type WaitEventBlockingScraper struct {
	client                        client.OracleClient
	mb                            *metadata.MetricsBuilder
	logger                        *zap.Logger
	instanceName                  string
	metricsBuilderConfig          metadata.MetricsBuilderConfig
	queryMonitoringCountThreshold int
}

// NewWaitEventBlockingScraper creates a new combined Wait Events and Blocking Scraper instance
func NewWaitEventBlockingScraper(oracleClient client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, metricsBuilderConfig metadata.MetricsBuilderConfig, countThreshold int) (*WaitEventBlockingScraper, error) {
	if oracleClient == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if mb == nil {
		return nil, fmt.Errorf("metrics builder cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if instanceName == "" {
		return nil, fmt.Errorf("instance name cannot be empty")
	}

	return &WaitEventBlockingScraper{
		client:                        oracleClient,
		mb:                            mb,
		logger:                        logger,
		instanceName:                  instanceName,
		metricsBuilderConfig:          metricsBuilderConfig,
		queryMonitoringCountThreshold: countThreshold,
	}, nil
}

// ScrapeWaitEventsAndBlocking collects both wait events and blocking query metrics in a single query
func (s *WaitEventBlockingScraper) ScrapeWaitEventsAndBlocking(ctx context.Context, slowQuerySQLIDs []string) ([]models.SQLIdentifier, []error) {
	var scrapeErrors []error

	waitEvents, err := s.client.QueryWaitEventsWithBlocking(ctx, s.queryMonitoringCountThreshold, slowQuerySQLIDs)
	if err != nil {
		s.logger.Error("Failed to query wait events with blocking information", zap.Error(err))
		return nil, []error{err}
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	waitEventMetricCount, blockingMetricCount := s.emitWaitEventMetrics(now, waitEvents)

	sqlIdentifiers := s.extractSQLIdentifiers(waitEvents)

	s.logger.Debug("Wait events and blocking scrape completed",
		zap.Int("total_events", len(waitEvents)),
		zap.Int("wait_metrics", waitEventMetricCount),
		zap.Int("blocking_metrics", blockingMetricCount),
		zap.Int("unique_sql_identifiers", len(sqlIdentifiers)),
		zap.Int("slow_query_filter_count", len(slowQuerySQLIDs)),
		zap.Int("errors", len(scrapeErrors)))

	return sqlIdentifiers, scrapeErrors
}

// recordWaitEventMetrics records wait event metrics for a session
func (s *WaitEventBlockingScraper) recordWaitEventMetrics(now pcommon.Timestamp, event *models.WaitEventWithBlocking) {
	if !event.HasValidCurrentWaitSeconds() {
		return
	}

	collectionTimestamp := event.GetCollectionTimestamp().Format("2006-01-02 15:04:05")
	cdbName := event.GetCDBName()
	dbName := event.GetDatabaseName()
	username := event.GetUsername()
	sid := strconv.FormatInt(event.GetSID(), 10)
	serial := event.GetSerial()
	status := event.GetStatus()
	state := event.GetState()
	qID := event.GetQueryID()
	sqlChildNumber := event.GetSQLChildNumber()
	waitCat := event.GetWaitCategory()
	waitEventName := event.GetWaitEventName()
	program := event.GetProgram()
	machine := event.GetMachine()
	waitObjectOwner := event.GetObjectOwner()
	waitObjectName := event.GetObjectNameWaitedOn()
	waitObjectType := event.GetObjectTypeWaitedOn()
	sqlExecStart := event.GetSQLExecStart().Format("2006-01-02 15:04:05")
	sqlExecID := event.GetSQLExecID()
	rowWaitObjID := strconv.FormatInt(event.GetLockedObjectID(), 10)
	rowWaitFileID := strconv.FormatInt(event.GetLockedFileID(), 10)
	rowWaitBlockID := strconv.FormatInt(event.GetLockedBlockID(), 10)

	s.mb.RecordNewrelicoracledbWaitEventsCurrentWaitTimeMsDataPoint(
		now,
		event.GetCurrentWaitMs(),
		collectionTimestamp,
		cdbName,
		dbName,
		username,
		sid,
		serial,
		status,
		state,
		qID,
		sqlChildNumber,
		waitEventName,
		waitCat,
		program,
		machine,
		waitObjectOwner,
		waitObjectName,
		waitObjectType,
		sqlExecStart,
		sqlExecID,
		rowWaitObjID,
		rowWaitFileID,
		rowWaitBlockID,
	)
}

// GetSQLIdentifiers retrieves unique SQL identifiers from wait events without emitting metrics
func (s *WaitEventBlockingScraper) GetSQLIdentifiers(ctx context.Context, slowQuerySQLIDs []string) ([]models.SQLIdentifier, []error) {
	waitEvents, err := s.client.QueryWaitEventsWithBlocking(ctx, s.queryMonitoringCountThreshold, slowQuerySQLIDs)
	if err != nil {
		s.logger.Error("Failed to query wait events for SQL identifiers", zap.Error(err))
		return nil, []error{err}
	}

	sqlIdentifiers := s.extractSQLIdentifiers(waitEvents)

	s.logger.Debug("SQL identifiers collected without emitting metrics",
		zap.Int("total_events", len(waitEvents)),
		zap.Int("unique_sql_identifiers", len(sqlIdentifiers)),
		zap.Int("slow_query_filter_count", len(slowQuerySQLIDs)))

	return sqlIdentifiers, nil
}

// emitWaitEventMetrics emits metrics for wait events and blocking queries
func (s *WaitEventBlockingScraper) emitWaitEventMetrics(
	now pcommon.Timestamp,
	waitEvents []models.WaitEventWithBlocking,
) (int, int) {
	waitEventMetricCount := 0
	blockingMetricCount := 0

	for _, event := range waitEvents {
		if event.IsValidForMetrics() {
			s.recordWaitEventMetrics(now, &event)
			waitEventMetricCount++
		}

		if event.IsBlocked() {
			s.recordBlockingMetrics(now, &event)
			blockingMetricCount++
		}
	}

	return waitEventMetricCount, blockingMetricCount
}

// extractSQLIdentifiers extracts unique SQL identifiers from wait events
func (s *WaitEventBlockingScraper) extractSQLIdentifiers(
	waitEvents []models.WaitEventWithBlocking,
) []models.SQLIdentifier {
	sqlIdentifiersMap := make(map[string]models.SQLIdentifier)

	for _, event := range waitEvents {
		if !event.IsValidForMetrics() {
			continue
		}

		sqlID := event.GetQueryID()
		childNumber := event.GetSQLChildNumber()
		hasValidQueryID := event.HasValidQueryID()
		hasValidChildNumber := event.SQLChildNumber.Valid

		if hasValidQueryID && hasValidChildNumber {
			key := fmt.Sprintf("%s#%d", sqlID, childNumber)

			if _, exists := sqlIdentifiersMap[key]; !exists {
				// Use the collection timestamp from the wait event as the query execution timestamp
				timestamp := event.GetCollectionTimestamp()
				if timestamp.IsZero() {
					timestamp = time.Now()
				}

				sqlIdentifiersMap[key] = models.SQLIdentifier{
					SQLID:       sqlID,
					ChildNumber: childNumber,
					Timestamp:   timestamp,
				}
				s.logger.Debug("Added SQL identifier for execution plan",
					zap.String("sql_id", sqlID),
					zap.Int64("child_number", childNumber),
					zap.Time("timestamp", timestamp))
			}
		}
	}

	sqlIdentifiers := make([]models.SQLIdentifier, 0, len(sqlIdentifiersMap))
	for _, identifier := range sqlIdentifiersMap {
		sqlIdentifiers = append(sqlIdentifiers, identifier)
	}

	return sqlIdentifiers
}

// recordBlockingMetrics records blocking query metrics when a session is blocked
func (s *WaitEventBlockingScraper) recordBlockingMetrics(now pcommon.Timestamp, event *models.WaitEventWithBlocking) {
	blockedWaitMs := event.GetCurrentWaitMs()

	if blockedWaitMs <= 0 {
		return
	}

	collectionTimestamp := event.GetCollectionTimestamp().Format("2006-01-02 15:04:05")
	cdbName := event.GetCDBName()
	dbName := event.GetDatabaseName()
	blockedUser := event.GetUsername()
	queryID := event.GetQueryID()
	sessionID := strconv.FormatInt(event.GetSID(), 10)
	blockedSerial := event.GetSerial()
	state := event.GetState()
	sqlChildNumber := event.GetSQLChildNumber()
	sqlExecID := event.GetSQLExecID()
	sqlExecStart := event.GetSQLExecStart().Format("2006-01-02 15:04:05")

	waitEventName := event.GetWaitEventName()
	waitCat := event.GetWaitCategory()
	waitObjectName := event.GetObjectNameWaitedOn()
	waitObjectOwner := event.GetObjectOwner()
	waitObjectType := event.GetObjectTypeWaitedOn()

	blockingSessionStatus := event.GetBlockingSessionStatus()
	immediateBlockerSID := strconv.FormatInt(event.GetImmediateBlockerSID(), 10)
	finalBlockingSessionStatus := event.GetFinalBlockingSessionStatus()

	finalBlockerSID := strconv.FormatInt(event.GetFinalBlockerSID(), 10)
	finalBlockerSerial := strconv.FormatInt(event.GetFinalBlockerSerial(), 10)
	finalBlockerUser := event.GetFinalBlockerUser()
	finalBlockerQueryID := event.GetFinalBlockerQueryID()
	finalBlockerQueryText := commonutils.AnonymizeAndNormalize(event.GetFinalBlockerQueryText())

	s.mb.RecordNewrelicoracledbBlockingQueriesWaitTimeMsDataPoint(
		now,
		blockedWaitMs,
		collectionTimestamp,
		cdbName,
		dbName,
		blockedUser,
		sessionID,
		blockedSerial,
		state,
		queryID,
		sqlChildNumber,
		sqlExecID,
		sqlExecStart,
		waitEventName,
		waitCat,
		waitObjectName,
		waitObjectOwner,
		waitObjectType,
		blockingSessionStatus,
		immediateBlockerSID,
		finalBlockingSessionStatus,
		finalBlockerUser,
		finalBlockerSID,
		finalBlockerSerial,
		finalBlockerQueryID,
		finalBlockerQueryText,
	)
}
