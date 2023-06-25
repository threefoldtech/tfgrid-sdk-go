package test

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/mock_client"
)

type ContractsAggregate struct {
	contractIDs      []uint64
	TwinIDs          []uint32
	NodeIDs          []uint32
	Types            []string
	States           []string
	Names            []string
	DeploymentDatas  []string
	DeploymentHashes []string

	maxNumberOfPublicIPs uint64
}

const (
	CONTRACTS_TESTS = 2000
)

func TestContracts(t *testing.T) {

	t.Run("contracts pagination test", func(t *testing.T) {
		node := "node"
		f := proxytypes.ContractFilter{
			Type: &node,
		}

		l := proxytypes.Limit{
			Size:     5,
			Page:     1,
			RetCount: true,
		}

		for {
			localContracts, localCount, err := mockClient.Contracts(f, l)
			assert.NoError(t, err)

			remoteContracts, remoteCount, err := proxyClient.Contracts(f, l)
			assert.NoError(t, err)

			require.Equal(t, localCount, remoteCount, serializeFilter(f))
			require.Equal(t, len(localContracts), len(remoteContracts), serializeFilter(f))

			require.True(t, reflect.DeepEqual(localContracts, remoteContracts), serializeFilter(f), cmp.Diff(localContracts, remoteContracts))

			if l.Page*l.Size >= uint64(localCount) {
				break
			}

			l.Page++
		}
	})

	t.Run("contracts stress test", func(t *testing.T) {
		agg := calcContractsAggregates(&dbData)
		for i := 0; i < CONTRACTS_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: true,
			}

			f := randomContractsFilter(&agg)

			localContracts, localCount, err := mockClient.Contracts(f, l)
			assert.NoError(t, err)

			remoteContracts, remoteCount, err := proxyClient.Contracts(f, l)
			assert.NoError(t, err)

			require.Equal(t, localCount, remoteCount, serializeFilter(f))
			require.Equal(t, len(localContracts), len(remoteContracts), serializeFilter(f))

			require.True(t, reflect.DeepEqual(localContracts, remoteContracts), serializeFilter(f), cmp.Diff(localContracts, remoteContracts))
		}
	})
}

func calcContractsAggregates(data *mock.DBData) (res ContractsAggregate) {
	types := make(map[string]struct{})

	for _, contract := range data.NodeContracts {
		res.contractIDs = append(res.contractIDs, contract.ContractID)
		res.maxNumberOfPublicIPs = max(res.maxNumberOfPublicIPs, contract.NumberOfPublicIPs)
		res.DeploymentDatas = append(res.DeploymentDatas, contract.DeploymentData)
		res.DeploymentHashes = append(res.DeploymentHashes, contract.DeploymentHash)
		res.NodeIDs = append(res.NodeIDs, contract.NodeID)
		res.States = append(res.States, contract.State)
		res.TwinIDs = append(res.TwinIDs, contract.TwinID)
		types["node"] = struct{}{}
	}

	for _, contract := range data.NameContracts {
		res.contractIDs = append(res.contractIDs, contract.ContractID)
		res.States = append(res.States, contract.State)
		res.TwinIDs = append(res.TwinIDs, contract.TwinID)
		res.Names = append(res.Names, contract.Name)
		types["name"] = struct{}{}
	}

	for _, contract := range data.RentContracts {
		res.contractIDs = append(res.contractIDs, contract.ContractID)
		res.NodeIDs = append(res.NodeIDs, contract.NodeID)
		res.States = append(res.States, contract.State)
		res.TwinIDs = append(res.TwinIDs, contract.TwinID)
		types["rent"] = struct{}{}
	}

	for typ := range types {
		res.Types = append(res.Types, typ)
	}
	sort.Slice(res.contractIDs, func(i, j int) bool {
		return res.contractIDs[i] < res.contractIDs[j]
	})
	sort.Slice(res.TwinIDs, func(i, j int) bool {
		return res.TwinIDs[i] < res.TwinIDs[j]
	})
	sort.Slice(res.NodeIDs, func(i, j int) bool {
		return res.NodeIDs[i] < res.NodeIDs[j]
	})
	sort.Slice(res.Types, func(i, j int) bool {
		return res.Types[i] < res.Types[j]
	})
	sort.Slice(res.States, func(i, j int) bool {
		return res.States[i] < res.States[j]
	})
	sort.Slice(res.Names, func(i, j int) bool {
		return res.Names[i] < res.Names[j]
	})
	sort.Slice(res.DeploymentDatas, func(i, j int) bool {
		return res.DeploymentDatas[i] < res.DeploymentDatas[j]
	})
	sort.Slice(res.DeploymentHashes, func(i, j int) bool {
		return res.DeploymentHashes[i] < res.DeploymentHashes[j]
	})
	return
}

func randomContractsFilter(agg *ContractsAggregate) proxytypes.ContractFilter {
	var f proxytypes.ContractFilter

	f.ContractID = contractRandContractID(agg)
	f.TwinID = contractRandTwinID(agg)
	f.NodeID = contractRandNodeID(agg)
	f.Type = contractRandContractType(agg)
	f.State = contractRandContractState(agg)
	f.Name = contractRandContractName(agg)
	f.NumberOfPublicIps = contractRandNumberOfPublicIPs(agg)
	f.DeploymentData = contractRandDeploymentData(agg)
	f.DeploymentHash = contractRandDeploymentHash(agg)

	return f
}

func contractRandContractID(agg *ContractsAggregate) *uint64 {
	if flip(.5) {
		c := agg.contractIDs[rand.Intn(len(agg.contractIDs))]
		return &c
	}

	return nil
}

func contractRandTwinID(agg *ContractsAggregate) *uint32 {
	if flip(.5) {
		c := agg.TwinIDs[rand.Intn(len(agg.TwinIDs))]
		return &c
	}

	return nil
}

func contractRandNodeID(agg *ContractsAggregate) *uint32 {
	if flip(.5) {
		c := agg.NodeIDs[rand.Intn(len(agg.NodeIDs))]
		return &c
	}

	return nil
}

func contractRandContractType(agg *ContractsAggregate) *string {
	if flip(.5) {
		c := agg.Types[rand.Intn(len(agg.Types))]
		return &c
	}

	return nil
}

func contractRandContractState(agg *ContractsAggregate) *string {
	if flip(.5) {
		c := agg.States[rand.Intn(len(agg.States))]
		return &c
	}

	return nil
}

func contractRandContractName(agg *ContractsAggregate) *string {
	if flip(.5) {
		c := agg.Names[rand.Intn(len(agg.Names))]
		return &c
	}

	return nil
}

func contractRandNumberOfPublicIPs(agg *ContractsAggregate) *uint64 {
	if flip(.5) {
		return rndref(0, agg.maxNumberOfPublicIPs)
	}

	return nil
}

func contractRandDeploymentData(agg *ContractsAggregate) *string {
	if flip(.5) {
		c := agg.DeploymentDatas[rand.Intn(len(agg.DeploymentDatas))]
		return &c
	}

	return nil
}

func contractRandDeploymentHash(agg *ContractsAggregate) *string {
	if flip(.5) {
		c := agg.DeploymentHashes[rand.Intn(len(agg.DeploymentHashes))]
		return &c
	}

	return nil
}
