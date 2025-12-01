package config

import (
	"fmt"
	"regexp"
	"strconv"
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
	IntervalStr    string      `mapstructure:"interval"`
	Workers        int         `mapstructure:"workers"`
	TimeoutStr     string      `mapstructure:"timeout"`
	LogLevel       string      `mapstructure:"log_level"`
	Grid           GridConfig  `mapstructure:"grid"`
	Nodes          NodesConfig `mapstructure:"nodes"`
	Workload       string      `mapstructure:"workload"`
	ScoreWindowStr string      `mapstructure:"score_window"`
	TimescaleDB    TimescaleDB `mapstructure:"timescaledb"`

	interval    time.Duration
	timeout     time.Duration
	scoreWindow time.Duration
}

type GridConfig struct {
	Network  string `mapstructure:"network"`
	Mnemonic string `mapstructure:"mnemonic"`
}

type NodesConfig struct {
	Status  []string `mapstructure:"status"`
	Farms   []int    `mapstructure:"farms"`
	Nodes   []int    `mapstructure:"nodes"`
	Exclude []int    `mapstructure:"exclude"`
}

type TimescaleDB struct {
	URL string `mapstructure:"url"`
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

	if cfg.IntervalStr != "" {
		d, err := parseDuration(cfg.IntervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid interval format: %w", err)
		}
		cfg.interval = d
	}

	if cfg.TimeoutStr != "" {
		d, err := parseDuration(cfg.TimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %w", err)
		}
		cfg.timeout = d
	}

	if cfg.ScoreWindowStr != "" {
		d, err := parseDuration(cfg.ScoreWindowStr)
		if err != nil {
			return nil, fmt.Errorf("invalid score_window format: %w", err)
		}
		cfg.scoreWindow = d
	}

	if cfg.Workload == "" {
		cfg.Workload = "light"
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func parseDuration(s string) (time.Duration, error) {
	re := regexp.MustCompile(`(\d+)d`)
	if re.MatchString(s) {
		matches := re.FindStringSubmatch(s)
		if len(matches) == 2 {
			days, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, fmt.Errorf("invalid days value: %s", matches[1])
			}
			hours := days * 24
			s = re.ReplaceAllString(s, fmt.Sprintf("%dh", hours))
		}
	}
	return time.ParseDuration(s)
}

func (c *Config) GetWorkload() (cpu uint8, memoryMB uint64, diskMB uint64) {
	switch c.Workload {
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
		return fmt.Errorf("interval must be positive")
	}
	if c.Workers <= 0 {
		return fmt.Errorf("workers must be positive")
	}
	if c.timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.Grid.Network == "" {
		return fmt.Errorf("grid.network is required")
	}
	if c.Grid.Mnemonic == "" {
		return fmt.Errorf("grid.mnemonic is required")
	}
	if c.TimescaleDB.URL == "" {
		return fmt.Errorf("timescaledb.url is required")
	}
	if c.Workload != "light" && c.Workload != "medium" && c.Workload != "heavy" {
		return fmt.Errorf("workload must be light, medium, or heavy")
	}
	return nil
}
