// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func (s *CoreScraper) scrapeSGASharedPoolLibraryCacheReloadRatioMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGASharedPoolLibraryCacheReloadRatio(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA shared pool library cache reload ratio", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbSgaSharedPoolLibraryCacheReloadRatioDataPoint(now, metric.Ratio, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA shared pool library cache reload ratio metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGASharedPoolLibraryCacheHitRatioMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGASharedPoolLibraryCacheHitRatio(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA shared pool library cache hit ratio", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbSgaSharedPoolLibraryCacheHitRatioDataPoint(now, metric.Ratio, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA shared pool library cache hit ratio metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGASharedPoolDictCacheMissRatioMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGASharedPoolDictCacheMissRatio(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA shared pool dictionary cache miss ratio", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbSgaSharedPoolDictCacheMissRatioDataPoint(now, metric.Ratio, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA shared pool dictionary cache miss ratio metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGALogBufferSpaceWaitsMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGALogBufferSpaceWaits(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA log buffer space waits", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbSgaLogBufferSpaceWaitsDataPoint(now, metric.Count, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA log buffer space waits metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGALogAllocRetriesMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGALogAllocRetries(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA log allocation retries", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		ratioValue := 0.0
		if metric.Ratio.Valid {
			ratioValue = metric.Ratio.Float64
		}

		s.mb.RecordNewrelicoracledbSgaLogAllocationRetriesRatioDataPoint(now, ratioValue, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA log allocation retries metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGAHitRatioMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGAHitRatio(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA hit ratio", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		ratioValue := 0.0
		if metric.Ratio.Valid {
			ratioValue = metric.Ratio.Float64
		}

		s.mb.RecordNewrelicoracledbSgaHitRatioDataPoint(now, ratioValue, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA hit ratio metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}
