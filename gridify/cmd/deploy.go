// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"errors"
	"time"

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
		cpu, err := cmd.Flags().GetUint8("cpu")
		if err != nil {
			return err
		}
		memory, err := cmd.Flags().GetUint64("memory")
		if err != nil {
			return err
		}
		storage, err := cmd.Flags().GetUint64("storage")
		if err != nil {
			return err
		}

		if spec == (deployer.VMSpec{}) {
			spec = deployer.VMSpec{CPU: cpu, Memory: memory, Storage: storage, Public: true}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err = command.Deploy(ctx, spec, ports, debug)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flag("spec").Changed &&
			(cmd.Flag("cpu").Changed || cmd.Flag("memory").Changed || cmd.Flag("storage").Changed) {
			return errors.New("spec flag cant't be set with cpu, memory or storage flags")
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
	deployCmd.Flags().Uint8("cpu", 1, "vm cpu")
	deployCmd.Flags().Uint64("memory", 2, "vm memory")
	deployCmd.Flags().Uint64("storage", 5, "vm storage")
}
