package client

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func decodeMultipleContracts(data []byte) ([]types.Contract, error) {
	// Unmarshal the data byte array into a slice of types.RawContract structs
	var rContracts []types.RawContract
	if err := json.Unmarshal(data, &rContracts); err != nil {
		return nil, err
	}

	contracts := []types.Contract{}
	for _, rContract := range rContracts {
		// Call the newContractFromRawContract function to convert each types.RawContract into a types.Contract struct
		contract, err := newContractFromRawContract(rContract)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, contract)
	}

	return contracts, nil
}

func decodeSingleContract(data []byte) (types.Contract, error) {
	var rContract types.RawContract
	if err := json.Unmarshal(data, &rContract); err != nil {
		return types.Contract{}, err
	}

	return newContractFromRawContract(rContract)
}

func newContractFromRawContract(rContract types.RawContract) (types.Contract, error) {
	var contract types.Contract

	// Assign values from the RawContract object to the corresponding fields in the Contract object
	contract.ContractID = rContract.ContractID
	contract.TwinID = rContract.TwinID
	contract.State = rContract.State
	contract.CreatedAt = rContract.CreatedAt
	contract.Type = rContract.Type

	switch rContract.Type {
	case "node":
		// Unmarshal the details of the contract based on the type
		var details types.NodeContractDetails
		if err := json.Unmarshal(rContract.Details, &details); err != nil {
			return types.Contract{}, err
		}
		contract.Details = details
		return contract, nil
	case "name":
		var details types.NameContractDetails
		if err := json.Unmarshal(rContract.Details, &details); err != nil {
			return types.Contract{}, err
		}
		contract.Details = details
		return contract, nil
	case "rent":
		var details types.RentContractDetails
		if err := json.Unmarshal(rContract.Details, &details); err != nil {
			return types.Contract{}, err
		}
		contract.Details = details
		return contract, nil
	default:
		return types.Contract{}, errors.Errorf("Unknown contract type: %s", rContract.Type)
	}
}
