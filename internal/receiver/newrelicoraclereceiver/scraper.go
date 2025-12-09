// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newrelicoraclereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver"

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/scrapers"
)

// Constants for scraper configuration
const (
	// keepAlive defines the connection keepalive duration
	keepAlive = 30 * time.Second
	// maxErrors defines the buffer size for error channel to prevent blocking
	maxErrors = 100
)

// Type definitions
type (
	// ScraperFunc represents a function that scrapes metrics and returns errors
	ScraperFunc func(context.Context) []error

	// dbProviderFunc represents a function that provides database connections
	dbProviderFunc func() (*sql.DB, error)
)

// newRelicOracleScraper orchestrates all metric collection scrapers for Oracle database monitoring
type newRelicOracleScraper struct {
	// Core scrapers for basic database metrics
	sessionScraper      *scrapers.SessionScraper
	tablespaceScraper   *scrapers.TablespaceScraper
	coreScraper         *scrapers.CoreScraper
	pdbScraper          *scrapers.PdbScraper
	systemScraper       *scrapers.SystemScraper
	connectionScraper   *scrapers.ConnectionScraper
	containerScraper    *scrapers.ContainerScraper
	racScraper          *scrapers.RacScraper
	databaseInfoScraper *scrapers.DatabaseInfoScraper

	// Query Performance Monitoring (QPM) scrapers
	slowQueriesScraper       *scrapers.SlowQueriesScraper
	executionPlanScraper     *scrapers.ExecutionPlanScraper
	waitEventBlockingScraper *scrapers.WaitEventBlockingScraper
	lockScraper              *scrapers.LockScraper
	childCursorsScraper      *scrapers.ChildCursorsScraper

	// Database and configuration
	db             *sql.DB
	client         client.OracleClient
	dbProviderFunc dbProviderFunc
	config         *Config

	// Metrics building and logging
	mb                   *metadata.MetricsBuilder
	metricsBuilderConfig metadata.MetricsBuilderConfig
	lb                   *metadata.LogsBuilder
	logsBuilderConfig    metadata.LogsBuilderConfig
	logger               *zap.Logger

	// Runtime configuration
	instanceName string
	hostName     string
	scrapeCfg    scraperhelper.ControllerConfig
	startTime    pcommon.Timestamp
}

// newScraper creates a new Oracle database metrics scraper
func newScraper(
	metricsBuilder *metadata.MetricsBuilder,
	metricsBuilderConfig metadata.MetricsBuilderConfig,
	logsBuilder *metadata.LogsBuilder,
	logsBuilderConfig metadata.LogsBuilderConfig,
	scrapeCfg scraperhelper.ControllerConfig,
	config *Config,
	logger *zap.Logger,
	providerFunc dbProviderFunc,
	instanceName, hostName string,
) (scraper.Metrics, error) {
	s := &newRelicOracleScraper{
		// Metrics and configuration
		mb:                   metricsBuilder,
		metricsBuilderConfig: metricsBuilderConfig,
		lb:                   logsBuilder,
		logsBuilderConfig:    logsBuilderConfig,
		config:               config,
		scrapeCfg:            scrapeCfg,

		// Database connection
		dbProviderFunc: providerFunc,

		// Runtime info
		logger:       logger,
		instanceName: instanceName,
		hostName:     hostName,
	}

	return scraper.NewMetrics(
		s.scrape,
		scraper.WithShutdown(s.shutdown),
		scraper.WithStart(s.start),
	)
}

// newLogsScraper creates a new logs scraper for Oracle execution plans
func newLogsScraper(
	logsBuilder *metadata.LogsBuilder,
	logsBuilderConfig metadata.LogsBuilderConfig,
	scrapeCfg scraperhelper.ControllerConfig,
	config *Config,
	logger *zap.Logger,
	providerFunc dbProviderFunc,
	instanceName, hostName string,
) (scraper.Logs, error) {
	s := &newRelicOracleScraper{
		// Logs and configuration
		lb:                logsBuilder,
		logsBuilderConfig: logsBuilderConfig,
		config:            config,
		scrapeCfg:         scrapeCfg,

		// Database connection
		dbProviderFunc: providerFunc,

		// Runtime info
		logger:       logger,
		instanceName: instanceName,
		hostName:     hostName,
	}

	return scraper.NewLogs(
		s.scrapeLogs,
		scraper.WithShutdown(s.shutdown),
		scraper.WithStart(s.startLogs),
	)
}

// start initializes the scraper and establishes database connections
func (s *newRelicOracleScraper) start(context.Context, component.Host) error {
	s.startTime = pcommon.NewTimestampFromTime(time.Now())

	// Establish database connection
	if err := s.initializeDatabase(); err != nil {
		return err
	}

	if err := s.initializeScrapers(); err != nil {
		return err
	}

	return nil
}

// initializeDatabase establishes the database connection
func (s *newRelicOracleScraper) initializeDatabase() error {
	db, err := s.dbProviderFunc()
	if err != nil {
		return fmt.Errorf("failed to open db connection: %w", err)
	}
	s.db = db
	s.client = client.NewSQLClient(db)
	return nil
}

// initializeScrapers initializes all metric scrapers
func (s *newRelicOracleScraper) initializeScrapers() error {
	if err := s.initializeCoreScrapers(); err != nil {
		return err
	}

	if err := s.initializeQPMScrapers(); err != nil {
		return err
	}

	return nil
}

func (s *newRelicOracleScraper) initializeCoreScrapers() error {
	s.sessionScraper = scrapers.NewSessionScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)
	s.tablespaceScraper = scrapers.NewTablespaceScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig, s.config.TablespaceFilter.IncludeTablespaces, s.config.TablespaceFilter.ExcludeTablespaces)

	var err error
	s.coreScraper, err = scrapers.NewCoreScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)
	if err != nil {
		return fmt.Errorf("failed to create core scraper: %w", err)
	}

	s.pdbScraper = scrapers.NewPdbScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)
	s.systemScraper = scrapers.NewSystemScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)

	s.connectionScraper, err = scrapers.NewConnectionScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection scraper: %w", err)
	}

	s.containerScraper, err = scrapers.NewContainerScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig, s.config.TablespaceFilter.IncludeTablespaces, s.config.TablespaceFilter.ExcludeTablespaces)
	if err != nil {
		return fmt.Errorf("failed to create container scraper: %w", err)
	}

	s.racScraper = scrapers.NewRacScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)
	s.databaseInfoScraper = scrapers.NewDatabaseInfoScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)

	return nil
}

func (s *newRelicOracleScraper) initializeQPMScrapers() error {
	s.slowQueriesScraper = scrapers.NewSlowQueriesScraper(
		s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig,
		s.config.QueryMonitoringResponseTimeThreshold,
		s.config.QueryMonitoringCountThreshold,
		s.config.QueryMonitoringIntervalSeconds,
		s.config.EnableIntervalBasedAveraging,
		s.config.IntervalCalculatorCacheTTLMinutes,
	)

	s.executionPlanScraper = scrapers.NewExecutionPlanScraper(s.client, s.lb, s.logger, s.instanceName, s.logsBuilderConfig)

	var err error
	s.waitEventBlockingScraper, err = scrapers.NewWaitEventBlockingScraper(
		s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig,
		s.config.QueryMonitoringCountThreshold,
	)
	if err != nil {
		return fmt.Errorf("failed to create wait event blocking scraper: %w", err)
	}

	s.childCursorsScraper = scrapers.NewChildCursorsScraper(s.client, s.mb, s.logger, s.instanceName, s.metricsBuilderConfig)

	s.lockScraper = scrapers.NewLockScraper(s.logger, s.client, s.mb, s.instanceName)

	return nil
}

// scrape orchestrates the collection of all Oracle database metrics
func (s *newRelicOracleScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	s.logger.Debug("Begin New Relic Oracle scrape")

	scrapeCtx := s.createScrapeContext(ctx)

	errChan := make(chan error, maxErrors)
	var wg sync.WaitGroup

	s.executeQPMScrapers(scrapeCtx, errChan)

	s.executeIndependentScrapers(scrapeCtx, errChan, &wg)

	scrapeErrors := s.collectScrapingErrors(scrapeCtx, errChan, &wg)

	metrics := s.buildMetrics()

	s.logScrapeCompletion(scrapeErrors)

	if len(scrapeErrors) > 0 {
		return metrics, scrapererror.NewPartialScrapeError(multierr.Combine(scrapeErrors...), len(scrapeErrors))
	}
	return metrics, nil
}

// scrapeLogs orchestrates the collection of Oracle execution plans as logs
func (s *newRelicOracleScraper) scrapeLogs(ctx context.Context) (plog.Logs, error) {
	s.logger.Debug("Begin New Relic Oracle logs scrape for execution plans")

	logs := plog.NewLogs()

	// Only scrape execution plans if event is enabled
	if !s.logsBuilderConfig.Events.NewrelicoracledbExecutionPlan.Enabled {
		s.logger.Debug("Execution plan event disabled, skipping logs scrape")
		return logs, nil
	}

	// Check if QPM is enabled
	if !s.config.EnableQueryMonitoring {
		s.logger.Debug("Query Performance Monitoring disabled, skipping execution plan scrape")
		return logs, nil
	}

	scrapeCtx := s.createScrapeContext(ctx)

	s.logger.Debug("Fetching query IDs for execution plans (no metrics)")
	queryIDs, slowQueryErrs := s.slowQueriesScraper.GetSlowQueryIDs(scrapeCtx)

	if len(slowQueryErrs) > 0 {
		s.logger.Warn("Errors occurred while getting query IDs for execution plans",
			zap.Int("error_count", len(slowQueryErrs)),
			zap.Error(multierr.Combine(slowQueryErrs...)))
	}

	s.logger.Info("Query IDs fetched for execution plans",
		zap.Int("query_ids_found", len(queryIDs)),
		zap.Strings("query_ids", queryIDs))

	if len(queryIDs) == 0 {
		s.logger.Debug("No query IDs found, skipping execution plan scrape")
		return logs, nil
	}

	s.logger.Debug("Getting SQL identifiers from wait events for execution plan logs (no metrics)",
		zap.Strings("query_ids", queryIDs))

	if s.waitEventBlockingScraper == nil {
		err := fmt.Errorf("waitEventBlockingScraper is not initialized")
		s.logger.Error("Cannot get SQL identifiers for execution plans", zap.Error(err))
		return logs, err
	}

	sqlIdentifiers, waitEventErrs := s.waitEventBlockingScraper.GetSQLIdentifiers(scrapeCtx, queryIDs)
	if len(waitEventErrs) > 0 {
		s.logger.Warn("Errors occurred while getting SQL identifiers from wait events for execution plan logs",
			zap.Int("error_count", len(waitEventErrs)))
	}

	if len(sqlIdentifiers) == 0 {
		s.logger.Info("No SQL identifiers found for execution plan logs")
		return logs, nil
	}

	s.logger.Debug("Scraping execution plans for logs",
		zap.Int("sql_identifiers", len(sqlIdentifiers)))

	executionPlanErrs := s.executionPlanScraper.ScrapeExecutionPlans(scrapeCtx, sqlIdentifiers)

	s.logger.Info("Execution plan scraper completed for logs",
		zap.Int("sql_identifiers_processed", len(sqlIdentifiers)),
		zap.Int("execution_plan_errors", len(executionPlanErrs)))

	logs = s.lb.Emit()

	if len(executionPlanErrs) > 0 {
		return logs, scrapererror.NewPartialScrapeError(multierr.Combine(executionPlanErrs...), len(executionPlanErrs))
	}

	return logs, nil
}

// startLogs initializes the logs scraper (execution plans only)
func (s *newRelicOracleScraper) startLogs(_ context.Context, _ component.Host) error {
	s.startTime = pcommon.NewTimestampFromTime(time.Now())

	if err := s.initializeDatabase(); err != nil {
		return err
	}

	s.logger.Info("Initializing logs scrapers for execution plans")

	s.client = client.NewSQLClient(s.db)

	tempSettings := receiver.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: s.logger,
		},
	}
	tempMb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), tempSettings)

	slowQueriesScraper := scrapers.NewSlowQueriesScraper(
		s.client,
		tempMb,
		s.logger,
		s.instanceName,
		metadata.DefaultMetricsBuilderConfig(),
		s.config.QueryMonitoringResponseTimeThreshold,
		s.config.QueryMonitoringCountThreshold,
		s.config.QueryMonitoringIntervalSeconds,
		s.config.EnableIntervalBasedAveraging,
		s.config.IntervalCalculatorCacheTTLMinutes,
	)
	s.slowQueriesScraper = slowQueriesScraper

	executionPlanScraper := scrapers.NewExecutionPlanScraper(
		s.client,
		s.lb,
		s.logger,
		s.instanceName,
		s.logsBuilderConfig,
	)
	s.executionPlanScraper = executionPlanScraper

	waitEventBlockingScraper, err := scrapers.NewWaitEventBlockingScraper(
		s.client,
		tempMb,
		s.logger,
		s.instanceName,
		metadata.DefaultMetricsBuilderConfig(),
		s.config.QueryMonitoringCountThreshold,
	)
	if err != nil {
		return fmt.Errorf("failed to create wait event blocking scraper for logs: %w", err)
	}
	s.waitEventBlockingScraper = waitEventBlockingScraper

	return nil
}

// createScrapeContext creates a context with timeout for scraping operations
func (s *newRelicOracleScraper) createScrapeContext(ctx context.Context) context.Context {
	if s.scrapeCfg.Timeout > 0 {
		scrapeCtx, cancel := context.WithTimeout(ctx, s.scrapeCfg.Timeout)
		_ = cancel
		return scrapeCtx
	}
	return ctx
}

// executeQPMScrapers executes Query Performance Monitoring scrapers sequentially
func (s *newRelicOracleScraper) executeQPMScrapers(ctx context.Context, errChan chan<- error) {
	if !s.config.EnableQueryMonitoring {
		s.logger.Debug("Query Performance Monitoring disabled, skipping QPM scrapers")
		return
	}

	s.logger.Debug("Starting slow queries scraper")
	queryIDs, slowQueryErrs := s.slowQueriesScraper.ScrapeSlowQueries(ctx)

	s.logger.Info("Slow queries scraper completed",
		zap.Int("query_ids_found", len(queryIDs)),
		zap.Strings("query_ids", queryIDs),
		zap.Int("slow_query_errors", len(slowQueryErrs)))

	s.sendErrorsToChannel(errChan, slowQueryErrs, "slow query")

	s.logger.Debug("Starting wait events & blocking scraper for slow query SQL_IDs",
		zap.Int("slow_query_ids", len(queryIDs)))
	waitEventSQLIdentifiers, waitEventErrs := s.waitEventBlockingScraper.ScrapeWaitEventsAndBlocking(ctx, queryIDs)

	s.logger.Info("Wait events & blocking scraper completed",
		zap.Int("unique_sql_identifiers", len(waitEventSQLIdentifiers)),
		zap.Int("errors", len(waitEventErrs)))

	s.sendErrorsToChannel(errChan, waitEventErrs, "wait events & blocking")

	if len(waitEventSQLIdentifiers) > 0 {
		s.executeChildCursors(ctx, errChan, waitEventSQLIdentifiers)
	} else {
		s.logger.Debug("No SQL identifiers from wait events, skipping child cursor scraping")
	}
}

// executeChildCursors executes child cursor scraper for SQL identifiers from wait events
func (s *newRelicOracleScraper) executeChildCursors(ctx context.Context, errChan chan<- error, sqlIdentifiers []models.SQLIdentifier) {
	s.logger.Debug("Starting child cursor scraping from wait events",
		zap.Int("sql_identifiers", len(sqlIdentifiers)))

	if len(sqlIdentifiers) == 0 {
		s.logger.Debug("No SQL identifiers from wait events")
		return
	}

	childCursorErrs := s.childCursorsScraper.ScrapeChildCursorsForIdentifiers(ctx, sqlIdentifiers, s.config.ChildCursorsPerSQLID)
	if len(childCursorErrs) > 0 {
		s.logger.Warn("Errors occurred while scraping child cursor metrics",
			zap.Int("error_count", len(childCursorErrs)))
		s.sendErrorsToChannel(errChan, childCursorErrs, "child cursors")
	}

	s.logger.Info("Child cursor scraping completed",
		zap.Int("sql_identifiers_processed", len(sqlIdentifiers)),
		zap.Int("errors", len(childCursorErrs)))
}

// executeIndependentScrapers launches concurrent scrapers for independent metrics
func (s *newRelicOracleScraper) executeIndependentScrapers(ctx context.Context, errChan chan<- error, wg *sync.WaitGroup) {
	scraperFuncs := s.getIndependentScraperFunctions()

	for i, scraperFunc := range scraperFuncs {
		wg.Add(1)
		go s.runScraperWithErrorHandling(ctx, errChan, wg, i, scraperFunc)
	}
}

// getIndependentScraperFunctions returns the list of scraper functions that can run independently
func (s *newRelicOracleScraper) getIndependentScraperFunctions() []ScraperFunc {
	scraperFuncs := []ScraperFunc{
		s.sessionScraper.ScrapeSessionCount,
		s.tablespaceScraper.ScrapeTablespaceMetrics,
		s.coreScraper.ScrapeCoreMetrics,
		s.pdbScraper.ScrapePdbMetrics,
		s.systemScraper.ScrapeSystemMetrics,
		s.connectionScraper.ScrapeConnectionMetrics,
		s.containerScraper.ScrapeContainerMetrics,
		s.racScraper.ScrapeRacMetrics,
		s.databaseInfoScraper.ScrapeDatabaseInfo,
		s.databaseInfoScraper.ScrapeHostingInfo,
		s.databaseInfoScraper.ScrapeDatabaseRole,
	}

	// Add lock scraper
	scraperFuncs = append(scraperFuncs,
		s.lockScraper.ScrapeLocks,
	)

	return scraperFuncs
}

// runScraperWithErrorHandling executes a single scraper function with proper error handling
func (s *newRelicOracleScraper) runScraperWithErrorHandling(ctx context.Context, errChan chan<- error, wg *sync.WaitGroup, index int, scraperFunc ScraperFunc) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		s.logger.Debug("Context cancelled before starting scraper",
			zap.Int("scraper_index", index),
			zap.Error(ctx.Err()))
		return
	default:
	}

	s.logger.Debug("Starting scraper", zap.Int("scraper_index", index))
	startTime := time.Now()

	errs := scraperFunc(ctx)

	duration := time.Since(startTime)
	s.logger.Debug("Completed scraper",
		zap.Int("scraper_index", index),
		zap.Duration("duration", duration),
		zap.Int("error_count", len(errs)))

	s.sendErrorsToChannelWithCancellation(ctx, errChan, errs, index)
}

// collectScrapingErrors collects all errors from concurrent scrapers
func (s *newRelicOracleScraper) collectScrapingErrors(ctx context.Context, errChan chan error, wg *sync.WaitGroup) []error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
		close(errChan)
	}()

	var scrapeErrors []error
	collecting := true

	for collecting {
		select {
		case err, ok := <-errChan:
			if !ok {
				collecting = false
				break
			}
			scrapeErrors = append(scrapeErrors, err)
		case <-ctx.Done():
			s.logger.Warn("Context cancelled while collecting errors", zap.Error(ctx.Err()))
			scrapeErrors = append(scrapeErrors, fmt.Errorf("scrape operation cancelled: %w", ctx.Err()))
			collecting = false
		case <-done:
			for err := range errChan {
				scrapeErrors = append(scrapeErrors, err)
			}
			collecting = false
		}
	}

	return scrapeErrors
}

// buildMetrics constructs the final metrics output
func (s *newRelicOracleScraper) buildMetrics() pmetric.Metrics {
	rb := s.mb.NewResourceBuilder()
	rb.SetNewrelicoracledbInstanceName(s.instanceName)
	rb.SetHostName(s.hostName)
	return s.mb.Emit(metadata.WithResource(rb.Emit()))
}

// logScrapeCompletion logs the completion of the scraping operation
func (s *newRelicOracleScraper) logScrapeCompletion(scrapeErrors []error) {
	scraperCount := len(s.getIndependentScraperFunctions())
	s.logger.Debug("Done New Relic Oracle scraping",
		zap.Int("total_errors", len(scrapeErrors)),
		zap.Int("scrapers_executed", scraperCount),
	)
}

// sendErrorsToChannel sends a slice of errors to the error channel
func (s *newRelicOracleScraper) sendErrorsToChannel(errChan chan<- error, errs []error, scraperType string) {
	for _, err := range errs {
		select {
		case errChan <- err:
		default:
			s.logger.Warn("Error channel full, dropping error",
				zap.Error(err),
				zap.String("scraper_type", scraperType))
		}
	}
}

// sendErrorsToChannelWithCancellation sends errors to channel with context cancellation check
func (s *newRelicOracleScraper) sendErrorsToChannelWithCancellation(ctx context.Context, errChan chan<- error, errs []error, scraperIndex int) {
	for _, err := range errs {
		select {
		case errChan <- err:
		case <-ctx.Done():
			s.logger.Debug("Context cancelled while sending errors",
				zap.Int("scraper_index", scraperIndex),
				zap.Error(ctx.Err()))
			return
		default:
			s.logger.Warn("Error channel full, dropping error",
				zap.Error(err),
				zap.Int("scraper_index", scraperIndex))
		}
	}
}

// shutdown closes the database connection
func (s *newRelicOracleScraper) shutdown(_ context.Context) error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
