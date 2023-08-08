// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/filters"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// deployZDBCmd represents the deploy zdb command
var deployZDBCmd = &cobra.Command{
	Use:   "zdb",
	Short: "Deploy a zdb",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := cmd.Flags().GetString("project_name")
		if err != nil {
			return err
		}

		n, err := cmd.Flags().GetInt("n")
		if err != nil {
			return err
		}

		names, err := cmd.Flags().GetStringSlice("names")
		if err != nil {
			return err
		}

		if len(names) > 0 && len(names) != n {
			return fmt.Errorf("please provide '%d' names not '%d'", n, len(names))
		}

		if len(names) == 0 {
			for i := 0; i < n; i++ {
				names = append(names, fmt.Sprintf("%s%d", projectName, i))
			}
		}

		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			return err
		}

		size, err := cmd.Flags().GetInt("size")
		if err != nil {
			return err
		}

		public, err := cmd.Flags().GetBool("public")
		if err != nil {
			return err
		}

		mode, err := cmd.Flags().GetString("mode")
		if err != nil {
			return err
		}

		if zos.ZDBMode(mode) != zos.ZDBModeUser && zos.ZDBMode(mode) != zos.ZDBModeSeq {
			return fmt.Errorf("invalid mode '%s', must be user or seq", mode)
		}

		node, err := cmd.Flags().GetUint32("node")
		if err != nil {
			return err
		}
		farm, err := cmd.Flags().GetUint64("farm")
		if err != nil {
			return err
		}

		zdb := workloads.ZDB{
			Password:    password,
			Public:      public,
			Size:        size,
			Description: description,
			Mode:        mode,
		}

		var zdbs []workloads.ZDB
		for i := 0; i < n; i++ {
			if strings.TrimSpace(names[i]) == "" {
				return fmt.Errorf("invalid empty name at index '%d'", i)
			}

			zdb.Name = names[i]
			zdbs = append(zdbs, zdb)
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, "sr25519", cfg.Network, "", "", "", 100, false)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		if node == 0 {
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filters.BuildZDBFilter(zdb, n, farm),
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			node = uint32(nodes[0].NodeID)
		}

		resZDBs, err := command.DeployZDBs(cmd.Context(), t, projectName, zdbs, n, node)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		for _, z := range resZDBs {
			log.Info().Msgf("zdb '%s' is deployed", z.Name)
		}
		return nil
	},
}

func init() {
	deployCmd.AddCommand(deployZDBCmd)

	deployZDBCmd.Flags().String("project_name", "", "project name of the zdbs to be deployed")
	err := deployZDBCmd.MarkFlagRequired("project_name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deployZDBCmd.Flags().Int("size", 0, "disk size in gb of zdb")
	err = deployZDBCmd.MarkFlagRequired("size")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deployZDBCmd.Flags().Int("n", 1, "number of zdbs to be deployed")

	deployZDBCmd.Flags().StringSlice("names", []string{}, "list of names of the zdb to be deployed")
	deployZDBCmd.Flags().String("password", "", "password of the zdb")
	deployZDBCmd.Flags().String("description", "", "description of the zdb")
	deployZDBCmd.Flags().String("mode", "user", "mode of zdb, if it is user or seq")
	deployZDBCmd.Flags().Bool("public", false, "make zdb public")

	deployZDBCmd.Flags().Uint32("node", 0, "node id that zdb should be deployed on")
	deployZDBCmd.Flags().Uint64("farm", 1, "farm id that zdb should be deployed on")
	deployZDBCmd.MarkFlagsMutuallyExclusive("node", "farm")
}
