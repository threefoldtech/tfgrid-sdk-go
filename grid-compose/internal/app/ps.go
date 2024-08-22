package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Ps lists deployments on the grid with the option to show more details
func (a *App) Ps(ctx context.Context, verbose bool) error {
	var output strings.Builder
	outputMap := make(map[string]gridtypes.Deployment)

	if !verbose {
		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n",
			"Deployment Name", "Node ID", "Network", "Services", "Storage", "State", "IP Address"))
		output.WriteString(strings.Repeat("-", 150) + "\n")
	}

	if len(a.Config.Networks) == 0 {
		a.Config.Networks[a.GenerateDefaultNetworkName()] = types.Network{}
	}

	for networkName := range a.Config.Networks {
		projectName := a.GetProjectName(networkName)

		if err := a.loadCurrentNodeDeployments(projectName); err != nil {
			return err
		}

		contracts, err := a.Client.ContractsGetter.ListContractsOfProjectName(projectName)
		if err != nil {
			return err
		}

		for _, contract := range contracts.NodeContracts {
			nodeClient, err := a.Client.State.NcPool.GetNodeClient(a.Client.SubstrateConn, contract.NodeID)
			if err != nil {
				return err
			}

			contId, _ := strconv.ParseUint(contract.ContractID, 10, 64)
			dl, err := nodeClient.DeploymentGet(ctx, contId)
			if err != nil {
				return err
			}

			dlAdded := false
			for _, wl := range dl.Workloads {
				if wl.Type.String() != "zmachine" {
					continue
				}

				vm, err := workloads.NewVMFromWorkload(&wl, &dl)
				if err != nil {
					return err
				}

				log.WriteVmDetails(&output, vm, wl, a.GetDeploymentName(wl.Name.String()), contract.NodeID, dlAdded, getVmAddresses(vm))
				dlAdded = true
			}

			dlData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return err
			}

			if verbose {
				outputMap[dlData.Name] = dl
			}
		}
	}

	if verbose {
		out, err := json.MarshalIndent(outputMap, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	}

	// print for better formatting
	fmt.Printf("\n%s\n", output.String())
	return nil
}
