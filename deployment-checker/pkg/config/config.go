package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	WorkloadLightCPU    = 1
	WorkloadLightMemory = 1
	WorkloadLightDisk   = 10

	WorkloadMediumCPU    = 2
	WorkloadMediumMemory = 4
	WorkloadMediumDisk   = 50

	WorkloadHeavyCPU    = 4
	WorkloadHeavyMemory = 8
	WorkloadHeavyDisk   = 100
)

type Config struct {
	LogLevel    string         `mapstructure:"log_level"`
	Probe       ProbeConfig    `mapstructure:"probe"`
	Scoring     ScoringConfig  `mapstructure:"scoring"`
	Grid        GridConfig     `mapstructure:"grid"`
	Nodes       NodesConfig    `mapstructure:"nodes"`
	TimescaleDB TimescaleDB    `mapstructure:"timescaledb"`
	Database    DatabaseConfig `mapstructure:"database"`
	API         APIConfig      `mapstructure:"api"`

	// viper does not parse duration directly
	interval        time.Duration
	timeout         time.Duration
	scoreWindow     time.Duration
	shutdownTimeout time.Duration
	initialBackoff  time.Duration
	maxBackoff      time.Duration
}

type ProbeConfig struct {
	IntervalStr        string      `mapstructure:"interval"`
	ConcurrencyLimit   int         `mapstructure:"concurrency_limit"`
	TimeoutStr         string      `mapstructure:"timeout"`
	WorkloadSize       string      `mapstructure:"workload_size"`
	Retry              RetryConfig `mapstructure:"retry"`
	ShutdownTimeoutStr string      `mapstructure:"shutdown_timeout"`
	BatchSize          int         `mapstructure:"batch_size"`
	JitterMinSeconds   int         `mapstructure:"jitter_min_seconds"`
	JitterMaxSeconds   int         `mapstructure:"jitter_max_seconds"`
	CleanupOnStartup   bool        `mapstructure:"cleanup_on_startup"`
}

type ScoringConfig struct {
	WindowStr   string        `mapstructure:"window"`
	Scorers     ScorersConfig `mapstructure:"scorers"`
	MinAttempts int           `mapstructure:"min_attempts"`
}

type ScorersConfig struct {
	Deployment ScorerConfig `mapstructure:"deployment"`
	Duration   ScorerConfig `mapstructure:"duration"`
	Uptime     ScorerConfig `mapstructure:"uptime"`
}

type ScorerConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	Weight  float64 `mapstructure:"weight"`
}

type GridConfig struct {
	Network  string `mapstructure:"network"`
	Mnemonic string `mapstructure:"mnemonic"`
}

type NodesConfig struct {
	Status  string   `mapstructure:"status"`
	Farms   []uint64 `mapstructure:"farms"`
	Nodes   []uint64 `mapstructure:"nodes"`
	Exclude []uint64 `mapstructure:"exclude"`
}

type TimescaleDB struct {
	URL string `mapstructure:"url"`
}

type DatabaseConfig struct {
	RetentionDays           int  `mapstructure:"retention_days"`
	UseTimescaleDBRetention bool `mapstructure:"use_timescaledb_retention"`
}

type APIConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type RetryConfig struct {
	MaxRetries        int     `mapstructure:"max_retries"`
	InitialBackoffStr string  `mapstructure:"initial_backoff"`
	MaxBackoffStr     string  `mapstructure:"max_backoff"`
	Multiplier        float64 `mapstructure:"multiplier"`
}

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

	// manually parse durations

	if cfg.Probe.IntervalStr != "" {
		d, err := ParseDuration(cfg.Probe.IntervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid interval format: %w", err)
		}
		cfg.interval = d
	}

	if cfg.Probe.TimeoutStr != "" {
		d, err := ParseDuration(cfg.Probe.TimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %w", err)
		}
		cfg.timeout = d
	}

	if cfg.Scoring.WindowStr != "" {
		d, err := ParseDuration(cfg.Scoring.WindowStr)
		if err != nil {
			return nil, fmt.Errorf("invalid scoring window format: %w", err)
		}
		cfg.scoreWindow = d
	}

	if cfg.Probe.ShutdownTimeoutStr != "" {
		d, err := ParseDuration(cfg.Probe.ShutdownTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid shutdown_timeout format: %w", err)
		}
		cfg.shutdownTimeout = d
	}

	if cfg.Probe.Retry.InitialBackoffStr != "" {
		d, err := ParseDuration(cfg.Probe.Retry.InitialBackoffStr)
		if err != nil {
			return nil, fmt.Errorf("invalid retry.initial_backoff format: %w", err)
		}
		cfg.initialBackoff = d
	}

	if cfg.Probe.Retry.MaxBackoffStr != "" {
		d, err := ParseDuration(cfg.Probe.Retry.MaxBackoffStr)
		if err != nil {
			return nil, fmt.Errorf("invalid retry.max_backoff format: %w", err)
		}
		cfg.maxBackoff = d
	}

	// add default values

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
		cfg.Probe.Retry.MaxRetries = 3
	}

	if cfg.Probe.Retry.Multiplier == 0 {
		cfg.Probe.Retry.Multiplier = 2.0
	}

	if cfg.Probe.BatchSize == 0 {
		cfg.Probe.BatchSize = 100
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
		cfg.initialBackoff = 1 * time.Second
	}

	if cfg.maxBackoff == 0 {
		cfg.maxBackoff = 30 * time.Second
	}

	if cfg.shutdownTimeout == 0 {
		cfg.shutdownTimeout = 30 * time.Second
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

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func ParseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", strings.TrimSuffix(s, "d"))
		}
		d := time.Duration(days) * 24 * time.Hour
		return d, nil
	}
	return time.ParseDuration(s)
}

func (c *Config) GetWorkload() (cpu uint8, memoryMB uint64, diskMB uint64) {
	switch c.Probe.WorkloadSize {
	case "medium":
		return WorkloadMediumCPU, uint64(WorkloadMediumMemory * 1024), uint64(WorkloadMediumDisk * 1024)
	case "heavy":
		return WorkloadHeavyCPU, uint64(WorkloadHeavyMemory * 1024), uint64(WorkloadHeavyDisk * 1024)
	default:
		return WorkloadLightCPU, uint64(WorkloadLightMemory * 1024), uint64(WorkloadLightDisk * 1024)
	}
}

func (c *Config) Interval() time.Duration {
	return c.interval
}

func (c *Config) Timeout() time.Duration {
	return c.timeout
}

func (c *Config) ScoreWindow() time.Duration {
	return c.scoreWindow
}

func (c *Config) ShutdownTimeout() time.Duration {
	return c.shutdownTimeout
}

func (c *Config) RetryConfig() RetryConfig {
	return c.Probe.Retry
}

func (c *Config) InitialBackoff() time.Duration {
	return c.initialBackoff
}

func (c *Config) MaxBackoff() time.Duration {
	return c.maxBackoff
}

func (c *Config) BatchSize() int {
	return c.Probe.BatchSize
}

func (c *Config) JitterMinSeconds() int {
	return c.Probe.JitterMinSeconds
}

func (c *Config) JitterMaxSeconds() int {
	return c.Probe.JitterMaxSeconds
}

func (c *Config) CleanupOnStartup() bool {
	return c.Probe.CleanupOnStartup
}

func (c *Config) ScoringConfig() ScoringConfig {
	return c.Scoring
}

func (c *Config) RetentionDays() int {
	return c.Database.RetentionDays
}

func (c *Config) UseTimescaleDBRetention() bool {
	return c.Database.UseTimescaleDBRetention
}

func (c *Config) validate() error {
	if c.interval <= 0 {
		return fmt.Errorf("probe.interval must be positive")
	}
	if c.Probe.ConcurrencyLimit <= 0 {
		return fmt.Errorf("probe.concurrency_limit must be positive")
	}
	if c.timeout <= 0 {
		return fmt.Errorf("probe.timeout must be positive")
	}

	validNetworks := map[string]struct{}{"dev": {}, "qa": {}, "test": {}, "main": {}}
	if _, ok := validNetworks[c.Grid.Network]; !ok {
		return fmt.Errorf("grid.network must be dev, qa, test, or main")
	}

	if c.Grid.Mnemonic == "" {
		return fmt.Errorf("grid.mnemonic is required")
	}
	if c.TimescaleDB.URL == "" {
		return fmt.Errorf("timescaledb.url is required")
	}

	validWorkloads := map[string]struct{}{"light": {}, "medium": {}, "heavy": {}}
	if _, ok := validWorkloads[c.Probe.WorkloadSize]; !ok {
		return fmt.Errorf("probe.workload_size must be light, medium, or heavy")
	}
	validStatuses := map[string]struct{}{"up": {}, "healthy": {}}
	if _, ok := validStatuses[c.Nodes.Status]; !ok {
		return fmt.Errorf("nodes.status must be up or healthy")
	}

	// Validate jitter configuration
	if c.Probe.JitterMinSeconds < 0 {
		return fmt.Errorf("probe.jitter_min_seconds must be non-negative")
	}
	if c.Probe.JitterMaxSeconds < 0 {
		return fmt.Errorf("probe.jitter_max_seconds must be non-negative")
	}
	if c.Probe.JitterMinSeconds > c.Probe.JitterMaxSeconds {
		return fmt.Errorf("probe.jitter_min_seconds (%d) must be less than or equal to probe.jitter_max_seconds (%d)", c.Probe.JitterMinSeconds, c.Probe.JitterMaxSeconds)
	}

	return nil
}
