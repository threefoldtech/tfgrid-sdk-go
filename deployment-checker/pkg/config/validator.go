package config

import "fmt"

// validate validates the configuration
func validate(c *Config) error {
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
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	validWorkloads := map[string]struct{}{"light": {}, "medium": {}, "heavy": {}}
	if _, ok := validWorkloads[c.Probe.WorkloadSize]; !ok {
		return fmt.Errorf("probe.workload_size must be light, medium, or heavy")
	}
	validStatuses := map[string]struct{}{"up": {}, "healthy": {}}
	if _, ok := validStatuses[c.Nodes.Status]; !ok {
		return fmt.Errorf("nodes.status must be up or healthy")
	}

	if c.Probe.JitterMinSeconds < 0 {
		return fmt.Errorf("probe.jitter_min_seconds must be non-negative")
	}
	if c.Probe.JitterMaxSeconds < 0 {
		return fmt.Errorf("probe.jitter_max_seconds must be non-negative")
	}
	if c.Probe.JitterMinSeconds > c.Probe.JitterMaxSeconds {
		return fmt.Errorf("probe.jitter_min_seconds (%d) must be less than or equal to probe.jitter_max_seconds (%d)", c.Probe.JitterMinSeconds, c.Probe.JitterMaxSeconds)
	}

	if c.Probe.BatchSize <= 0 {
		return fmt.Errorf("probe.batch_size must be positive, got %d", c.Probe.BatchSize)
	}
	if c.Probe.BatchSize > MaxBatchSize {
		return fmt.Errorf("probe.batch_size exceeds maximum of %d, got %d", MaxBatchSize, c.Probe.BatchSize)
	}

	if err := validateScorerWeights(c); err != nil {
		return err
	}

	return nil
}

// validateScorerWeights validates scorer weight configuration
func validateScorerWeights(c *Config) error {
	totalWeight := 0.0
	enabledScorers := 0

	if c.Scoring.Scorers.Deployment.Enabled {
		if c.Scoring.Scorers.Deployment.Weight < 0 {
			return fmt.Errorf("deployment scorer weight must be non-negative, got %f", c.Scoring.Scorers.Deployment.Weight)
		}
		totalWeight += c.Scoring.Scorers.Deployment.Weight
		enabledScorers++
	}

	if c.Scoring.Scorers.Duration.Enabled {
		if c.Scoring.Scorers.Duration.Weight < 0 {
			return fmt.Errorf("duration scorer weight must be non-negative, got %f", c.Scoring.Scorers.Duration.Weight)
		}
		totalWeight += c.Scoring.Scorers.Duration.Weight
		enabledScorers++
	}

	if c.Scoring.Scorers.Uptime.Enabled {
		if c.Scoring.Scorers.Uptime.Weight < 0 {
			return fmt.Errorf("uptime scorer weight must be non-negative, got %f", c.Scoring.Scorers.Uptime.Weight)
		}
		totalWeight += c.Scoring.Scorers.Uptime.Weight
		enabledScorers++
	}

	if enabledScorers > 0 && totalWeight == 0 {
		return fmt.Errorf("at least one enabled scorer must have positive weight")
	}

	return nil
}
