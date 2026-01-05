package retry

import (
	"context"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
)

func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	transientPatterns := []string{
		"timeout",
		"deadline exceeded",
		"broken pipe",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

type BackoffConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

func DoWithBackoff(ctx context.Context, cfg BackoffConfig, fn func() error) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = cfg.InitialBackoff
	expBackoff.MaxInterval = cfg.MaxBackoff
	expBackoff.Multiplier = cfg.Multiplier
	expBackoff.Reset()

	operation := func() error {
		err := fn()
		if err == nil {
			return nil
		}
		if !IsTransientError(err) {
			return backoff.Permanent(err)
		}
		return err
	}

	backoffWithCtx := backoff.WithContext(
		backoff.WithMaxRetries(expBackoff, uint64(cfg.MaxRetries)),
		ctx,
	)

	notify := func(err error, d time.Duration) {
		log.Debug().Err(err).Dur("delay", d).Msg("retrying operation due to transient error")
	}

	return backoff.RetryNotify(
		operation,
		backoffWithCtx,
		notify,
	)
}
