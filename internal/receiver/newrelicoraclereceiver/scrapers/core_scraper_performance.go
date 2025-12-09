// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func (s *CoreScraper) scrapeSysstatMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QuerySysstatMetrics(ctx)
	if err != nil {
		s.logger.Error("Failed to query sysstat metrics", zap.Error(err))
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
		case "redo buffer allocation retries":
			s.mb.RecordNewrelicoracledbSgaLogBufferRedoAllocationRetriesDataPoint(now, valueInt, s.instanceName, instanceID)
		case "redo entries":
			s.mb.RecordNewrelicoracledbSgaLogBufferRedoEntriesDataPoint(now, valueInt, s.instanceName, instanceID)
		case "sorts (memory)":
			s.mb.RecordNewrelicoracledbSortsMemoryDataPoint(now, valueInt, s.instanceName, instanceID)
		case "sorts (disk)":
			s.mb.RecordNewrelicoracledbSortsDiskDataPoint(now, valueInt, s.instanceName, instanceID)
		default:
			s.logger.Debug("Unknown sysstat metric", zap.String("name", metric.Name))
		}

		metricCount++
	}

	s.logger.Debug("Sysstat metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeRollbackSegmentsMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QueryRollbackSegmentsMetrics(ctx)
	if err != nil {
		s.logger.Error("Failed to query rollback segments metrics", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		getsValue := int64(0)
		if metric.Gets.Valid {
			getsValue = metric.Gets.Int64
		}

		waitsValue := int64(0)
		if metric.Waits.Valid {
			waitsValue = metric.Waits.Int64
		}

		ratioValue := 0.0
		if metric.Ratio.Valid {
			ratioValue = metric.Ratio.Float64
		}

		s.mb.RecordNewrelicoracledbRollbackSegmentsGetsDataPoint(now, getsValue, s.instanceName, instanceID)
		s.mb.RecordNewrelicoracledbRollbackSegmentsWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		s.mb.RecordNewrelicoracledbRollbackSegmentsWaitRatioDataPoint(now, ratioValue, s.instanceName, instanceID)

		metricCount++
	}

	s.logger.Debug("Rollback segments metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}

func (s *CoreScraper) scrapeRedoLogWaitsMetrics(ctx context.Context, now pcommon.Timestamp) []error {
	metrics, err := s.client.QueryRedoLogWaitsMetrics(ctx)
	if err != nil {
		s.logger.Error("Failed to query redo log waits metrics", zap.Error(err))
		return []error{err}
	}

	metricCount := 0
	for _, metric := range metrics {
		instanceID := getInstanceIDString(metric.InstID)

		waitsValue := int64(0)
		if metric.TotalWaits.Valid {
			waitsValue = metric.TotalWaits.Int64
		}

		switch {
		case strings.Contains(metric.Event, "log file parallel write"):
			s.mb.RecordNewrelicoracledbRedoLogParallelWriteWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		case strings.Contains(metric.Event, "log file switch completion"):
			s.mb.RecordNewrelicoracledbRedoLogSwitchCompletionWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		case strings.Contains(metric.Event, "log file switch (check"):
			s.mb.RecordNewrelicoracledbRedoLogSwitchCheckpointIncompleteWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		case strings.Contains(metric.Event, "log file switch (arch"):
			s.mb.RecordNewrelicoracledbRedoLogSwitchArchivingNeededWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		case strings.Contains(metric.Event, "buffer busy waits"):
			s.mb.RecordNewrelicoracledbSgaBufferBusyWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		case strings.Contains(metric.Event, "freeBufferWaits"):
			s.mb.RecordNewrelicoracledbSgaFreeBufferWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		case strings.Contains(metric.Event, "free buffer inspected"):
			s.mb.RecordNewrelicoracledbSgaFreeBufferInspectedWaitsDataPoint(now, waitsValue, s.instanceName, instanceID)
		default:
			s.logger.Debug("Unknown redo log waits event", zap.String("event", metric.Event))
		}

		metricCount++
	}

	s.logger.Debug("Redo log waits metrics scrape completed", zap.Int("metrics", metricCount))

	return nil
}
