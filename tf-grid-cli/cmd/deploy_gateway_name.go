// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/workloads"
	command "github.com/threefoldtech/tfgrid-sdk-go/tf-grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/tf-grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/tf-grid-cli/internal/filters"
)

// deployGatewayNameCmd represents the deploy gateway name command
var deployGatewayNameCmd = &cobra.Command{
	Use:   "name",
	Short: "Deploy a gateway name proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, tls, zosBackends, node, farm, err := parseCommonGatewayFlags(cmd)
		if err != nil {
			return err
		}
		gateway := workloads.GatewayNameProxy{
			Name:           name,
			Backends:       zosBackends,
			TLSPassthrough: tls,
			SolutionType:   name,
		}
		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, "sr25519", cfg.Network, "", "", "", 100, true, false)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if node == 0 {
			node, err = filters.GetAvailableNode(
				t.GridProxyClient,
				filters.BuildGatewayFilter(farm),
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}
		gateway.NodeID = node
		resGateway, err := command.DeployGatewayName(t, gateway)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Msgf("fqdn: %s", resGateway.FQDN)
		return nil
	},
}

func init() {
	deployGatewayCmd.AddCommand(deployGatewayNameCmd)

}
