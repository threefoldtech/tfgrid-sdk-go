package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
