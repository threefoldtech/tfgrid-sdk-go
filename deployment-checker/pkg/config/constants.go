package config

import "time"

const (
	// Workload constants
	WorkloadLightCPU    = 1
	WorkloadLightMemory = 1
	WorkloadLightDisk   = 10

	WorkloadMediumCPU    = 2
	WorkloadMediumMemory = 4
	WorkloadMediumDisk   = 50

	WorkloadHeavyCPU    = 4
	WorkloadHeavyMemory = 8
	WorkloadHeavyDisk   = 100

	// Timeouts
	DefaultProxyTimeout      = 5 * time.Minute
	DefaultDeploymentTimeout = 10 * time.Minute
	DefaultShutdownTimeout   = 30 * time.Second

	// Retries
	DefaultRetryMaxRetries = 3
	DefaultRetryMultiplier = 2.0
	DefaultInitialBackoff  = 1 * time.Second
	DefaultMaxBackoff      = 30 * time.Second

	// Batching
	DefaultBatchSize = 100
	MinBatchSize     = 1
	MaxBatchSize     = 10000

	// Windows
	DefaultScoreWindow = 90 * 24 * time.Hour
	MinWindow          = 1 * time.Hour
	MaxWindow          = 365 * 24 * time.Hour

	// Proxy
	DefaultProxyPageSize = 100
	ProxyRetryMaxRetries = 3
)
