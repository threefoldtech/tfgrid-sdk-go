// Package cmd for parsing command line arguments
package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/cmd"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the project in the current directory on threefold grid",
	RunE: func(cmd *cobra.Command, args []string) error {
		ports, err := cmd.Flags().GetUintSlice("ports")
		if err != nil {
			return err
		}

		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return err
		}

		err = command.Deploy(ports, debug)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().UintSliceP("ports", "p", []uint{}, "ports to forward the FQDNs to")
	err := deployCmd.MarkFlagRequired("ports")
	if err != nil {
		log.Error().Err(err).Send()
		os.Exit(1)
	}
}
