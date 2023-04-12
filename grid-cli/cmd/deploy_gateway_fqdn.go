// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/filters"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/workloads"
)

// deployGatewayFQDNCmd represents the deploy gateway fqdn command
var deployGatewayFQDNCmd = &cobra.Command{
	Use:   "fqdn",
	Short: "Deploy a gateway FQDN proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, tls, zosBackends, node, farm, err := parseCommonGatewayFlags(cmd)
		if err != nil {
			return err
		}
		fqdn, err := cmd.Flags().GetString("fqdn")
		if err != nil {
			return err
		}
		gateway := workloads.GatewayFQDNProxy{
			Name:           name,
			Backends:       zosBackends,
			TLSPassthrough: tls,
			SolutionType:   name,
			FQDN:           fqdn,
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
		err = command.DeployGatewayFQDN(t, gateway)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Msg("gateway fqdn deployed")
		return nil
	},
}

func init() {
	deployGatewayCmd.AddCommand(deployGatewayFQDNCmd)

	deployGatewayFQDNCmd.Flags().String("fqdn", "", "fqdn pointing to the specified node")
	err := deployGatewayFQDNCmd.MarkFlagRequired("fqdn")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

}
