// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/cmd"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy the project in the current directory from threefold grid",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return err
		}

		err = command.Destroy(debug)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
