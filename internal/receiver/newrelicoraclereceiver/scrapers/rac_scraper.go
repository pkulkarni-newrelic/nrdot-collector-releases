// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

type RacScraper struct {
	client               client.OracleClient
	mb                   *metadata.MetricsBuilder
	logger               *zap.Logger
	instanceName         string
	metricsBuilderConfig metadata.MetricsBuilderConfig
	isRacMode            *bool
	racModeMutex         sync.RWMutex
}

func NewRacScraper(c client.OracleClient, mb *metadata.MetricsBuilder, logger *zap.Logger, instanceName string, config metadata.MetricsBuilderConfig) *RacScraper {
	return &RacScraper{
		client:               c,
		mb:                   mb,
		logger:               logger,
		instanceName:         instanceName,
		metricsBuilderConfig: config,
	}
}

func (s *RacScraper) isRacEnabled(ctx context.Context) (bool, error) {
	s.racModeMutex.RLock()
	if s.isRacMode != nil {
		result := *s.isRacMode
		s.racModeMutex.RUnlock()
		return result, nil
	}
	s.racModeMutex.RUnlock()

	s.racModeMutex.Lock()
	defer s.racModeMutex.Unlock()

	if s.isRacMode != nil {
		return *s.isRacMode, nil
	}

	detection, err := s.client.QueryRACDetection(ctx)
	if err != nil {
		return false, err
	}

	racEnabled := detection.ClusterDB.Valid && strings.ToUpper(detection.ClusterDB.String) == "TRUE"
	s.isRacMode = &racEnabled
	s.logger.Debug("RAC mode detection", zap.Bool("enabled", racEnabled))

	return racEnabled, nil
}

func (s *RacScraper) isASMAvailable(ctx context.Context) (bool, error) {
	detection, err := s.client.QueryASMDetection(ctx)
	if err != nil {
		s.logger.Debug("ASM not configured", zap.Error(err))
		return false, nil
	}

	asmAvailable := detection.ASMCount > 0
	s.logger.Debug("ASM detection", zap.Bool("available", asmAvailable), zap.Int("diskgroups", detection.ASMCount))

	return asmAvailable, nil
}

func (s *RacScraper) ScrapeRacMetrics(ctx context.Context) []error {
	racEnabled, err := s.isRacEnabled(ctx)
	if err != nil {
		return []error{err}
	}

	asmAvailable, err := s.isASMAvailable(ctx)
	if err != nil {
		return []error{err}
	}

	if !racEnabled && !asmAvailable {
		return nil
	}

	if ctx.Err() != nil {
		return []error{ctx.Err()}
	}

	var scrapers []func(context.Context) []error
	var scraperCount int

	if asmAvailable {
		scrapers = append(scrapers, s.scrapeASMDiskGroups)
		scraperCount++
	}

	if racEnabled {
		scrapers = append(scrapers,
			s.scrapeClusterWaitEvents,
			s.scrapeInstanceStatus,
			s.scrapeActiveServices,
		)
		scraperCount += 3
	}

	if scraperCount == 0 {
		return nil
	}

	errorChan := make(chan []error, scraperCount)
	var wg sync.WaitGroup
	wg.Add(scraperCount)

	for _, scraper := range scrapers {
		go func(fn func(context.Context) []error) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				errorChan <- []error{ctx.Err()}
				return
			default:
			}
			if errs := fn(ctx); len(errs) > 0 {
				errorChan <- errs
			} else {
				errorChan <- nil
			}
		}(scraper)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	var allErrors []error
	for errors := range errorChan {
		if errors != nil {
			allErrors = append(allErrors, errors...)
		}
	}

	if ctx.Err() != nil {
		allErrors = append(allErrors, ctx.Err())
	}

	return allErrors
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func stringStatusToBinary(status, expectedValue string) int64 {
	if strings.ToUpper(status) == strings.ToUpper(expectedValue) {
		return 1
	}
	return 0
}

func (s *RacScraper) scrapeASMDiskGroups(ctx context.Context) []error {
	if !s.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupTotalMb.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupFreeMb.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupOfflineDisks.Enabled {
		return nil
	}

	var scrapeErrors []error

	diskGroups, err := s.client.QueryASMDiskGroups(ctx)
	if err != nil {
		return []error{err}
	}

	for _, diskGroup := range diskGroups {
		if !diskGroup.Name.Valid {
			continue
		}

		now := pcommon.NewTimestampFromTime(time.Now())

		if diskGroup.TotalMB.Valid && s.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupTotalMb.Enabled {
			s.mb.RecordNewrelicoracledbAsmDiskgroupTotalMbDataPoint(now, diskGroup.TotalMB.Float64, s.instanceName, diskGroup.Name.String)
		}
		if diskGroup.FreeMB.Valid && s.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupFreeMb.Enabled {
			s.mb.RecordNewrelicoracledbAsmDiskgroupFreeMbDataPoint(now, diskGroup.FreeMB.Float64, s.instanceName, diskGroup.Name.String)
		}
		if diskGroup.OfflineDisks.Valid && s.metricsBuilderConfig.Metrics.NewrelicoracledbAsmDiskgroupOfflineDisks.Enabled {
			s.mb.RecordNewrelicoracledbAsmDiskgroupOfflineDisksDataPoint(now, int64(diskGroup.OfflineDisks.Float64), s.instanceName, diskGroup.Name.String)
		}
	}

	return scrapeErrors
}

func (s *RacScraper) scrapeClusterWaitEvents(ctx context.Context) []error {
	if !s.metricsBuilderConfig.Metrics.NewrelicoracledbRacWaitTime.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacTotalWaits.Enabled {
		return nil
	}

	var scrapeErrors []error

	events, err := s.client.QueryClusterWaitEvents(ctx)
	if err != nil {
		return []error{err}
	}

	for _, event := range events {
		if !event.InstID.Valid || !event.Event.Valid {
			continue
		}

		now := pcommon.NewTimestampFromTime(time.Now())
		instanceIDStr := event.InstID.String
		eventName := event.Event.String

		if event.TimeWaitedMicro.Valid && s.metricsBuilderConfig.Metrics.NewrelicoracledbRacWaitTime.Enabled {
			s.mb.RecordNewrelicoracledbRacWaitTimeDataPoint(now, event.TimeWaitedMicro.Float64, s.instanceName, instanceIDStr, eventName)
		}
		if event.TotalWaits.Valid && s.metricsBuilderConfig.Metrics.NewrelicoracledbRacTotalWaits.Enabled {
			s.mb.RecordNewrelicoracledbRacTotalWaitsDataPoint(now, int64(event.TotalWaits.Float64), s.instanceName, instanceIDStr, eventName)
		}
	}

	return scrapeErrors
}

func (s *RacScraper) scrapeInstanceStatus(ctx context.Context) []error {
	// Check if any instance status metrics are enabled
	if !s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceStatus.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceUptimeSeconds.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceDatabaseStatus.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceActiveState.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceLoginsAllowed.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceArchiverStarted.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceVersionInfo.Enabled {
		return nil
	}

	var scrapeErrors []error

	instances, err := s.client.QueryRACInstanceStatus(ctx)
	if err != nil {
		return []error{err}
	}

	for _, instance := range instances {
		if !instance.InstID.Valid || !instance.Status.Valid {
			continue
		}

		now := pcommon.NewTimestampFromTime(time.Now())
		instanceIDStr := instance.InstID.String
		statusStr := instance.Status.String
		instanceNameStr := nullStringToString(instance.InstanceName)
		hostNameStr := nullStringToString(instance.HostName)
		databaseStatusStr := nullStringToString(instance.DatabaseStatus)
		activeStateStr := nullStringToString(instance.ActiveState)
		loginsStr := nullStringToString(instance.Logins)
		archiverStr := nullStringToString(instance.Archiver)
		versionStr := nullStringToString(instance.Version)

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceStatus.Enabled {
			statusValue := stringStatusToBinary(statusStr, "OPEN")
			s.mb.RecordNewrelicoracledbRacInstanceStatusDataPoint(now, statusValue, s.instanceName, instanceIDStr, instanceNameStr, hostNameStr, statusStr)
		}

		if instance.StartupTime.Valid && s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceUptimeSeconds.Enabled {
			uptime := time.Since(instance.StartupTime.Time).Seconds()
			s.mb.RecordNewrelicoracledbRacInstanceUptimeSecondsDataPoint(now, int64(uptime), s.instanceName, instanceIDStr, instanceNameStr, hostNameStr)
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceDatabaseStatus.Enabled {
			dbStatusValue := stringStatusToBinary(databaseStatusStr, "ACTIVE")
			s.mb.RecordNewrelicoracledbRacInstanceDatabaseStatusDataPoint(now, dbStatusValue, s.instanceName, instanceIDStr, instanceNameStr, hostNameStr, databaseStatusStr)
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceActiveState.Enabled {
			activeStateValue := stringStatusToBinary(activeStateStr, "NORMAL")
			s.mb.RecordNewrelicoracledbRacInstanceActiveStateDataPoint(now, activeStateValue, s.instanceName, instanceIDStr, instanceNameStr, hostNameStr, activeStateStr)
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceLoginsAllowed.Enabled {
			loginsValue := stringStatusToBinary(loginsStr, "ALLOWED")
			s.mb.RecordNewrelicoracledbRacInstanceLoginsAllowedDataPoint(now, loginsValue, s.instanceName, instanceIDStr, instanceNameStr, hostNameStr, loginsStr)
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceArchiverStarted.Enabled {
			archiverValue := stringStatusToBinary(archiverStr, "STARTED")
			s.mb.RecordNewrelicoracledbRacInstanceArchiverStartedDataPoint(now, archiverValue, s.instanceName, instanceIDStr, instanceNameStr, hostNameStr, archiverStr)
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacInstanceVersionInfo.Enabled {
			s.mb.RecordNewrelicoracledbRacInstanceVersionInfoDataPoint(now, 1, s.instanceName, instanceNameStr, hostNameStr, versionStr)
		}
	}

	return scrapeErrors
}

func (s *RacScraper) scrapeActiveServices(ctx context.Context) []error {
	if !s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceInstanceID.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceNetworkConfig.Enabled &&
		!s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceClbConfig.Enabled {
		return nil
	}

	var scrapeErrors []error

	services, err := s.client.QueryRACActiveServices(ctx)
	if err != nil {
		return []error{err}
	}

	for _, service := range services {
		if !service.ServiceName.Valid || !service.InstID.Valid {
			continue
		}

		now := pcommon.NewTimestampFromTime(time.Now())
		serviceNameStr := service.ServiceName.String
		instanceIDStr := service.InstID.String

		// Record metrics only if enabled
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceInstanceID.Enabled {
			instanceIDInt, _ := strconv.ParseFloat(instanceIDStr, 64)
			s.mb.RecordNewrelicoracledbRacServiceInstanceIDDataPoint(now, instanceIDInt, s.instanceName, serviceNameStr, instanceIDStr)
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceNetworkConfig.Enabled {
			s.mb.RecordNewrelicoracledbRacServiceNetworkConfigDataPoint(now, 1, s.instanceName, serviceNameStr, nullStringToString(service.NetworkName))
		}

		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceClbConfig.Enabled {
			s.mb.RecordNewrelicoracledbRacServiceClbConfigDataPoint(now, 1, s.instanceName, serviceNameStr, nullStringToString(service.ClbGoal))
		}

		// Service goal configuration
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceGoalConfig.Enabled {
			s.mb.RecordNewrelicoracledbRacServiceGoalConfigDataPoint(now, 1, s.instanceName, serviceNameStr, nullStringToString(service.Goal))
		}

		// Service blocked status (1 if blocked, 0 if not)
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceBlockedStatus.Enabled {
			blockedValue := stringStatusToBinary(nullStringToString(service.Blocked), "YES")
			s.mb.RecordNewrelicoracledbRacServiceBlockedStatusDataPoint(now, blockedValue, s.instanceName, serviceNameStr, nullStringToString(service.Blocked))
		}

		// Fast Application Notification (FAN) enabled status (1 if enabled, 0 if not)
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceFanEnabled.Enabled {
			fanValue := stringStatusToBinary(nullStringToString(service.AqHaNotification), "YES")
			s.mb.RecordNewrelicoracledbRacServiceFanEnabledDataPoint(now, fanValue, s.instanceName, serviceNameStr, nullStringToString(service.AqHaNotification))
		}

		// Transaction Guard enabled status (1 if enabled, 0 if not)
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceTransactionGuardEnabled.Enabled {
			tgValue := stringStatusToBinary(nullStringToString(service.CommitOutcome), "TRUE")
			s.mb.RecordNewrelicoracledbRacServiceTransactionGuardEnabledDataPoint(now, tgValue, s.instanceName, serviceNameStr, nullStringToString(service.CommitOutcome))
		}

		// Session drain timeout in seconds
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceDrainTimeoutSeconds.Enabled {
			var drainTimeout int64
			if service.DrainTimeout.Valid {
				drainTimeout = int64(service.DrainTimeout.Float64)
			}
			drainTimeoutStr := fmt.Sprintf("%.0f", service.DrainTimeout.Float64)
			if !service.DrainTimeout.Valid {
				drainTimeoutStr = "0"
			}
			s.mb.RecordNewrelicoracledbRacServiceDrainTimeoutSecondsDataPoint(now, drainTimeout, s.instanceName, serviceNameStr, drainTimeoutStr)
		}

		// Application Continuity replay initiation timeout in seconds
		if s.metricsBuilderConfig.Metrics.NewrelicoracledbRacServiceReplayTimeoutSeconds.Enabled {
			var replayTimeout int64
			if service.ReplayInitiationTimeout.Valid {
				replayTimeout = int64(service.ReplayInitiationTimeout.Float64)
			}
			replayTimeoutStr := fmt.Sprintf("%.0f", service.ReplayInitiationTimeout.Float64)
			if !service.ReplayInitiationTimeout.Valid {
				replayTimeoutStr = "0"
			}
			s.mb.RecordNewrelicoracledbRacServiceReplayTimeoutSecondsDataPoint(now, replayTimeout, s.instanceName, serviceNameStr, replayTimeoutStr)
		}
	}

	return scrapeErrors
}
