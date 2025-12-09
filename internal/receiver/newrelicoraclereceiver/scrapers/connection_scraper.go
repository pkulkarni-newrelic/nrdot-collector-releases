// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

// ConnectionScraper contains the scraper for Oracle connection statistics
type ConnectionScraper struct {
	client               client.OracleClient
	mb                   *metadata.MetricsBuilder
	logger               *zap.Logger
	instanceName         string
	metricsBuilderConfig metadata.MetricsBuilderConfig
}

// NewConnectionScraper creates a new Connection Scraper instance
func NewConnectionScraper(
	oracleClient client.OracleClient,
	mb *metadata.MetricsBuilder,
	logger *zap.Logger,
	instanceName string,
	metricsBuilderConfig metadata.MetricsBuilderConfig,
) (*ConnectionScraper, error) {
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

	return &ConnectionScraper{
		client:               oracleClient,
		mb:                   mb,
		logger:               logger,
		instanceName:         instanceName,
		metricsBuilderConfig: metricsBuilderConfig,
	}, nil
}

// ScrapeConnectionMetrics collects Oracle connection statistics
func (s *ConnectionScraper) ScrapeConnectionMetrics(ctx context.Context) []error {
	var scrapeErrors []error
	now := pcommon.NewTimestampFromTime(time.Now())

	scrapeErrors = append(scrapeErrors, s.scrapeCoreConnectionCounts(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeSessionBreakdown(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeLogonStats(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeSessionResourceConsumption(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeWaitEvents(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeBlockingSessions(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeWaitEventSummary(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeConnectionPoolMetrics(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeSessionLimits(ctx, now)...)
	scrapeErrors = append(scrapeErrors, s.scrapeConnectionQuality(ctx, now)...)

	return scrapeErrors
}

// scrapeCoreConnectionCounts scrapes basic connection counts
func (s *ConnectionScraper) scrapeCoreConnectionCounts(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	totalSessions, err := s.client.QueryTotalSessions(ctx)
	if err != nil {
		s.logger.Debug("Failed to query total sessions", zap.Error(err))
		errors = append(errors, err)
	} else {
		s.mb.RecordNewrelicoracledbConnectionTotalSessionsDataPoint(timestamp, float64(totalSessions), s.instanceName)
	}

	activeSessions, err := s.client.QueryActiveSessions(ctx)
	if err != nil {
		s.logger.Debug("Failed to query active sessions", zap.Error(err))
		errors = append(errors, err)
	} else {
		s.mb.RecordNewrelicoracledbConnectionActiveSessionsDataPoint(timestamp, float64(activeSessions), s.instanceName)
	}

	inactiveSessions, err := s.client.QueryInactiveSessions(ctx)
	if err != nil {
		s.logger.Debug("Failed to query inactive sessions", zap.Error(err))
		errors = append(errors, err)
	} else {
		s.mb.RecordNewrelicoracledbConnectionInactiveSessionsDataPoint(timestamp, float64(inactiveSessions), s.instanceName)
	}

	return errors
}

// scrapeSessionBreakdown scrapes session breakdown by status and type
func (s *ConnectionScraper) scrapeSessionBreakdown(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	sessionStatuses, err := s.client.QuerySessionStatus(ctx)
	if err != nil {
		s.logger.Debug("Failed to query session status", zap.Error(err))
		errors = append(errors, err)
	} else {
		for _, status := range sessionStatuses {
			if status.Status.Valid && status.Count.Valid {
				s.mb.RecordNewrelicoracledbConnectionSessionsByStatusDataPoint(
					timestamp,
					float64(status.Count.Int64),
					s.instanceName,
					status.Status.String,
				)
			}
		}
	}

	sessionTypes, err := s.client.QuerySessionTypes(ctx)
	if err != nil {
		s.logger.Debug("Failed to query session type", zap.Error(err))
		errors = append(errors, err)
	} else {
		for _, sessionType := range sessionTypes {
			if sessionType.Type.Valid && sessionType.Count.Valid {
				s.mb.RecordNewrelicoracledbConnectionSessionsByTypeDataPoint(
					timestamp,
					float64(sessionType.Count.Int64),
					s.instanceName,
					sessionType.Type.String,
				)
			}
		}
	}

	return errors
}

// scrapeLogonStats scrapes logon statistics
func (s *ConnectionScraper) scrapeLogonStats(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	logonStats, err := s.client.QueryLogonStats(ctx)
	if err != nil {
		s.logger.Debug("Failed to query logon stats", zap.Error(err))
		return []error{err}
	}

	for _, stat := range logonStats {
		if !stat.Name.Valid || !stat.Value.Valid {
			continue
		}

		switch stat.Name.String {
		case "logons cumulative":
			s.mb.RecordNewrelicoracledbConnectionLogonsCumulativeDataPoint(
				timestamp,
				stat.Value.Float64,
				s.instanceName,
			)
		case "logons current":
			s.mb.RecordNewrelicoracledbConnectionLogonsCurrentDataPoint(
				timestamp,
				stat.Value.Float64,
				s.instanceName,
			)
		}
	}

	return errors
}

// scrapeSessionResourceConsumption scrapes top resource consuming sessions
func (s *ConnectionScraper) scrapeSessionResourceConsumption(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	resources, err := s.client.QuerySessionResources(ctx)
	if err != nil {
		s.logger.Debug("Failed to query session resource consumption", zap.Error(err))
		return []error{err}
	}

	for _, resource := range resources {
		sidStr := s.formatInt64(resource.SID)
		userStr := s.formatString(resource.Username)
		statusStr := s.formatString(resource.Status)
		programStr := s.formatString(resource.Program)

		if resource.CPUUsageSeconds.Valid {
			s.mb.RecordNewrelicoracledbConnectionSessionCPUUsageDataPoint(
				timestamp,
				resource.CPUUsageSeconds.Float64,
				s.instanceName,
				sidStr,
				userStr,
				statusStr,
				programStr,
			)
		}

		if resource.PGAMemoryBytes.Valid {
			s.mb.RecordNewrelicoracledbConnectionSessionPgaMemoryDataPoint(
				timestamp,
				float64(resource.PGAMemoryBytes.Int64),
				s.instanceName,
				sidStr,
				userStr,
				statusStr,
				programStr,
			)
		}

		if resource.LogicalReads.Valid {
			s.mb.RecordNewrelicoracledbConnectionSessionLogicalReadsDataPoint(
				timestamp,
				float64(resource.LogicalReads.Int64),
				s.instanceName,
				sidStr,
				userStr,
				statusStr,
				programStr,
			)
		}

		if resource.LastCallET.Valid {
			s.mb.RecordNewrelicoracledbConnectionSessionIdleTimeDataPoint(
				timestamp,
				float64(resource.LastCallET.Int64),
				s.instanceName,
				sidStr,
				userStr,
				statusStr,
				programStr,
			)
		}
	}

	return errors
}

// scrapeWaitEvents scrapes current wait events
func (s *ConnectionScraper) scrapeWaitEvents(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	waitEvents, err := s.client.QueryCurrentWaitEvents(ctx)
	if err != nil {
		s.logger.Debug("Failed to query wait events", zap.Error(err))
		return []error{err}
	}

	for _, event := range waitEvents {
		if !event.SID.Valid || !event.Event.Valid || !event.SecondsInWait.Valid {
			continue
		}

		s.mb.RecordNewrelicoracledbConnectionWaitEventsDataPoint(
			timestamp,
			float64(event.SecondsInWait.Int64),
			s.instanceName,
			s.formatInt64(event.SID),
			s.formatString(event.Username),
			event.Event.String,
			s.formatString(event.State),
			s.formatString(event.WaitClass),
		)
	}

	return errors
}

// scrapeBlockingSessions scrapes blocking sessions information
func (s *ConnectionScraper) scrapeBlockingSessions(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	blockingSessions, err := s.client.QueryBlockingSessions(ctx)
	if err != nil {
		s.logger.Debug("Failed to query blocking sessions", zap.Error(err))
		return []error{err}
	}

	for _, session := range blockingSessions {
		if !session.SID.Valid || !session.BlockingSession.Valid || !session.SecondsInWait.Valid {
			continue
		}

		s.mb.RecordNewrelicoracledbConnectionBlockingSessionsDataPoint(
			timestamp,
			float64(session.SecondsInWait.Int64),
			s.instanceName,
			s.formatInt64(session.SID),
			s.formatInt64(session.BlockingSession),
			s.formatString(session.Username),
			s.formatString(session.Event),
			s.formatString(session.Program),
		)
	}

	return errors
}

// scrapeWaitEventSummary scrapes wait event summary
func (s *ConnectionScraper) scrapeWaitEventSummary(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	summaries, err := s.client.QueryWaitEventSummary(ctx)
	if err != nil {
		s.logger.Debug("Failed to query wait event summary", zap.Error(err))
		return []error{err}
	}

	for _, summary := range summaries {
		if !summary.Event.Valid || !summary.TotalWaits.Valid || !summary.TimeWaitedMicro.Valid {
			continue
		}

		eventStr := summary.Event.String
		waitClassStr := s.formatString(summary.WaitClass)

		s.mb.RecordNewrelicoracledbConnectionWaitEventTotalWaitsDataPoint(
			timestamp,
			float64(summary.TotalWaits.Int64),
			s.instanceName,
			eventStr,
			waitClassStr,
		)

		s.mb.RecordNewrelicoracledbConnectionWaitEventTimeWaitedDataPoint(
			timestamp,
			float64(summary.TimeWaitedMicro.Int64)/1000.0,
			s.instanceName,
			eventStr,
			waitClassStr,
		)

		if summary.AverageWaitMicro.Valid {
			s.mb.RecordNewrelicoracledbConnectionWaitEventAvgWaitTimeDataPoint(
				timestamp,
				summary.AverageWaitMicro.Float64/1000.0,
				s.instanceName,
				eventStr,
				waitClassStr,
			)
		}
	}

	return errors
}

// scrapeConnectionPoolMetrics scrapes connection pool metrics
func (s *ConnectionScraper) scrapeConnectionPoolMetrics(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	poolMetrics, err := s.client.QueryConnectionPoolMetrics(ctx)
	if err != nil {
		s.logger.Debug("Failed to query connection pool metrics", zap.Error(err))
		return []error{err}
	}

	for _, metric := range poolMetrics {
		if !metric.MetricName.Valid || !metric.Value.Valid {
			continue
		}

		switch metric.MetricName.String {
		case "shared_servers":
			s.mb.RecordNewrelicoracledbConnectionSharedServersDataPoint(
				timestamp,
				float64(metric.Value.Int64),
				s.instanceName,
			)
		case "dispatchers":
			s.mb.RecordNewrelicoracledbConnectionDispatchersDataPoint(
				timestamp,
				float64(metric.Value.Int64),
				s.instanceName,
			)
		case "circuits":
			s.mb.RecordNewrelicoracledbConnectionCircuitsDataPoint(
				timestamp,
				float64(metric.Value.Int64),
				s.instanceName,
			)
		}
	}

	return errors
}

// scrapeSessionLimits scrapes session limits
func (s *ConnectionScraper) scrapeSessionLimits(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	limits, err := s.client.QuerySessionLimits(ctx)
	if err != nil {
		s.logger.Debug("Failed to query session limits", zap.Error(err))
		return []error{err}
	}

	for _, limit := range limits {
		if !limit.ResourceName.Valid || !limit.CurrentUtilization.Valid {
			continue
		}

		resourceStr := limit.ResourceName.String

		s.mb.RecordNewrelicoracledbConnectionResourceCurrentUtilizationDataPoint(
			timestamp,
			float64(limit.CurrentUtilization.Int64),
			s.instanceName,
			resourceStr,
		)

		if limit.MaxUtilization.Valid {
			s.mb.RecordNewrelicoracledbConnectionResourceMaxUtilizationDataPoint(
				timestamp,
				float64(limit.MaxUtilization.Int64),
				s.instanceName,
				resourceStr,
			)
		}

		if limit.LimitValue.Valid && limit.LimitValue.String != "UNLIMITED" {
			if limitVal, err := strconv.ParseInt(limit.LimitValue.String, 10, 64); err == nil {
				s.mb.RecordNewrelicoracledbConnectionResourceLimitDataPoint(
					timestamp,
					float64(limitVal),
					s.instanceName,
					resourceStr,
				)
			}
		}
	}

	return errors
}

// scrapeConnectionQuality scrapes connection quality metrics
func (s *ConnectionScraper) scrapeConnectionQuality(ctx context.Context, timestamp pcommon.Timestamp) []error {
	var errors []error

	qualityMetrics, err := s.client.QueryConnectionQuality(ctx)
	if err != nil {
		s.logger.Debug("Failed to query connection quality", zap.Error(err))
		return []error{err}
	}

	for _, metric := range qualityMetrics {
		if !metric.Name.Valid || !metric.Value.Valid {
			continue
		}

		switch metric.Name.String {
		case "user commits":
			s.mb.RecordNewrelicoracledbConnectionUserCommitsDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "user rollbacks":
			s.mb.RecordNewrelicoracledbConnectionUserRollbacksDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "parse count (total)":
			s.mb.RecordNewrelicoracledbConnectionParseCountTotalDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "parse count (hard)":
			s.mb.RecordNewrelicoracledbConnectionParseCountHardDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "execute count":
			s.mb.RecordNewrelicoracledbConnectionExecuteCountDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "SQL*Net roundtrips to/from client":
			s.mb.RecordNewrelicoracledbConnectionSqlnetRoundtripsDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "bytes sent via SQL*Net to client":
			s.mb.RecordNewrelicoracledbConnectionBytesSentDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		case "bytes received via SQL*Net from client":
			s.mb.RecordNewrelicoracledbConnectionBytesReceivedDataPoint(timestamp, metric.Value.Float64, s.instanceName)
		}
	}

	return errors
}

// formatInt64 converts sql.NullInt64 to string
func (s *ConnectionScraper) formatInt64(val sql.NullInt64) string {
	if val.Valid {
		return strconv.FormatInt(val.Int64, 10)
	}
	return ""
}

// formatString converts sql.NullString to string
func (s *ConnectionScraper) formatString(val sql.NullString) string {
	if val.Valid {
		return val.String
	}
	return ""
}
