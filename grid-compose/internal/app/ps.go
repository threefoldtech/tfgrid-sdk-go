package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/deploy"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Ps lists the deployed services
// TODO: error handling and better output(mounts index out of range)
func (a *App) Ps(ctx context.Context, verbose bool) error {

	var output strings.Builder
	outputMap := make(map[string]gridtypes.Deployment)

	if !verbose {
		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n", "Deployment Name", "Node ID", "Network", "Services", "Storage", "State", "IP Address"))
		output.WriteString(strings.Repeat("-", 150) + "\n")
	}

	if len(a.Config.Networks) == 0 {
		a.Config.Networks[deploy.GenerateDefaultNetworkName(a.Config.Services)] = types.Network{}
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

			for _, wl := range dl.Workloads {
				if wl.Type.String() != "zmachine" {
					continue
				}

				vm, err := workloads.NewVMFromWorkload(&wl, &dl)

				if err != nil {
					return err
				}
				output.WriteString(fmt.Sprintf("%-15s | %-15d | %-15s | %-15s | %-15s | %-10s | %s\n", a.GetDeploymentName(wl.Name.String()), contract.NodeID, vm.NetworkName, vm.Name, vm.Mounts[0].DiskName, wl.Result.State, getVmAddresses(vm)))
			}

			dlData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return err
			}

			if verbose {
				outputMap[dlData.Name] = dl
			}

			// 	if !verbose {
			// 		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n", contractDlData.Name, contract.NodeID, vm.NetworkName, vm.Name, vm.Mounts[0].DiskName, wl.Result.State, getVmAddresses(vm)))
			// 		// 	output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-10s | %s\n", strings.Repeat("-", 15), strings.Repeat("-", 15), vm.Name, vm.Mounts[0].DiskName, wl.Result.State, getVmAddresses(vm)))
			// 	}
			// }

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

func getVmAddresses(vm workloads.VM) string {
	var addresses strings.Builder

	if vm.IP != "" {
		addresses.WriteString(fmt.Sprintf("wireguard: %v, ", vm.IP))
	}
	if vm.Planetary {
		addresses.WriteString(fmt.Sprintf("yggdrasil: %v, ", vm.PlanetaryIP))
	}
	if vm.PublicIP {
		addresses.WriteString(fmt.Sprintf("publicIp4: %v, ", vm.ComputedIP))
	}
	if vm.PublicIP6 {
		addresses.WriteString(fmt.Sprintf("publicIp6: %v, ", vm.ComputedIP6))
	}
	if len(vm.MyceliumIPSeed) != 0 {
		addresses.WriteString(fmt.Sprintf("mycelium: %v, ", vm.MyceliumIP))
	}

	return strings.TrimSuffix(addresses.String(), ", ")
}
