package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start a node in your farm",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonicOrSeed, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		nodeID, err := cmd.Flags().GetUint32("node")
		if err != nil {
			return fmt.Errorf("invalid node ID '%d'", nodeID)
		}

		identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonicOrSeed)
		if err != nil {
			return err
		}

		substrateManager := substrate.NewManager(constants.SubstrateURLs[network]...)
		subConn, err := substrateManager.Substrate()
		if err != nil {
			return err
		}

		defer subConn.Close()

		_, err = subConn.SetNodePowerTarget(identity, nodeID, true)
		if err != nil {
			return fmt.Errorf("failed to power on node '%d'", nodeID)
		}

		return nil
	},
}
