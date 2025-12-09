// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newrelicsqlserverreceiver

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`

	// Connection configuration
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Hostname string `mapstructure:"hostname"`
	Port     string `mapstructure:"port"`
	Instance string `mapstructure:"instance"`

	// Azure AD authentication
	ClientID     string `mapstructure:"client_id"`
	TenantID     string `mapstructure:"tenant_id"`
	ClientSecret string `mapstructure:"client_secret"`

	// SSL configuration
	EnableSSL              bool   `mapstructure:"enable_ssl"`
	TrustServerCertificate bool   `mapstructure:"trust_server_certificate"`
	CertificateLocation    string `mapstructure:"certificate_location"`

	// Concurrency and timeouts
	MaxConcurrentWorkers int           `mapstructure:"max_concurrent_workers"`
	Timeout              time.Duration `mapstructure:"timeout"`

	// Custom queries
	CustomMetricsQuery  string `mapstructure:"custom_metrics_query"`
	CustomMetricsConfig string `mapstructure:"custom_metrics_config"`

	// Additional connection parameters
	ExtraConnectionURLArgs string `mapstructure:"extra_connection_url_args"`

	// Query monitoring configuration
	EnableQueryMonitoring                bool `mapstructure:"enable_query_monitoring"`
	QueryMonitoringResponseTimeThreshold int  `mapstructure:"query_monitoring_response_time_threshold"`
	QueryMonitoringCountThreshold        int  `mapstructure:"query_monitoring_count_threshold"`
	QueryMonitoringFetchInterval         int  `mapstructure:"query_monitoring_fetch_interval"`
	QueryMonitoringTextTruncateLimit     int  `mapstructure:"query_monitoring_text_truncate_limit"`

	// Active running queries configuration
	EnableActiveRunningQueries               bool `mapstructure:"enable_active_running_queries"`
	ActiveRunningQueriesElapsedTimeThreshold int  `mapstructure:"active_running_queries_elapsed_time_threshold"` // Minimum elapsed time in milliseconds (default: 0 = capture all)

	// Slow query smoothing configuration (EWMA-based smoothing)
	EnableSlowQuerySmoothing         bool    `mapstructure:"enable_slow_query_smoothing"`          // Enable/disable EWMA smoothing algorithm
	SlowQuerySmoothingFactor         float64 `mapstructure:"slow_query_smoothing_factor"`          // Weight for new data (0.0-1.0, default: 0.3)
	SlowQuerySmoothingDecayThreshold int     `mapstructure:"slow_query_smoothing_decay_threshold"` // Consecutive misses before removal (default: 3)
	SlowQuerySmoothingMaxAgeMinutes  int     `mapstructure:"slow_query_smoothing_max_age_minutes"` // Maximum age in minutes (default: 5)

	// Interval-based averaging configuration (Simplified delta-based interval calculations)
	// This addresses the problem of cumulative averages masking recent query optimizations
	EnableIntervalBasedAveraging      bool `mapstructure:"enable_interval_based_averaging"`       // Enable/disable interval-based averaging
	IntervalCalculatorCacheTTLMinutes int  `mapstructure:"interval_calculator_cache_ttl_minutes"` // State cache TTL in minutes (default: 10)

	// Wait resource enrichment configuration
	// Enriches wait_resource with human-readable names (database names, object names, file names, etc.)
	EnableWaitResourceEnrichment       bool `mapstructure:"enable_wait_resource_enrichment"`        // Enable/disable wait_resource name enrichment
	WaitResourceMetadataRefreshMinutes int  `mapstructure:"wait_resource_metadata_refresh_minutes"` // Metadata cache refresh interval in minutes (default: 5)
}

// DefaultConfig returns a Config struct with default values
func DefaultConfig() component.Config {
	cfg := &Config{
		ControllerConfig: scraperhelper.NewDefaultControllerConfig(),

		// Default connection settings
		Hostname: "127.0.0.1",
		Port:     "1433",

		// Default concurrency and timeout
		MaxConcurrentWorkers: 10,
		Timeout:              30 * time.Second,

		// Default SSL settings
		EnableSSL:              false,
		TrustServerCertificate: false,

		// Default query monitoring settings
		EnableQueryMonitoring:                true,
		QueryMonitoringResponseTimeThreshold: 0, // 0 = capture all queries (no threshold)
		QueryMonitoringCountThreshold:        20,
		QueryMonitoringFetchInterval:         15,
		QueryMonitoringTextTruncateLimit:     4094, // Default text truncate limit (4KB - 2 bytes for null terminator)

		// Default active running queries settings
		EnableActiveRunningQueries:               true, // Enable by default for comprehensive query monitoring
		ActiveRunningQueriesElapsedTimeThreshold: 0,    // Default: 0ms (capture all active queries, including very short ones)

		// Default slow query smoothing settings (EWMA-based)
		EnableSlowQuerySmoothing:         false, // Disabled - using delta calculation only
		SlowQuerySmoothingFactor:         0.3,   // 30% new data, 70% historical data
		SlowQuerySmoothingDecayThreshold: 3,     // Remove after 3 consecutive misses
		SlowQuerySmoothingMaxAgeMinutes:  5,     // Maximum age of 5 minutes

		// Default interval-based averaging settings (Simplified delta-based)
		EnableIntervalBasedAveraging:      true, // Enabled by default for delta-based averaging
		IntervalCalculatorCacheTTLMinutes: 10,   // 10 minute TTL for state cache

		// Default wait resource enrichment settings
		EnableWaitResourceEnrichment:       true, // Enabled by default for human-readable wait_resource
		WaitResourceMetadataRefreshMinutes: 5,    // Refresh metadata cache every 5 minutes
	}

	// Set default collection interval to 15 seconds
	cfg.ControllerConfig.CollectionInterval = 15 * time.Second

	return cfg
}

// Validate validates the configuration and sets defaults where needed
func (cfg *Config) Validate() error {
	if cfg.Hostname == "" {
		return errors.New("hostname cannot be empty")
	}

	if cfg.Port != "" && cfg.Instance != "" {
		return errors.New("specify either port or instance but not both")
	} else if cfg.Port == "" && cfg.Instance == "" {
		// Default to port 1433 if neither is specified
		cfg.Port = "1433"
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	if cfg.MaxConcurrentWorkers <= 0 {
		return errors.New("max_concurrent_workers must be positive")
	}

	if cfg.EnableQueryMonitoring {
		if cfg.QueryMonitoringResponseTimeThreshold < 0 {
			return errors.New("query_monitoring_response_time_threshold must be >= 0 when query monitoring is enabled (0 = no threshold)")
		}
		if cfg.QueryMonitoringCountThreshold <= 0 {
			return errors.New("query_monitoring_count_threshold must be positive when query monitoring is enabled")
		}
		if cfg.QueryMonitoringTextTruncateLimit <= 0 {
			return errors.New("query_monitoring_text_truncate_limit must be positive when query monitoring is enabled")
		}
	}

	if cfg.EnableSSL && (!cfg.TrustServerCertificate && cfg.CertificateLocation == "") {
		return errors.New("must specify a certificate file when using SSL and not trusting server certificate")
	}

	if len(cfg.CustomMetricsConfig) > 0 {
		if len(cfg.CustomMetricsQuery) > 0 {
			return errors.New("cannot specify both custom_metrics_query and custom_metrics_config")
		}
		if _, err := os.Stat(cfg.CustomMetricsConfig); err != nil {
			return fmt.Errorf("custom_metrics_config file error: %w", err)
		}
	}

	return nil
}

// Unmarshal implements the confmap.Unmarshaler interface
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// GetMaxConcurrentWorkers returns the configured max concurrent workers with fallback
func (cfg *Config) GetMaxConcurrentWorkers() int {
	if cfg.MaxConcurrentWorkers <= 0 {
		return 10 // Default max concurrent workers
	}
	return cfg.MaxConcurrentWorkers
}

// IsAzureADAuth checks if Azure AD Service Principal authentication is configured
func (cfg *Config) IsAzureADAuth() bool {
	return cfg.ClientID != "" && cfg.TenantID != "" && cfg.ClientSecret != ""
}

// CreateConnectionURL creates a connection string for SQL Server authentication
func (cfg *Config) CreateConnectionURL(dbName string) string {
	// Use ADO.NET connection string format instead of URL format
	// This avoids URL encoding issues with special characters in passwords
	connStr := fmt.Sprintf("server=%s", cfg.Hostname)

	// Add port or instance
	if cfg.Port != "" {
		connStr += fmt.Sprintf(";port=%s", cfg.Port)
	} else if cfg.Instance != "" {
		connStr += fmt.Sprintf("\\%s", cfg.Instance)
	}

	// Add authentication
	if cfg.Username != "" && cfg.Password != "" {
		connStr += fmt.Sprintf(";user id=%s;password=%s", cfg.Username, cfg.Password)
	}

	// Add database
	if dbName != "" {
		connStr += fmt.Sprintf(";database=%s", dbName)
	} else {
		connStr += ";database=master"
	}

	// Add timeouts
	connStr += fmt.Sprintf(";dial timeout=%.0f;connection timeout=%.0f",
		cfg.Timeout.Seconds(), cfg.Timeout.Seconds())

	// Add SSL settings
	if cfg.EnableSSL {
		connStr += ";encrypt=true"
		if cfg.TrustServerCertificate {
			connStr += ";TrustServerCertificate=true"
		}
		if !cfg.TrustServerCertificate && cfg.CertificateLocation != "" {
			connStr += fmt.Sprintf(";certificate=%s", cfg.CertificateLocation)
		}
	} else {
		// Explicitly disable encryption when SSL is not enabled
		connStr += ";encrypt=disable"
	}

	// Add extra connection args
	if cfg.ExtraConnectionURLArgs != "" {
		extraArgsMap, err := url.ParseQuery(cfg.ExtraConnectionURLArgs)
		if err == nil {
			for k, v := range extraArgsMap {
				connStr += fmt.Sprintf(";%s=%s", k, v[0])
			}
		}
	}

	return connStr
}

// CreateAzureADConnectionURL creates a connection string for Azure AD authentication
func (cfg *Config) CreateAzureADConnectionURL(dbName string) string {
	connectionString := fmt.Sprintf(
		"server=%s;port=%s;fedauth=ActiveDirectoryServicePrincipal;applicationclientid=%s;clientsecret=%s;database=%s",
		cfg.Hostname,
		cfg.Port,
		cfg.ClientID,     // Client ID
		cfg.ClientSecret, // Client Secret
		dbName,           // Database
	)

	if cfg.ExtraConnectionURLArgs != "" {
		extraArgsMap, err := url.ParseQuery(cfg.ExtraConnectionURLArgs)
		if err == nil {
			for k, v := range extraArgsMap {
				connectionString += fmt.Sprintf(";%s=%s", k, v[0])
			}
		}
	}

	if cfg.EnableSSL {
		connectionString += ";encrypt=true"
		if cfg.TrustServerCertificate {
			connectionString += ";TrustServerCertificate=true"
		} else {
			connectionString += ";TrustServerCertificate=false"
			if cfg.CertificateLocation != "" {
				connectionString += fmt.Sprintf(";certificate=%s", cfg.CertificateLocation)
			}
		}
	}

	return connectionString
}
