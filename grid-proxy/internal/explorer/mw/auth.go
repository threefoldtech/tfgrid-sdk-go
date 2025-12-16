package mw

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

const (
	// AuthHeader is the name of the authentication header
	AuthHeader = "X-Auth"
	// ChallengeValidity is the duration a challenge remains valid
	ChallengeValidity = 1 * time.Minute
)

// twinIDKey is the context key for storing authenticated twin ID
type twinIDKey struct{}

// TwinDB interface for getting twin public keys
type TwinDB interface {
	Twins(ctx context.Context, filter types.TwinFilter, limit types.Limit) ([]types.Twin, int, error)
}

// AuthMiddleware creates a middleware that authenticates requests based on X-Auth header
// Header format: <timestamp>:<twin_id>:<signature_base64>
// The signature is over the challenge message (timestamp:twin_id)
func AuthMiddleware(twinDB TwinDB) func(Action) Action {
	return func(next Action) Action {
		return func(r *http.Request) (interface{}, Response) {
			// Extract and validate auth header
			authHeader := r.Header.Get(AuthHeader)
			if authHeader == "" {
				return nil, UnAuthorized(errors.New("X-Auth header required"))
			}

			parts := strings.Split(authHeader, ":")
			if len(parts) != 3 {
				return nil, BadRequest(errors.New("invalid X-Auth header format, expected 'timestamp:twin_id:signature'"))
			}

			timestampStr, twinIDStr, signatureB64 := parts[0], parts[1], parts[2]

			// Validate timestamp
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				log.Debug().Err(err).Msg("invalid timestamp")
				return nil, BadRequest(errors.New("invalid timestamp"))
			}

			if time.Since(time.Unix(timestamp, 0)) > ChallengeValidity {
				return nil, UnAuthorized(errors.New("expired challenge"))
			}

			// Parse twin ID
			twinID, err := strconv.ParseUint(twinIDStr, 10, 64)
			if err != nil {
				log.Debug().Err(err).Msg("invalid twin ID format")
				return nil, BadRequest(errors.New("invalid twin ID format"))
			}

			// Create challenge message
			challenge := fmt.Sprintf("%s:%s", timestampStr, twinIDStr)

			// Get twin's public key from database
			filter := types.TwinFilter{TwinID: &twinID}
			twins, _, err := twinDB.Twins(r.Context(), filter, types.Limit{Size: 1, Page: 1})
			if err != nil {
				log.Debug().Err(err).Uint64("twinID", twinID).Msg("failed to get twin")
				return nil, UnAuthorized(errors.New("twin not found"))
			}
			if len(twins) == 0 {
				return nil, UnAuthorized(errors.New("twin not found"))
			}

			// Decode stored public key
			storedPK, err := base64.StdEncoding.DecodeString(twins[0].PublicKey)
			if err != nil {
				log.Debug().Err(err).Msg("invalid stored public key")
				return nil, BadRequest(errors.New("invalid stored public key"))
			}

			// Decode signature
			sig, err := base64.StdEncoding.DecodeString(signatureB64)
			if err != nil {
				log.Debug().Err(err).Msg("invalid signature encoding")
				return nil, BadRequest(errors.New("invalid signature encoding"))
			}

			// Verify signature (auto-detect ed25519 or sr25519)
			if err := verifySignature(storedPK, []byte(challenge), sig); err != nil {
				log.Debug().Err(err).Msg("signature verification failed")
				return nil, UnAuthorized(err)
			}

			// Store verified twin ID in context
			ctx := context.WithValue(r.Context(), twinIDKey{}, twinID)
			r = r.WithContext(ctx)

			// Call next handler
			return next(r)
		}
	}
}

// GetAuthenticatedTwinID extracts the authenticated twin ID from the request context
// Returns 0 if not authenticated
func GetAuthenticatedTwinID(r *http.Request) uint64 {
	twinID, ok := r.Context().Value(twinIDKey{}).(uint64)
	if !ok {
		return 0
	}
	return twinID
}

// EnsureOwner checks if the authenticated twin ID matches the expected twin ID
// Returns a Response error if not authorized, nil if authorized
func EnsureOwner(r *http.Request, expectedTwinID uint64) Response {
	authTwinID := GetAuthenticatedTwinID(r)
	if authTwinID == 0 {
		return UnAuthorized(errors.New("not authenticated"))
	}
	if authTwinID != expectedTwinID || expectedTwinID == 0 {
		return Forbidden(errors.New("not authorized to access this resource"))
	}
	return nil
}
