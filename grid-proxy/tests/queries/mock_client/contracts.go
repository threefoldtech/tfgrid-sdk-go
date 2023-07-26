package mock

import (
	"fmt"
	"reflect"
	"sort"

	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var rentContractFilterFieldValidator = map[string]func(contract RentContract, f proxytypes.ContractFilter) bool{
	"ContractID": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.ContractID == nil || contract.ContractID == *f.ContractID
	},
	"TwinID": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.TwinID == nil || contract.TwinID == *f.TwinID
	},
	"NodeID": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.NodeID == nil || contract.NodeID == *f.NodeID
	},
	"Type": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.Type == nil || *f.Type == "rent"
	},
	"State": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.State == nil || contract.State == *f.State
	},
	"Name": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.Name == nil || *f.Name == ""
	},
	"NumberOfPublicIps": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.NumberOfPublicIps == nil || *f.NumberOfPublicIps == 0
	},
	"DeploymentData": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.DeploymentData == nil || *f.DeploymentData == ""
	},
	"DeploymentHash": func(contract RentContract, f proxytypes.ContractFilter) bool {
		return f.DeploymentHash == nil || *f.DeploymentHash == ""
	},
}

var nodeContractFilterFieldValidator = map[string]func(contract NodeContract, f proxytypes.ContractFilter) bool{
	"ContractID": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.ContractID == nil || contract.ContractID == *f.ContractID
	},
	"TwinID": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.TwinID == nil || contract.TwinID == *f.TwinID
	},
	"NodeID": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.NodeID == nil || contract.NodeID == *f.NodeID
	},
	"Type": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.Type == nil || *f.Type == "node"
	},
	"State": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.State == nil || contract.State == *f.State
	},
	"Name": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.Name == nil || *f.Name == ""
	},
	"NumberOfPublicIps": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.NumberOfPublicIps == nil || contract.NumberOfPublicIPs >= *f.NumberOfPublicIps
	},
	"DeploymentData": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.DeploymentData == nil || *f.DeploymentData == contract.DeploymentData
	},
	"DeploymentHash": func(contract NodeContract, f proxytypes.ContractFilter) bool {
		return f.DeploymentHash == nil || *f.DeploymentHash == contract.DeploymentHash
	},
}

var nameContractFilterFieldValidator = map[string]func(contract NameContract, f proxytypes.ContractFilter) bool{
	"ContractID": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.ContractID == nil || contract.ContractID == *f.ContractID
	},
	"TwinID": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.TwinID == nil || contract.TwinID == *f.TwinID
	},
	"NodeID": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.NodeID == nil
	},
	"Type": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.Type == nil || *f.Type == "name"
	},
	"State": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.State == nil || contract.State == *f.State
	},
	"Name": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.Name == nil || *f.Name == contract.Name
	},
	"NumberOfPublicIps": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.NumberOfPublicIps == nil || *f.NumberOfPublicIps == 0
	},
	"DeploymentData": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.DeploymentData == nil || *f.DeploymentData == ""
	},
	"DeploymentHash": func(contract NameContract, f proxytypes.ContractFilter) bool {
		return f.DeploymentHash == nil || *f.DeploymentHash == ""
	},
}

// Contracts returns contracts with the given filters and pagination parameters
func (g *GridProxyMockClient) Contracts(filter proxytypes.ContractFilter, limit proxytypes.Limit) (res []proxytypes.Contract, totalCount int, err error) {
	res = []proxytypes.Contract{}

	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
	billings := make(map[uint64][]proxytypes.ContractBilling)
	for contractID, contractBillings := range g.data.Billings {
		for _, billing := range contractBillings {
			billings[contractID] = append(billings[contractID], proxytypes.ContractBilling{
				AmountBilled:     billing.AmountBilled,
				DiscountReceived: billing.DiscountReceived,
				Timestamp:        billing.Timestamp,
			})
		}
		sort.Slice(billings[contractID], func(i, j int) bool {
			return billings[contractID][i].Timestamp < billings[contractID][j].Timestamp
		})
	}

	for _, contract := range g.data.NodeContracts {
		satisfies, err := nodeContractsSatisfies(contract, filter)
		if err != nil {
			return res, totalCount, err
		}
		if satisfies {
			contract := proxytypes.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "node",
				Details: proxytypes.NodeContractDetails{
					NodeID:            uint(contract.NodeID),
					DeploymentData:    contract.DeploymentData,
					DeploymentHash:    contract.DeploymentHash,
					NumberOfPublicIps: uint(contract.NumberOfPublicIPs),
				},
				Billing: append([]proxytypes.ContractBilling{}, billings[contract.ContractID]...),
			}
			res = append(res, contract)
		}
	}

	for _, contract := range g.data.RentContracts {
		satisfies, err := rentContractsSatisfies(contract, filter)
		if err != nil {
			return res, totalCount, err
		}

		if satisfies {
			contract := proxytypes.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "rent",
				Details: proxytypes.RentContractDetails{
					NodeID: uint(contract.NodeID),
				},
				Billing: append([]proxytypes.ContractBilling{}, billings[contract.ContractID]...),
			}
			res = append(res, contract)
		}
	}

	for _, contract := range g.data.NameContracts {
		satisfies, err := nameContractsSatisfies(contract, filter)
		if err != nil {
			return res, totalCount, err
		}
		if satisfies {
			contract := proxytypes.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "name",
				Details: proxytypes.NameContractDetails{
					Name: contract.Name,
				},
				Billing: append([]proxytypes.ContractBilling{}, billings[contract.ContractID]...),
			}
			res = append(res, contract)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ContractID < res[j].ContractID
	})

	res, totalCount = getPage(res, limit)

	return
}

func rentContractsSatisfies(contract RentContract, f proxytypes.ContractFilter) (bool, error) {
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		valid, ok := rentContractFilterFieldValidator[v.Type().Field(i).Name]
		if !ok {
			return false, fmt.Errorf("Field %s has no validator", v.Type().Field(i).Name)
		}

		if !valid(contract, f) {
			return false, nil
		}
	}

	return true, nil
}

func nameContractsSatisfies(contract NameContract, f proxytypes.ContractFilter) (bool, error) {
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		valid, ok := nameContractFilterFieldValidator[v.Type().Field(i).Name]
		if !ok {
			return false, fmt.Errorf("Field %s has no validator", v.Type().Field(i).Name)
		}

		if !valid(contract, f) {
			return false, nil
		}
	}

	return true, nil
}

func nodeContractsSatisfies(contract NodeContract, f proxytypes.ContractFilter) (bool, error) {
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		valid, ok := nodeContractFilterFieldValidator[v.Type().Field(i).Name]
		if !ok {
			return false, fmt.Errorf("Field %s has no validator", v.Type().Field(i).Name)
		}

		if !valid(contract, f) {
			return false, nil
		}
	}

	return true, nil
}
