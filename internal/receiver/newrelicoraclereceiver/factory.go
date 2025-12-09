// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newrelicoraclereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver"

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/godror/godror"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicoraclereceiver/internal/metadata"
)

// NewFactory creates a new New Relic Oracle receiver factory.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiverFunc(func(dataSourceName string) (*sql.DB, error) {
			return sql.Open("godror", dataSourceName)
		}), metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiverFunc(func(dataSourceName string) (*sql.DB, error) {
			return sql.Open("godror", dataSourceName)
		}), metadata.LogsStability))
}

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = defaultCollectionInterval
	cfg.Timeout = 30 * time.Second // Increased from default to handle Oracle database timeouts

	config := &Config{
		ControllerConfig:      cfg,
		MetricsBuilderConfig:  metadata.DefaultMetricsBuilderConfig(),
		LogsBuilderConfig:     metadata.DefaultLogsBuilderConfig(),
		DisableConnectionPool: false,
	}

	// Apply defaults
	config.SetDefaults()

	return config
}

type sqlOpenerFunc func(dataSourceName string) (*sql.DB, error)

func createMetricsReceiverFunc(sqlOpenerFunc sqlOpenerFunc) receiver.CreateMetricsFunc {
	return func(
		_ context.Context,
		settings receiver.Settings,
		cfg component.Config,
		consumer consumer.Metrics,
	) (receiver.Metrics, error) {
		sqlCfg := cfg.(*Config)

		// Ensure defaults are set and configuration is valid
		sqlCfg.SetDefaults()
		if err := sqlCfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		metricsBuilder := metadata.NewMetricsBuilder(sqlCfg.MetricsBuilderConfig, settings)
		logsBuilder := metadata.NewLogsBuilder(sqlCfg.LogsBuilderConfig, settings)

		instanceName, err := getInstanceName(getDataSource(*sqlCfg))
		if err != nil {
			return nil, err
		}
		hostName, hostNameErr := getHostName(getDataSource(*sqlCfg))
		if hostNameErr != nil {
			return nil, hostNameErr
		}

		mp, err := newScraper(metricsBuilder, sqlCfg.MetricsBuilderConfig, logsBuilder, sqlCfg.LogsBuilderConfig, sqlCfg.ControllerConfig, sqlCfg, settings.Logger, func() (*sql.DB, error) {
			db, err := sqlOpenerFunc(getDataSource(*sqlCfg))
			if err != nil {
				return nil, err
			}

			// Configure connection pool settings
			if !sqlCfg.DisableConnectionPool {
				db.SetMaxOpenConns(sqlCfg.MaxOpenConnections)
			} else {
				// Disable connection pooling
				db.SetMaxOpenConns(1)
			}

			// Set connection timeouts to ensure proper cancellation
			// MaxIdleConns should be reasonable to prevent too many idle connections
			db.SetMaxIdleConns(2)
			// ConnMaxLifetime ensures connections are refreshed periodically
			db.SetConnMaxLifetime(10 * time.Minute)
			// ConnMaxIdleTime closes idle connections
			db.SetConnMaxIdleTime(30 * time.Second)

			return db, nil
		}, instanceName, hostName)
		if err != nil {
			return nil, err
		}
		opt := scraperhelper.AddScraper(metadata.Type, mp)

		return scraperhelper.NewMetricsController(
			&sqlCfg.ControllerConfig,
			settings,
			consumer,
			opt,
		)
	}
}

func createLogsReceiverFunc(sqlOpenerFunc sqlOpenerFunc) receiver.CreateLogsFunc {
	return func(
		_ context.Context,
		settings receiver.Settings,
		cfg component.Config,
		consumer consumer.Logs,
	) (receiver.Logs, error) {
		sqlCfg := cfg.(*Config)

		// Ensure defaults are set and configuration is valid
		sqlCfg.SetDefaults()
		if err := sqlCfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		logsBuilder := metadata.NewLogsBuilder(sqlCfg.LogsBuilderConfig, settings)

		instanceName, err := getInstanceName(getDataSource(*sqlCfg))
		if err != nil {
			return nil, err
		}
		hostName, hostNameErr := getHostName(getDataSource(*sqlCfg))
		if hostNameErr != nil {
			return nil, hostNameErr
		}

		lp, err := newLogsScraper(logsBuilder, sqlCfg.LogsBuilderConfig, sqlCfg.ControllerConfig, sqlCfg, settings.Logger, func() (*sql.DB, error) {
			db, err := sqlOpenerFunc(getDataSource(*sqlCfg))
			if err != nil {
				return nil, err
			}

			// Configure connection pool settings
			if !sqlCfg.DisableConnectionPool {
				db.SetMaxOpenConns(sqlCfg.MaxOpenConnections)
			} else {
				// Disable connection pooling
				db.SetMaxOpenConns(1)
			}

			// Set connection timeouts to ensure proper cancellation
			db.SetMaxIdleConns(2)
			db.SetConnMaxLifetime(10 * time.Minute)
			db.SetConnMaxIdleTime(30 * time.Second)

			return db, nil
		}, instanceName, hostName)
		if err != nil {
			return nil, err
		}

		f := scraper.NewFactory(metadata.Type, nil,
			scraper.WithLogs(func(context.Context, scraper.Settings, component.Config) (scraper.Logs, error) {
				return lp, nil
			}, metadata.LogsStability))
		opt := scraperhelper.AddFactoryWithConfig(f, nil)

		return scraperhelper.NewLogsController(
			&sqlCfg.ControllerConfig,
			settings,
			consumer,
			opt,
		)
	}
}

func getDataSource(cfg Config) string {
	if cfg.DataSource != "" {
		return cfg.DataSource
	}

	// Build godror connection string format
	// Format: user/password@host:port/service_name
	host, portStr, _ := net.SplitHostPort(cfg.Endpoint)
	port, _ := strconv.ParseInt(portStr, 10, 32)

	return fmt.Sprintf("%s/%s@%s:%d/%s", cfg.Username, cfg.Password, host, port, cfg.Service)
}

func getInstanceName(datasource string) (string, error) {
	// For godror format: user/password@host:port/service_name
	// Extract the part after @
	if atIndex := strings.Index(datasource, "@"); atIndex != -1 {
		return datasource[atIndex+1:], nil
	}

	// Fallback to URL parsing for oracle:// format
	datasourceURL, err := url.Parse(datasource)
	if err != nil {
		return "", err
	}

	instanceName := datasourceURL.Host + datasourceURL.Path
	return instanceName, nil
}

func getHostName(datasource string) (string, error) {
	// For godror format: user/password@host:port/service_name
	// Extract the host:port part
	if atIndex := strings.Index(datasource, "@"); atIndex != -1 {
		hostPart := datasource[atIndex+1:]
		if slashIndex := strings.Index(hostPart, "/"); slashIndex != -1 {
			return hostPart[:slashIndex], nil
		}
		return hostPart, nil
	}

	// Fallback to URL parsing for oracle:// format
	datasourceURL, err := url.Parse(datasource)
	if err != nil {
		return "", err
	}
	return datasourceURL.Host, nil
}
