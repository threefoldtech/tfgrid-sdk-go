package probe

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/deployment-checker/pkg/config"
	"github.com/threefoldtech/deployment-checker/pkg/grid"
	"github.com/threefoldtech/deployment-checker/pkg/models"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Executor handles individual deployment execution
type Executor struct {
	gridClient *grid.Client
	cfg        *config.Config
}

// NewExecutor creates a new deployment executor
func NewExecutor(gridClient *grid.Client, cfg *config.Config) *Executor {
	return &Executor{
		gridClient: gridClient,
		cfg:        cfg,
	}
}

// ExecuteDeployment executes a single deployment to a node
func (e *Executor) ExecuteDeployment(ctx context.Context, node types.Node, jitterMin, jitterMax int) (*models.Attempt, error) {
	// Apply jitter before deployment
	if jitterMax > jitterMin {
		jitterRange := jitterMax - jitterMin
		jitterSeconds, err := rand.Int(rand.Reader, big.NewInt(int64(jitterRange+1)))
		if err != nil {
			log.Warn().
				Err(err).
				Int("node_id", node.NodeID).
				Msg("Failed to generate jitter, using minimum")
			jitterSeconds = big.NewInt(0)
		}
		jitterDuration := time.Duration(jitterMin+int(jitterSeconds.Int64())) * time.Second
		log.Debug().
			Int("node_id", node.NodeID).
			Dur("jitter", jitterDuration).
			Msg("Applying jitter before deployment")
		time.Sleep(jitterDuration)
	}

	// Check if node is zoslight
	isZoslight := grid.IsZoslightNode(node)

	log.Debug().
		Int("node_id", node.NodeID).
		Int("farm_id", node.FarmID).
		Str("workload", e.cfg.Probe.WorkloadSize).
		Bool("zoslight", isZoslight).
		Msg("Deploying VM to node")

	cpu, memoryMB, diskMB := e.cfg.GetWorkload()
	timeoutCtx, cancel := context.WithTimeout(ctx, e.cfg.Timeout())
	defer cancel()

	var result *grid.DeploymentResult
	var err error

	if isZoslight {
		result, err = e.gridClient.DeployVMLight(timeoutCtx, uint32(int(node.NodeID)), cpu, memoryMB, diskMB)
	} else {
		result, err = e.gridClient.DeployVM(timeoutCtx, uint32(int(node.NodeID)), cpu, memoryMB, diskMB)
	}

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

	return &attempt, nil
}
