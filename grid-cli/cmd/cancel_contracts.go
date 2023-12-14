package cmd

import (
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
)

// cancelContracts represents the cancel contracts command
var cancelContracts = &cobra.Command{
	Use:   "contracts",
	Short: "Cancel twin contracts",
	Run: func(cmd *cobra.Command, args []string) {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if len(args) == 0 && !all {
			log.Info().Msg("please specify contracts to cancel or use -a to cancel all")
			return
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, "sr25519", cfg.Network, "", "", "", 100, false)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		var contracts []uint64
		if all {
			contracts, err = getAllContracts(t.ContractsGetter)
		} else {
			contracts, err = getContractsFromCLI(args)
		}
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		err = t.BatchCancelContract(contracts)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to cancel contracts")
		}
		log.Info().Msg("contracts canceled successfully")
	},
}

func init() {
	cancelCmd.AddCommand(cancelContracts)

	cancelContracts.Flags().BoolP("all", "a", false, "delete all contracts")
}

func getAllContracts(getter graphql.ContractsGetter) ([]uint64, error) {
	var contractIDs []uint64
	cs, err := getter.ListContractsByTwinID([]string{"Created"})
	if err != nil {
		return nil, err
	}
	contracts := append(cs.NameContracts, cs.NodeContracts...)
	contracts = append(contracts, cs.RentContracts...)
	for _, contract := range contracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s not a valid contract id: %w", contract.ContractID, err)
		}
		contractIDs = append(contractIDs, contractID)
	}
	return contractIDs, nil
}

func getContractsFromCLI(args []string) ([]uint64, error) {
	var contractIDs []uint64
	for _, arg := range args {
		contract, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s not a valid contract id: %w", arg, err)
		}
		contractIDs = append(contractIDs, contract)
	}
	return contractIDs, nil
}
