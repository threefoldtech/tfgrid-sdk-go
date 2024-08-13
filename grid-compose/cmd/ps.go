package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "list deployments on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()

		if err := app.Ps(cmd.Context(), flags); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func init() {
	psCmd.PersistentFlags().BoolP("verbose", "v", false, "all information about deployed services")
	psCmd.PersistentFlags().StringP("output", "o", "", "output result to a file")
	rootCmd.AddCommand(psCmd)
}
