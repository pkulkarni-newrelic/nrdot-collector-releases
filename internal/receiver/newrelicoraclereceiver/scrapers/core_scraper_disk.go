// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func (s *CoreScraper) scrapeReadWriteMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	if !s.config.Metrics.NewrelicoracledbDiskReads.Enabled &&
		!s.config.Metrics.NewrelicoracledbDiskWrites.Enabled &&
		!s.config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled &&
		!s.config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled &&
		!s.config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled &&
		!s.config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled {
		return nil
	}

	metrics, err := s.client.QueryDiskIOMetrics(ctx)
	if err != nil {
		s.logger.Error("Failed to query disk I/O metrics", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		if s.config.Metrics.NewrelicoracledbDiskReads.Enabled {
			s.mb.RecordNewrelicoracledbDiskReadsDataPoint(now, metric.PhysicalReads, s.instanceName, instanceID)
		}
		if s.config.Metrics.NewrelicoracledbDiskWrites.Enabled {
			s.mb.RecordNewrelicoracledbDiskWritesDataPoint(now, metric.PhysicalWrites, s.instanceName, instanceID)
		}
		if s.config.Metrics.NewrelicoracledbDiskBlocksRead.Enabled {
			s.mb.RecordNewrelicoracledbDiskBlocksReadDataPoint(now, metric.PhysicalBlockReads, s.instanceName, instanceID)
		}
		if s.config.Metrics.NewrelicoracledbDiskBlocksWritten.Enabled {
			s.mb.RecordNewrelicoracledbDiskBlocksWrittenDataPoint(now, metric.PhysicalBlockWrites, s.instanceName, instanceID)
		}
		if s.config.Metrics.NewrelicoracledbDiskReadTimeMilliseconds.Enabled {
			s.mb.RecordNewrelicoracledbDiskReadTimeMillisecondsDataPoint(now, metric.ReadTime, s.instanceName, instanceID)
		}
		if s.config.Metrics.NewrelicoracledbDiskWriteTimeMilliseconds.Enabled {
			s.mb.RecordNewrelicoracledbDiskWriteTimeMillisecondsDataPoint(now, metric.WriteTime, s.instanceName, instanceID)
		}

		metricCount++
	}

	s.logger.Debug("Disk I/O metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}
