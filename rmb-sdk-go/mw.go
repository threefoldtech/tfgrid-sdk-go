package rmb

import (
	"context"

	"github.com/rs/zerolog/log"
)

// LoggerMiddleware simple logger middleware.
func LoggerMiddleware(ctx context.Context, payload []byte) (context.Context, error) {
	msg := GetRequest(ctx)
	log.Debug().
		Str("twin", msg.TwinSrc).
		Str("fn", msg.Command).
		Int("body-size", len(payload)).Msg("call")
	return ctx, nil
}

var (
	_ Middleware = LoggerMiddleware
)
