package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// getContractsCmd represents the get contracts command
var getContractsCmd = &cobra.Command{
	Use:   "contracts",
	Short: "Get twin contracts",
	Run: func(cmd *cobra.Command, args []string) {
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return
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
		contracts, err := t.ContractsGetter.ListContractsByTwinID([]string{"Created"})
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		printContractTables(contracts, cmd.OutOrStdout())
	},
}

func init() {
	getCmd.AddCommand(getContractsCmd)
	getContractsCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}

func printContractTables(contracts graphql.Contracts, writer io.Writer) {
	fmt.Fprintln(writer, "Node contracts:")

	nodeTable := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	fmt.Fprintln(nodeTable, "ID\tNode ID\tType\tName\tProject Name")
	for _, contract := range contracts.NodeContracts {
		// ignoring the error because deployment data does not have a standard structure throughout the grid
		// it will be displayed as empty columns in case it failed
		data, _ := workloads.ParseDeploymentData(contract.DeploymentData)
		fmt.Fprintf(nodeTable, "%s\t%d\t%s\t%s\t%s\n", contract.ContractID, contract.NodeID, data.Type, data.Name, data.ProjectName)
	}
	nodeTable.Flush()

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Name contracts:")

	nameTable := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	fmt.Fprintln(nameTable, "ID\tName")
	for _, contract := range contracts.NameContracts {
		fmt.Fprintf(nameTable, "%s\t%s\n", contract.ContractID, contract.Name)
	}
	nameTable.Flush()
}
