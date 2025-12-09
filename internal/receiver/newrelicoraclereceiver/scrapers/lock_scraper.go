// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

// LockScraper collects Oracle lock-related metrics
type LockScraper struct {
	logger       *zap.Logger
	client       client.OracleClient
	mb           *metadata.MetricsBuilder
	instanceName string
}

// NewLockScraper creates a new lock scraper
func NewLockScraper(logger *zap.Logger, client client.OracleClient, mb *metadata.MetricsBuilder, instanceName string) *LockScraper {
	return &LockScraper{
		logger:       logger,
		client:       client,
		mb:           mb,
		instanceName: instanceName,
	}
}

// ScrapeLocks collects lock-related metrics
func (s *LockScraper) ScrapeLocks(ctx context.Context) []error {
	var errors []error
	now := pcommon.NewTimestampFromTime(time.Now())

	// Collect lock counts by type and mode
	if err := s.scrapeLockCounts(ctx, now); err != nil {
		s.logger.Error("Failed to scrape lock counts", zap.Error(err))
		errors = append(errors, err)
	}

	// Collect blocked sessions by lock type
	if err := s.scrapeBlockedSessions(ctx, now); err != nil {
		s.logger.Error("Failed to scrape blocked sessions by locks", zap.Error(err))
		errors = append(errors, err)
	}

	return errors
}

// scrapeLockCounts collects lock counts by type and mode
func (s *LockScraper) scrapeLockCounts(ctx context.Context, now pcommon.Timestamp) error {
	lockCounts, err := s.client.QueryLockCounts(ctx)
	if err != nil {
		return err
	}

	for _, lc := range lockCounts {
		if lc.GetCount() >= 0 {
			s.mb.RecordNewrelicoracledbLocksCountDataPoint(
				now,
				lc.GetCount(),
				s.instanceName,
				lc.GetLockType(),
				lc.GetLockMode(),
			)
		}
	}

	return nil
}

// scrapeBlockedSessions collects blocked sessions by lock type
func (s *LockScraper) scrapeBlockedSessions(ctx context.Context, now pcommon.Timestamp) error {
	objectCounts, err := s.client.QueryLockedObjectCounts(ctx)
	if err != nil {
		return err
	}

	for _, oc := range objectCounts {
		if oc.GetObjectCount() >= 0 {
			s.mb.RecordNewrelicoracledbLocksBlockedSessionsDataPoint(
				now,
				oc.GetObjectCount(),
				s.instanceName,
				oc.GetLockType(),
				oc.GetObjectType(),
			)
		}
	}

	return nil
}
