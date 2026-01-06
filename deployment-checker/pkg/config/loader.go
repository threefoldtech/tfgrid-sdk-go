package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load loads and parses the configuration file
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse durations
	if err := parseDurations(&cfg); err != nil {
		return nil, err
	}

	// Set default values
	setDefaults(&cfg)

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// parseDurations parses all duration strings in the config
func parseDurations(cfg *Config) error {
	if cfg.Probe.IntervalStr != "" {
		d, err := ParseDuration(cfg.Probe.IntervalStr)
		if err != nil {
			return fmt.Errorf("invalid interval format: %w", err)
		}
		cfg.interval = d
	}

	if cfg.Probe.TimeoutStr != "" {
		d, err := ParseDuration(cfg.Probe.TimeoutStr)
		if err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
		cfg.timeout = d
	}

	if cfg.Scoring.WindowStr != "" {
		d, err := ParseDuration(cfg.Scoring.WindowStr)
		if err != nil {
			return fmt.Errorf("invalid scoring window format: %w", err)
		}
		cfg.scoreWindow = d
	}

	if cfg.Probe.ShutdownTimeoutStr != "" {
		d, err := ParseDuration(cfg.Probe.ShutdownTimeoutStr)
		if err != nil {
			return fmt.Errorf("invalid shutdown_timeout format: %w", err)
		}
		cfg.shutdownTimeout = d
	}

	if cfg.Probe.Retry.InitialBackoffStr != "" {
		d, err := ParseDuration(cfg.Probe.Retry.InitialBackoffStr)
		if err != nil {
			return fmt.Errorf("invalid retry.initial_backoff format: %w", err)
		}
		cfg.initialBackoff = d
	}

	if cfg.Probe.Retry.MaxBackoffStr != "" {
		d, err := ParseDuration(cfg.Probe.Retry.MaxBackoffStr)
		if err != nil {
			return fmt.Errorf("invalid retry.max_backoff format: %w", err)
		}
		cfg.maxBackoff = d
	}

	return nil
}

// setDefaults sets default values for configuration fields
func setDefaults(cfg *Config) {
	if cfg.Probe.WorkloadSize == "" {
		cfg.Probe.WorkloadSize = "light"
	}

	if cfg.API.Host == "" {
		cfg.API.Host = "0.0.0.0"
	}

	if cfg.API.Port == 0 {
		cfg.API.Port = 8080
	}

	if cfg.Probe.Retry.MaxRetries == 0 {
		cfg.Probe.Retry.MaxRetries = DefaultRetryMaxRetries
	}

	if cfg.Probe.Retry.Multiplier == 0 {
		cfg.Probe.Retry.Multiplier = DefaultRetryMultiplier
	}

	if cfg.Probe.BatchSize == 0 {
		cfg.Probe.BatchSize = DefaultBatchSize
	}

	// Set default scoring configuration
	if cfg.Scoring.MinAttempts == 0 {
		cfg.Scoring.MinAttempts = 1
	}

	// Set default scorer configurations
	if cfg.Scoring.Scorers.Deployment.Weight == 0 {
		cfg.Scoring.Scorers.Deployment.Weight = 1.0
	}
	if !cfg.Scoring.Scorers.Deployment.Enabled {
		cfg.Scoring.Scorers.Deployment.Enabled = true
	}
	// Duration and uptime default to disabled (weight 0.0)

	if cfg.initialBackoff == 0 {
		cfg.initialBackoff = DefaultInitialBackoff
	}

	if cfg.maxBackoff == 0 {
		cfg.maxBackoff = DefaultMaxBackoff
	}

	if cfg.shutdownTimeout == 0 {
		cfg.shutdownTimeout = DefaultShutdownTimeout
	}

	// Set default jitter values
	if cfg.Probe.JitterMinSeconds == 0 {
		cfg.Probe.JitterMinSeconds = 6
	}
	if cfg.Probe.JitterMaxSeconds == 0 {
		cfg.Probe.JitterMaxSeconds = 10
	}

	// Set default cleanup on startup
	// Note: viper will set this to false if not present, so we check if it was explicitly set
	// For now, default to true if not specified
	if !viper.IsSet("probe.cleanup_on_startup") {
		cfg.Probe.CleanupOnStartup = true
	}

	// Set default retention configuration
	if cfg.Database.RetentionDays == 0 {
		cfg.Database.RetentionDays = 90
	}
	// Default to using TimescaleDB retention policies if not specified
	if !viper.IsSet("database.use_timescaledb_retention") {
		cfg.Database.UseTimescaleDBRetention = true
	}
}
