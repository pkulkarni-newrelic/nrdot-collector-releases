// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func (s *CoreScraper) scrapePGAMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QueryPGAMetrics(ctx)
	if err != nil {
		s.logger.Error("Failed to query PGA metrics", zap.Error(err))
		return []error{err}
	}

	instanceMetrics := make(map[string]map[string]int64)
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		if instanceMetrics[instanceID] == nil {
			instanceMetrics[instanceID] = make(map[string]int64)
		}

		instanceMetrics[instanceID][metric.Name] = int64(metric.Value)
	}

	metricCount := 0
	for instanceID, metricVals := range instanceMetrics {
		if val, exists := metricVals["total PGA inuse"]; exists {
			s.mb.RecordNewrelicoracledbMemoryPgaInUseBytesDataPoint(now, val, s.instanceName, instanceID)
		}
		if val, exists := metricVals["total PGA allocated"]; exists {
			s.mb.RecordNewrelicoracledbMemoryPgaAllocatedBytesDataPoint(now, val, s.instanceName, instanceID)
		}
		if val, exists := metricVals["total freeable PGA memory"]; exists {
			s.mb.RecordNewrelicoracledbMemoryPgaFreeableBytesDataPoint(now, val, s.instanceName, instanceID)
		}
		if val, exists := metricVals["global memory bound"]; exists {
			s.mb.RecordNewrelicoracledbMemoryPgaMaxSizeBytesDataPoint(now, val, s.instanceName, instanceID)
		}

		metricCount++
	}

	s.logger.Debug("PGA memory metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGAUGATotalMemoryMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGAUGATotalMemory(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA UGA total memory", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbMemorySgaUgaTotalBytesDataPoint(now, metric.Sum, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA UGA total memory metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGASharedPoolLibraryCacheMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGASharedPoolLibraryCache(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA shared pool library cache", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbMemorySgaSharedPoolLibraryCacheSharableBytesDataPoint(now, metric.Sum, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA shared pool library cache metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGASharedPoolLibraryCacheUserMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGASharedPoolLibraryCacheUser(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA shared pool library cache user", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)
		s.mb.RecordNewrelicoracledbMemorySgaSharedPoolLibraryCacheUserBytesDataPoint(now, metric.Sum, s.instanceName, instanceID)
		metricCount++
	}

	s.logger.Debug("SGA shared pool library cache user metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeSGAMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySGAMetrics(ctx)
	if err != nil {
		s.logger.Error("Failed to query SGA metrics", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		valueInt := int64(0)
		if metric.Value.Valid {
			valueInt = metric.Value.Int64
		}

		switch metric.Name {
		case "Fixed Size":
			s.mb.RecordNewrelicoracledbSgaFixedSizeBytesDataPoint(now, valueInt, s.instanceName, instanceID)
		case "Redo Buffers":
			s.mb.RecordNewrelicoracledbSgaRedoBuffersBytesDataPoint(now, valueInt, s.instanceName, instanceID)
		default:
			s.logger.Debug("Unknown SGA metric", zap.String("name", metric.Name))
		}

		metricCount++
	}

	s.logger.Debug("SGA metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}
