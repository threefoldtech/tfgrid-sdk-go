package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "list containers",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ps(cmd.Context()); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func ps(ctx context.Context) error {
	twindId := uint64(app.Client.TwinID)
	filters := types.ContractFilter{
		TwinID: &twindId,
	}
	limits := types.Limit{
		Size: 100,
	}
	cache := make(map[string]bool, 0)

	contracts, _, err := app.Client.GridProxyClient.Contracts(ctx, filters, limits)

	if err != nil {
		return err
	}

	for _, contract := range contracts {
		if contract.Type != "node" || contract.State == "Deleted" {
			continue
		}

		details, err := workloads.ParseDeploymentData(contract.Details.(types.NodeContractDetails).DeploymentData)
		if err != nil {
			return err
		}

		if strings.Split(details.ProjectName, "/")[0] != "compose" || cache[details.ProjectName] {
			continue
		}

		res, err := GetVM(ctx, app.Client, details.Name)
		if err != nil {
			return err
		}

		s, err := json.MarshalIndent(res, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msg("vm:\n" + string(s))
		cache[details.ProjectName] = true
	}

	return nil
}

// GetVM gets a vm with its project name
func GetVM(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.Deployment, error) {
	name, _ = strings.CutSuffix(name, "net")
	projectName := internal.GetProjectName(name, app.Client.TwinID)

	// try to get contracts with the old project name format "<name>"
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(projectName, true)
	if err != nil {
		return workloads.Deployment{}, err
	}

	if len(contracts.NodeContracts) == 0 {
		// if could not find any contracts try to get contracts with the new project name format "vm/<name>"
		projectName = fmt.Sprintf("vm/%s", name)
		contracts, err = t.ContractsGetter.ListContractsOfProjectName(projectName, true)
		if err != nil {
			return workloads.Deployment{}, err
		}

		if len(contracts.NodeContracts) == 0 {
			return workloads.Deployment{}, fmt.Errorf("couldn't find any contracts with name %s", name)
		}
	}

	var nodeID uint32

	for _, contract := range contracts.NodeContracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return workloads.Deployment{}, err
		}

		nodeID = contract.NodeID
		checkIfExistAndAppend(t, nodeID, contractID)

	}

	return t.State.LoadDeploymentFromGrid(ctx, nodeID, name)
}

func checkIfExistAndAppend(t deployer.TFPluginClient, node uint32, contractID uint64) {
	for _, n := range t.State.CurrentNodeDeployments[node] {
		if n == contractID {
			return
		}
	}

	t.State.CurrentNodeDeployments[node] = append(t.State.CurrentNodeDeployments[node], contractID)
}
