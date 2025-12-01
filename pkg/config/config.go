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
	LogLevel    string        `mapstructure:"log_level"`
	Probe       ProbeConfig   `mapstructure:"probe"`
	Scoring     ScoringConfig `mapstructure:"scoring"`
	Grid        GridConfig    `mapstructure:"grid"`
	Nodes       NodesConfig   `mapstructure:"nodes"`
	TimescaleDB TimescaleDB   `mapstructure:"timescaledb"`
	API         APIConfig     `mapstructure:"api"`

	// viper does not parse duration directly
	interval    time.Duration
	timeout     time.Duration
	scoreWindow time.Duration
}

type ProbeConfig struct {
	IntervalStr      string `mapstructure:"interval"`
	ConcurrencyLimit int    `mapstructure:"concurrency_limit"`
	TimeoutStr       string `mapstructure:"timeout"`
	WorkloadSize     string `mapstructure:"workload_size"`
}

type ScoringConfig struct {
	WindowStr string `mapstructure:"window"`
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

type APIConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
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
	return nil
}
