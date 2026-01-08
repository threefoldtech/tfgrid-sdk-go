package probe

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/grid"
	"github.com/threefoldtech/deployment-checker/pkg/models"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// calculateJitter calculates a random jitter duration between jitterMin and jitterMax seconds
func calculateJitter(jitterMin, jitterMax int, nodeID int) time.Duration {
	if jitterMax <= jitterMin {
		return 0
	}

	jitterRange := jitterMax - jitterMin
	jitterSeconds, err := rand.Int(rand.Reader, big.NewInt(int64(jitterRange+1)))
	if err != nil {
		log.Warn().
			Err(err).
			Int("node_id", nodeID).
			Msg("Failed to generate jitter, using minimum")
		jitterSeconds = big.NewInt(0)
	}

	duration := time.Duration(jitterMin+int(jitterSeconds.Int64())) * time.Second
	log.Debug().
		Int("node_id", nodeID).
		Dur("jitter", duration).
		Msg("Applying jitter before deployment")

	return duration
}

// mapResultToAttempt converts a deployment result to an Attempt model
func mapResultToAttempt(result *grid.DeploymentResult, node types.Node, err error) *models.Attempt {
	attempt := models.Attempt{
		Time:   time.Now().Unix(),
		NodeID: int64(int(node.NodeID)),
		FarmID: int64(int(node.FarmID)),
		Status: "failed",
	}

	if err != nil {
		errorCode := "unknown_error"
		if result != nil && result.ErrorCode != "" {
			errorCode = result.ErrorCode
			attempt.TotalDurationMs = &result.TotalDurationMs
		}
		attempt.ErrorCode = &errorCode
		log.Debug().
			Err(err).
			Int("node_id", node.NodeID).
			Str("error_code", errorCode).
			Msg("Deployment failed")
	} else if result != nil {
		attempt.Status = "success"
		attempt.TotalDurationMs = &result.TotalDurationMs
		log.Debug().
			Int("node_id", node.NodeID).
			Msg("Deployment succeeded")
	}

	return &attempt
}
