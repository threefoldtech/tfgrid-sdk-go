package deployer

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// CancelByProjectName cancels a deployed project
func (t *TFPluginClient) CancelByProjectName(projectName string, noGateways ...bool) error {
	log.Info().Str("project name", projectName).Msg("canceling contracts")

	contracts, err := t.ContractsGetter.ListContractsOfProjectName(projectName, noGateways...)
	if err != nil {
		return errors.Wrapf(err, "could not load contracts for project %s", projectName)
	}

	contractsSlice := append(contracts.NameContracts, contracts.NodeContracts...)

	const batchSize = 400 // Process contracts in groups of 400

	for i := 0; i < len(contractsSlice); i += batchSize {
		end := i + batchSize
		if end > len(contractsSlice) {
			end = len(contractsSlice)
		}

		batchContractIDS := make([]uint64, 0, batchSize) // New slice for each batch
		for _, contract := range contractsSlice[i:end] {
			contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
			if err != nil {
				return errors.Wrapf(err, "could not parse contract %s into uint64", contract.ContractID)
			}
			batchContractIDS = append(batchContractIDS, contractID)
		}

		if len(batchContractIDS) == 0 {
			continue // Skip empty batches
		}

		log.Debug().Uints64("contracts IDs", batchContractIDS).Msg("Batch cancel")
		if err := t.BatchCancelContract(batchContractIDS); err != nil {
			return fmt.Errorf("failed to cancel contracts (batch %d-%d) for project %s: %w", i, end, projectName, err)
		}
	}
	log.Info().Str("project name", projectName).Msg("project is canceled")
	return nil
}
