package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

type ContractsAggregate struct {
	contractIDs      []uint64
	TwinIDs          []uint64
	NodeIDs          []uint64
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
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSSWORD, POSTGRES_DB)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(errors.Wrap(err, "failed to open db"))
	}
	defer db.Close()

	data, err := load(db)
	if err != nil {
		panic(err)
	}
	proxyClient := proxyclient.NewClient(ENDPOINT)
	localClient := NewGridProxyClient(data)

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
			localContracts, localCount, err := localClient.Contracts(f, l)
			assert.NoError(t, err)

			remoteContracts, remoteCount, err := proxyClient.Contracts(f, l)
			assert.NoError(t, err)

			assert.Equal(t, localCount, remoteCount)
			require.True(t, reflect.DeepEqual(localContracts, remoteContracts), serializeFilter(f), cmp.Diff(localContracts, remoteContracts))

			if l.Page*l.Size >= uint64(localCount) {
				break
			}
			l.Page++
		}
	})

	t.Run("contracts stress test", func(t *testing.T) {
		agg := calcContractsAggregates(&data)
		for i := 0; i < CONTRACTS_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     9999999,
				Page:     1,
				RetCount: false,
			}
			f := randomContractsFilter(&agg)

			localContracts, _, err := localClient.Contracts(f, l)
			assert.NoError(t, err)
			remoteContracts, _, err := proxyClient.Contracts(f, l)
			assert.NoError(t, err)
			require.True(t, reflect.DeepEqual(localContracts, remoteContracts), serializeFilter(f), cmp.Diff(localContracts, remoteContracts))

		}
	})
}

func calcContractsAggregates(data *DBData) (res ContractsAggregate) {
	types := make(map[string]struct{})
	for _, contract := range data.nodeContracts {
		res.contractIDs = append(res.contractIDs, contract.contract_id)
		res.maxNumberOfPublicIPs = max(res.maxNumberOfPublicIPs, contract.number_of_public_i_ps)
		res.DeploymentDatas = append(res.DeploymentDatas, contract.deployment_data)
		res.DeploymentHashes = append(res.DeploymentHashes, contract.deployment_hash)
		res.NodeIDs = append(res.NodeIDs, contract.node_id)
		res.States = append(res.States, contract.state)
		res.TwinIDs = append(res.TwinIDs, contract.twin_id)
		types["node"] = struct{}{}
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
	if flip(.05) {
		c := agg.contractIDs[rand.Intn(len(agg.contractIDs))]
		f.ContractID = &c
	}
	if flip(.25) {
		c := agg.TwinIDs[rand.Intn(len(agg.TwinIDs))]
		f.TwinID = &c
	}
	if flip(.25) {
		c := agg.NodeIDs[rand.Intn(len(agg.NodeIDs))]
		f.NodeID = &c
	}
	if flip(.5) {
		c := agg.Types[rand.Intn(len(agg.Types))]
		f.Type = &c
	}
	if flip(.5) {
		c := agg.States[rand.Intn(len(agg.States))]
		f.State = &c
	}
	if flip(.25) && len(agg.Names) != 0 {
		c := agg.Names[rand.Intn(len(agg.Names))]
		f.Name = &c
	}
	if flip(.25) {
		f.NumberOfPublicIps = rndref(0, agg.maxNumberOfPublicIPs)
	}
	if flip(.25) && len(agg.DeploymentDatas) != 0 {
		c := agg.DeploymentDatas[rand.Intn(len(agg.DeploymentDatas))]
		f.DeploymentData = &c
	}
	if flip(.25) && len(agg.DeploymentHashes) != 0 {
		c := agg.DeploymentHashes[rand.Intn(len(agg.DeploymentHashes))]
		f.DeploymentHash = &c
	}
	return f
}
