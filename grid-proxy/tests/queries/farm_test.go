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

const (
	FARM_TESTS = 2000
)

type FarmsAggregate struct {
	stellarAddresses []string
	pricingPolicyIDs []uint64
	farmNames        []string
	farmIDs          []uint64
	twinIDs          []uint64
	certifications   []string
	rentersTwinIDs   []uint64

	maxFreeIPs  uint64
	maxTotalIPs uint64
}

func TestFarm(t *testing.T) {
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

	t.Run("farms pagination test", func(t *testing.T) {
		one := uint64(1)
		f := proxytypes.FarmFilter{
			TotalIPs: &one,
		}
		l := proxytypes.Limit{
			Size:     5,
			Page:     1,
			RetCount: true,
		}
		for ; ; l.Page++ {
			localFarms, localCount, err := localClient.Farms(f, l)
			assert.NoError(t, err)
			remoteFarms, remoteCount, err := proxyClient.Farms(f, l)
			assert.NoError(t, err)
			assert.Equal(t, localCount, remoteCount)
			sortPublicIPs(localFarms, remoteFarms)
			require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(f), cmp.Diff(localFarms, remoteFarms))

			if l.Page*l.Size >= uint64(localCount) {
				break
			}
		}
	})

	t.Run("farms stress test", func(t *testing.T) {
		agg := calcFarmsAggregates(&data)
		for i := 0; i < FARM_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: false,
			}
			f := randomFarmsFilter(&agg)
			localFarms, _, err := localClient.Farms(f, l)
			assert.NoError(t, err)
			remoteFarms, _, err := proxyClient.Farms(f, l)
			assert.NoError(t, err)
			sortPublicIPs(localFarms, remoteFarms)

			require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(f), cmp.Diff(localFarms, remoteFarms))

		}
	})
	t.Run("farms list node free hru", func(t *testing.T) {
		aggNode := calcNodesAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: false,
		}
		filter := proxytypes.FarmFilter{
			NodeFreeHRU: &aggNode.maxFreeHRU,
		}
		localFarms, _, err := localClient.Farms(filter, l)
		assert.NoError(t, err)
		remoteFarms, _, err := proxyClient.Farms(filter, l)
		assert.NoError(t, err)
		sortPublicIPs(localFarms, remoteFarms)

		require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(filter), cmp.Diff(localFarms, remoteFarms))

	})
	t.Run("farms list node free hru, mru", func(t *testing.T) {
		aggNode := calcNodesAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: false,
		}
		filter := proxytypes.FarmFilter{
			NodeFreeHRU: &aggNode.maxFreeHRU,
			NodeFreeMRU: &aggNode.maxFreeMRU,
		}
		localFarms, _, err := localClient.Farms(filter, l)
		assert.NoError(t, err)
		remoteFarms, _, err := proxyClient.Farms(filter, l)
		assert.NoError(t, err)
		sortPublicIPs(localFarms, remoteFarms)

		require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(filter), cmp.Diff(localFarms, remoteFarms))

	})
}

func calcFarmsAggregates(data *DBData) (res FarmsAggregate) {
	for _, farm := range data.farms {
		res.farmNames = append(res.farmNames, farm.name)
		res.stellarAddresses = append(res.stellarAddresses, farm.stellar_address)
		res.pricingPolicyIDs = append(res.pricingPolicyIDs, farm.pricing_policy_id)
		res.certifications = append(res.certifications, farm.certification)
		res.farmIDs = append(res.farmIDs, farm.farm_id)
		res.twinIDs = append(res.twinIDs, farm.twin_id)
	}

	for _, contract := range data.rentContracts {
		res.rentersTwinIDs = append(res.rentersTwinIDs, contract.twin_id)
	}

	farmIPs := make(map[uint64]uint64)
	farmTotalIPs := make(map[uint64]uint64)
	for _, publicIP := range data.publicIPs {
		if publicIP.contract_id == 0 {
			farmIPs[data.farmIDMap[publicIP.farm_id]] += 1
		}
		farmTotalIPs[data.farmIDMap[publicIP.farm_id]] += 1
	}
	for _, cnt := range farmIPs {
		res.maxFreeIPs = max(res.maxFreeIPs, cnt)
	}
	for _, cnt := range farmTotalIPs {
		res.maxTotalIPs = max(res.maxTotalIPs, cnt)
	}

	sort.Slice(res.stellarAddresses, func(i, j int) bool {
		return res.stellarAddresses[i] < res.stellarAddresses[j]
	})
	sort.Slice(res.pricingPolicyIDs, func(i, j int) bool {
		return res.pricingPolicyIDs[i] < res.pricingPolicyIDs[j]
	})
	sort.Slice(res.farmNames, func(i, j int) bool {
		return res.farmNames[i] < res.farmNames[j]
	})
	sort.Slice(res.farmIDs, func(i, j int) bool {
		return res.farmIDs[i] < res.farmIDs[j]
	})
	sort.Slice(res.twinIDs, func(i, j int) bool {
		return res.twinIDs[i] < res.twinIDs[j]
	})
	sort.Slice(res.certifications, func(i, j int) bool {
		return res.certifications[i] < res.certifications[j]
	})

	return
}

func randomFarmsFilter(agg *FarmsAggregate) proxytypes.FarmFilter {
	var f proxytypes.FarmFilter
	if flip(.5) {
		f.FreeIPs = rndref(0, agg.maxFreeIPs)
	}
	if flip(.5) {
		f.TotalIPs = rndref(0, agg.maxTotalIPs)
	}
	if flip(.05) {
		c := agg.stellarAddresses[rand.Intn(len(agg.stellarAddresses))]
		f.StellarAddress = &c
	}
	if flip(.5) {
		c := agg.pricingPolicyIDs[rand.Intn(len(agg.pricingPolicyIDs))]
		f.PricingPolicyID = &c
	}
	if flip(.05) {
		c := agg.farmIDs[rand.Intn(len(agg.farmIDs))]
		f.FarmID = &c
	}
	if flip(.05) {
		c := agg.twinIDs[rand.Intn(len(agg.twinIDs))]
		f.TwinID = &c
	}
	if flip(.05) {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		v := changeCase(c)
		f.Name = &v
	}
	if flip(.05) {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]
		f.NameContains = &c
	}
	if flip(.5) {
		c := agg.certifications[rand.Intn(len(agg.certifications))]
		f.CertificationType = &c
	}
	if flip(.5) {
		v := true
		if flip(.5) {
			v = false
		}
		f.Dedicated = &v
	}
	if flip(.5) {
		v := agg.rentersTwinIDs[rand.Intn(len(agg.rentersTwinIDs))]
		f.NodeAvailableFor = &v
	}
	if flip(.5) {
		v := true
		if flip(.5) {
			v = false
		}
		f.NodeCertified = &v
	}
	if flip(.5) {
		v := true
		if flip(.5) {
			v = false
		}
		f.NodeHasGPU = &v
	}
	if flip(.5) {
		v := agg.rentersTwinIDs[rand.Intn(len(agg.rentersTwinIDs))]
		f.NodeRentedBy = &v
	}
	if flip(.5) {
		nodeStatuses := []string{"up", "down", "standby"}
		f.NodeStatus = &nodeStatuses[rand.Intn(len(nodeStatuses))]
	}

	return f
}

func sortPublicIPs(local, remote []proxytypes.Farm) {
	for id := range local {
		sort.Slice(local[id].PublicIps, func(i, j int) bool {
			return local[id].PublicIps[i].ID < local[id].PublicIps[j].ID
		})

		sort.Slice(remote[id].PublicIps, func(i, j int) bool {
			return remote[id].PublicIps[i].ID < remote[id].PublicIps[j].ID
		})

	}

}
