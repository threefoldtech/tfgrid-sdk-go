// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/deployer"
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
		vmSpec, err := cmd.Flags().GetString("spec")
		if err != nil {
			return err
		}
		spec := deployer.VMSpec{}
		switch vmSpec {
		case "eco":
			spec = deployer.Eco
		case "standard":
			spec = deployer.Standard
		case "performance":
			spec = deployer.Performance
		}
		cpu, err := cmd.Flags().GetInt("cpu")
		if err != nil {
			return err
		}
		memory, err := cmd.Flags().GetInt("memory")
		if err != nil {
			return err
		}
		storage, err := cmd.Flags().GetInt("storage")
		if err != nil {
			return err
		}
		public, err := cmd.Flags().GetBool("public")
		if err != nil {
			return err
		}
		if spec == (deployer.VMSpec{}) {
			spec = deployer.VMSpec{CPU: cpu, Memory: memory, Storage: storage, Public: public}
		}

		err = command.Deploy(spec, ports, debug)
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
		log.Fatal().Err(err).Send()
	}
	deployCmd.Flags().StringP("spec", "s", "", "vm spec can be (eco, standard, performance)")
	deployCmd.Flags().Int("cpu", 1, "vm cpu, will be ignored if --spec is used")
	deployCmd.Flags().Int("memory", 2, "vm memory, will be ignored if --spec is used")
	deployCmd.Flags().Int("storage", 5, "vm storage, will be ignored if --spec is used")
	deployCmd.Flags().Bool("public", false, "vm public ip, will be ignored if --spec is used")

}
