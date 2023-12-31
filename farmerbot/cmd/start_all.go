package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
)

var startAllCmd = &cobra.Command{
	Use:   "all",
	Short: "start all nodes in your farm",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonicOrSeed, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		farmID, err := cmd.Flags().GetUint32("farm")
		if err != nil {
			return fmt.Errorf("invalid farm ID '%d'", farmID)
		}

		identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonicOrSeed)
		if err != nil {
			return err
		}

		substrateManager := substrate.NewManager(internal.SubstrateURLs[network]...)
		subConn, err := substrateManager.Substrate()
		if err != nil {
			return err
		}

		defer subConn.Close()

		farmNodes, err := subConn.GetNodes(farmID)
		if err != nil {
			return fmt.Errorf("failed to get nodes for farm '%d'", farmID)
		}

		for _, nodeID := range farmNodes {
			_, err = subConn.SetNodePowerTarget(identity, nodeID, true)
			if err != nil {
				return fmt.Errorf("failed to power on node '%d'", nodeID)
			}
		}

		log.Info().Msg("All nodes are started successfully")
		return nil
	},
}
