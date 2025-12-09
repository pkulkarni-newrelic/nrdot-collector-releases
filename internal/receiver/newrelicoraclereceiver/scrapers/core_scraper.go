// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

// CoreScraper handles Oracle core database metrics
type CoreScraper struct {
	client       client.OracleClient
	mb           *metadata.MetricsBuilder
	logger       *zap.Logger
	instanceName string
	config       metadata.MetricsBuilderConfig
}

// NewCoreScraper creates a new core scraper
func NewCoreScraper(c client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, config metadata.MetricsBuilderConfig) (*CoreScraper, error) {
	if c == nil {
		return nil, errors.New("client cannot be nil")
	}
	if mb == nil {
		return nil, errors.New("metrics builder cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	if instanceName == "" {
		return nil, errors.New("instance name cannot be empty")
	}

	return &CoreScraper{
		client:       c,
		mb:           mb,
		logger:       logger,
		instanceName: instanceName,
		config:       config,
	}, nil
}

// ScrapeCoreMetrics collects Oracle core database metrics
func (s *CoreScraper) ScrapeCoreMetrics(ctx context.Context) []error {
	var errors []error

	s.logger.Debug("Scraping Oracle core database metrics")
	now := pcommon.NewTimestampFromTime(time.Now())

	// Scrape locked accounts metrics
	errors = append(errors, s.scrapeLockedAccountsMetrics(ctx, now)...)

	// Scrape read/write disk I/O metrics
	errors = append(errors, s.scrapeReadWriteMetrics(ctx, now)...)

	// Scrape PGA memory metrics
	errors = append(errors, s.scrapePGAMetrics(ctx, now)...)

	// Scrape global name instance metrics
	errors = append(errors, s.scrapeGlobalNameInstanceMetrics(ctx, now)...)

	// Scrape database ID instance metrics
	errors = append(errors, s.scrapeDBIDInstanceMetrics(ctx, now)...)

	// Scrape long running queries metrics
	errors = append(errors, s.scrapeLongRunningQueriesMetrics(ctx, now)...)

	// Scrape SGA UGA total memory metrics
	errors = append(errors, s.scrapeSGAUGATotalMemoryMetrics(ctx, now)...)

	// Scrape SGA shared pool library cache metrics
	errors = append(errors, s.scrapeSGASharedPoolLibraryCacheMetrics(ctx, now)...)

	// Scrape SGA shared pool library cache user metrics
	errors = append(errors, s.scrapeSGASharedPoolLibraryCacheUserMetrics(ctx, now)...)

	// Scrape SGA shared pool library cache reload ratio metrics
	errors = append(errors, s.scrapeSGASharedPoolLibraryCacheReloadRatioMetrics(ctx, now)...)

	// Scrape SGA shared pool library cache hit ratio metrics
	errors = append(errors, s.scrapeSGASharedPoolLibraryCacheHitRatioMetrics(ctx, now)...)

	// Scrape SGA shared pool dictionary cache miss ratio metrics
	errors = append(errors, s.scrapeSGASharedPoolDictCacheMissRatioMetrics(ctx, now)...)

	// Scrape SGA log buffer space waits metrics
	errors = append(errors, s.scrapeSGALogBufferSpaceWaitsMetrics(ctx, now)...)

	// Scrape SGA log allocation retries metrics
	errors = append(errors, s.scrapeSGALogAllocRetriesMetrics(ctx, now)...)

	// Scrape SGA hit ratio metrics
	errors = append(errors, s.scrapeSGAHitRatioMetrics(ctx, now)...)

	// Scrape sysstat metrics
	errors = append(errors, s.scrapeSysstatMetrics(ctx, now)...)

	// Scrape SGA metrics
	errors = append(errors, s.scrapeSGAMetrics(ctx, now)...)

	// Scrape rollback segments metrics
	errors = append(errors, s.scrapeRollbackSegmentsMetrics(ctx, now)...)

	// Scrape redo log waits metrics
	errors = append(errors, s.scrapeRedoLogWaitsMetrics(ctx, now)...)

	return errors
}
