// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/filters"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// deployGatewayNameCmd represents the deploy gateway name command
var deployGatewayNameCmd = &cobra.Command{
	Use:   "name",
	Short: "Deploy a gateway name proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, tls, zosBackends, node, err := parseCommonGatewayFlags(cmd)
		if err != nil {
			return err
		}
		gateway := workloads.GatewayNameProxy{
			Name:           name,
			Backends:       zosBackends,
			TLSPassthrough: tls,
			SolutionType:   name,
		}
		farm, err := cmd.Flags().GetUint64("farm")
		if err != nil {
			return err
		}
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		opts := []deployer.PluginOpt{
			deployer.WithNetwork(cfg.Network),
			deployer.WithRMBTimeout(100),
		}

		if noColor {
			opts = append(opts, deployer.WithNoColorLogs())
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if node == 0 {
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filters.BuildGatewayFilter(farm),
				nil,
				nil,
				nil,
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			node = uint32(nodes[0].NodeID)
		}
		gateway.NodeID = node
		resGateway, err := command.DeployGatewayName(cmd.Context(), t, gateway)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Msgf("fqdn: %s", resGateway.FQDN)
		return nil
	},
}

func init() {
	deployGatewayCmd.AddCommand(deployGatewayNameCmd)

	deployGatewayNameCmd.Flags().Uint32("node", 0, "node id gateway should be deployed on")
	deployGatewayNameCmd.Flags().Uint64("farm", 1, "farm id gateway should be deployed on")
	deployGatewayNameCmd.MarkFlagsMutuallyExclusive("node", "farm")
	deployGatewayNameCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}
