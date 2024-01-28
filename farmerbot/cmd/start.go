package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start a node in your farm",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonicOrSeed, keyType, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		nodeID, err := cmd.Flags().GetUint32("node")
		if err != nil {
			return fmt.Errorf("invalid node ID '%d'", nodeID)
		}

		identity, err := internal.GetIdentityWithKeyType(mnemonicOrSeed, keyType)
		if err != nil {
			return err
		}

		substrateManager := substrate.NewManager(internal.SubstrateURLs[network]...)
		subConn, err := substrateManager.Substrate()
		if err != nil {
			return err
		}

		defer subConn.Close()

		_, err = subConn.SetNodePowerTarget(identity, nodeID, true)
		if err != nil {
			return fmt.Errorf("failed to power on node '%d' with error: %w", nodeID, err)
		}

		log.Info().Msgf("Node %d is started successfully", nodeID)
		return nil
	},
}
