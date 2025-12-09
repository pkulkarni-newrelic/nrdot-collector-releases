// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"database/sql"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/errors"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

type SessionScraper struct {
	client       client.OracleClient
	mb           *metadata.MetricsBuilder
	logger       *zap.Logger
	instanceName string
	config       metadata.MetricsBuilderConfig
}

func NewSessionScraper(c client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, config metadata.MetricsBuilderConfig) *SessionScraper {
	return &SessionScraper{
		client:       c,
		mb:           mb,
		logger:       logger,
		instanceName: instanceName,
		config:       config,
	}
}

func (s *SessionScraper) ScrapeSessionCount(ctx context.Context) []error {
	var errs []error

	if !s.config.Metrics.NewrelicoracledbSessionsCount.Enabled {
		return errs
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	count, err := s.client.QuerySessionCount(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return errs
		}

		scraperErr := errors.NewQueryError(
			"session_count_query",
			"SessionCountSQL",
			err,
			map[string]interface{}{
				"instance":  s.instanceName,
				"retryable": errors.IsRetryableError(err),
				"permanent": errors.IsPermanentError(err),
			},
		)

		errs = append(errs, scraperErr)
		return errs
	}

	if count != nil {
		s.mb.RecordNewrelicoracledbSessionsCountDataPoint(now, count.Count, s.instanceName)

		s.logger.Debug("Session count scrape completed",
			zap.Int64("count", count.Count))
	}

	return errs
}
