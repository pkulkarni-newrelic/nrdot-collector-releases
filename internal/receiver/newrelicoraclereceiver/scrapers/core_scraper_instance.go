// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func (s *CoreScraper) scrapeLockedAccountsMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	if !s.config.Metrics.NewrelicoracledbLockedAccounts.Enabled {
		return nil
	}

	metrics, err := s.client.QueryLockedAccounts(ctx)
	if err != nil {
		s.logger.Error("Failed to query locked accounts", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbLockedAccountsDataPoint(now, metric.LockedAccounts, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("Locked accounts metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeGlobalNameInstanceMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QueryGlobalName(ctx)
	if err != nil {
		s.logger.Error("Failed to query global name", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		s.mb.RecordNewrelicoracledbGlobalNameDataPoint(now, 1, s.instanceName, metric.GlobalName)
		metricCount++
	}

	s.logger.Debug("Global name metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeDBIDInstanceMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QueryDBID(ctx)
	if err != nil {
		s.logger.Error("Failed to query database ID", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		s.mb.RecordNewrelicoracledbDbIDDataPoint(now, 1, s.instanceName, metric.DBID)
		metricCount++
	}

	s.logger.Debug("Database ID metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeLongRunningQueriesMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QueryLongRunningQueries(ctx)
	if err != nil {
		s.logger.Error("Failed to query long running queries", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbLongRunningQueriesDataPoint(now, metric.Total, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("Long running queries metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}
