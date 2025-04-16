package deployer

import (
	"time"

	"github.com/getsentry/sentry-go"
)

type gridSentry struct {
	twinID uint32
}

func initSentry(twinID uint32, network string) (gridSentry, error) {
	// Flush buffered events before the program terminates.
	defer sentry.Flush(5 * time.Second)

	return gridSentry{
			twinID: twinID,
		}, sentry.Init(sentry.ClientOptions{
			Dsn:         SentryDSN[network],
			Environment: network,
			Debug:       true,
			// Set TracesSampleRate to 1.0 to capture 100%
			// of transactions for performance monitoring.
			// We recommend adjusting this value in production,
			TracesSampleRate: 1.0,
		})
}

func (s *gridSentry) error(err error) error {
	if s.twinID != 0 {
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetContext("user", map[string]interface{}{
				"twin": s.twinID,
			})
		})
		sentry.CaptureException(err)
	}

	return err
}
