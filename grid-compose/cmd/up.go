package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "deploy application on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		if err := up(cmd.Context()); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func up(ctx context.Context) error {
	results := make(map[string]workloads.VM)
	for key, val := range app.Specs.Services {
		projectName := internal.GetProjectName(key, app.Client.TwinID)

		networkName := key + "net"
		log.Info().Str("projectName", projectName).Str("workloadName", networkName).Msg("deploying network")
		net := workloads.ZNet{
			Name:  networkName,
			Nodes: []uint32{uint32(val.NodeID)},
			IPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}),
			SolutionType: projectName,
		}
		if err := app.Client.NetworkDeployer.Deploy(ctx, &net); err != nil {
			return err
		}

		log.Info().Str("projectName", projectName).Str("workloadName", key).Msg("deploying vm")
		vm := workloads.VM{
			Name:        key,
			Flist:       val.Flist,
			Entrypoint:  val.Entrypoint,
			EnvVars:     val.Environment,
			CPU:         int(val.Resources.CPU),
			Memory:      int(val.Resources.Memory),
			Planetary:   true,
			NetworkName: net.Name,
		}
		dl := workloads.NewDeployment(vm.Name, uint32(val.NodeID), projectName, nil, net.Name, nil, nil, []workloads.VM{vm}, nil)
		if err := app.Client.DeploymentDeployer.Deploy(ctx, &dl); err != nil {
			log.Error().Err(err).Msg("reverting deployed network")
			if err := app.Client.NetworkDeployer.Cancel(ctx, &net); err != nil {
				return err
			}
			return err
		}

		res, err := app.Client.State.LoadVMFromGrid(ctx, uint32(val.NodeID), vm.Name, dl.Name)
		if err != nil {
			return err
		}

		results[vm.Name] = res
	}

	for key, val := range results {
		fmt.Printf("%s vm addresses:\n", key)
		fmt.Println("\t", internal.GetVmAddresses(val))
	}
	return nil
}
