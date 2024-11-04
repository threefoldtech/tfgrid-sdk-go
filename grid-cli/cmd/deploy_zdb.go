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

		count, err := cmd.Flags().GetInt("count")
		if err != nil {
			return err
		}

		names, err := cmd.Flags().GetStringSlice("names")
		if err != nil {
			return err
		}
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}

		if len(names) > 0 && len(names) != count {
			return fmt.Errorf("please provide '%d' names not '%d'", count, len(names))
		}

		if len(names) == 0 {
			for i := 0; i < count; i++ {
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

		size, err := cmd.Flags().GetUint64("size")
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
			SizeGB:      size,
			Description: description,
			Mode:        mode,
		}

		var zdbs []workloads.ZDB
		for i := 0; i < count; i++ {
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

		opts := []deployer.PluginOpt{
			deployer.WithNetwork(cfg.Network),
			deployer.WithRMBTimeout(100),
		}

		if noColor {
			opts = append(opts, deployer.WithNoColorLogs())
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		if node == 0 {
			filter, disks := filters.BuildZDBFilter(zdb, count, farm)
			nodes, err := deployer.FilterNodes(
				cmd.Context(),
				t,
				filter,
				nil,
				disks,
				nil,
			)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			node = uint32(nodes[0].NodeID)
		}

		resZDBs, err := command.DeployZDBs(cmd.Context(), t, projectName, zdbs, count, node)
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

	deployZDBCmd.Flags().StringP("project_name", "n", "", "project name of the zdbs to be deployed")
	err := deployZDBCmd.MarkFlagRequired("project_name")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deployZDBCmd.Flags().Uint64("size", 0, "hdd of zdb in gb")
	err = deployZDBCmd.MarkFlagRequired("size")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	deployZDBCmd.Flags().Int("count", 1, "number of zdbs to be deployed")

	deployZDBCmd.Flags().StringSlice("names", []string{}, "list of names of the zdb to be deployed")
	deployZDBCmd.Flags().String("password", "", "password of the zdb")
	deployZDBCmd.Flags().String("description", "", "description of the zdb")
	deployZDBCmd.Flags().String("mode", "user", "mode of zdb, if it is user or seq")
	deployZDBCmd.Flags().Bool("public", false, "if zdb gets a public ip6")

	deployZDBCmd.Flags().Uint32("node", 0, "node id that zdb should be deployed on")
	deployZDBCmd.Flags().Uint64("farm", 1, "farm id that zdb should be deployed on")
	deployZDBCmd.MarkFlagsMutuallyExclusive("node", "farm")
	deployZDBCmd.Flags().Bool("no-color", false, "disable output styling")
}
