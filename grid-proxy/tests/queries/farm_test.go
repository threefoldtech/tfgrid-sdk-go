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

const (
	FARM_TESTS = 2000
)

type FarmsAggregate struct {
	stellarAddresses []string
	pricingPolicyIDs []uint32
	farmNames        []string
	farmIDs          []uint32
	twinIDs          []uint32
	certifications   []string

	maxFreeIPs  uint64
	maxTotalIPs uint64
}

func TestFarm(t *testing.T) {
	t.Run("farms pagination test", func(t *testing.T) {
		t.Parallel()

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
			localFarms, localCount, err := mockClient.Farms(f, l)
			assert.NoError(t, err)
			remoteFarms, remoteCount, err := proxyClient.Farms(f, l)
			assert.NoError(t, err)

			require.Equal(t, localCount, remoteCount, serializeFilter(f))
			require.Equal(t, len(localFarms), len(remoteFarms), serializeFilter(f))

			sortPublicIPs(localFarms, remoteFarms)
			require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(f), cmp.Diff(localFarms, remoteFarms))

			if l.Page*l.Size >= uint64(localCount) {
				break
			}
		}
	})

	t.Run("farms stress test", func(t *testing.T) {
		t.Parallel()

		agg := calcFarmsAggregates(&dbData)
		for i := 0; i < FARM_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: false,
			}
			f := randomFarmsFilter(&agg)
			localFarms, _, err := mockClient.Farms(f, l)
			assert.NoError(t, err)
			remoteFarms, _, err := proxyClient.Farms(f, l)
			assert.NoError(t, err)

			require.Equal(t, len(localFarms), len(remoteFarms), serializeFilter(f))

			sortPublicIPs(localFarms, remoteFarms)
			require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(f), cmp.Diff(localFarms, remoteFarms))
		}
	})
	t.Run("farms list node free hru", func(t *testing.T) {
		t.Parallel()

		aggNode, err := calcNodesAggregates(&dbData)
		assert.NoError(t, err)

		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: false,
		}
		filter := proxytypes.FarmFilter{
			NodeFreeHRU: &aggNode.maxFreeHRU,
		}
		localFarms, _, err := mockClient.Farms(filter, l)
		assert.NoError(t, err)
		remoteFarms, _, err := proxyClient.Farms(filter, l)
		assert.NoError(t, err)

		require.Equal(t, len(localFarms), len(remoteFarms), serializeFilter(filter))

		sortPublicIPs(localFarms, remoteFarms)
		require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(filter), cmp.Diff(localFarms, remoteFarms))

	})
	t.Run("farms list node free hru, mru", func(t *testing.T) {
		t.Parallel()

		aggNode, err := calcNodesAggregates(&dbData)
		assert.NoError(t, err)

		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: false,
		}
		filter := proxytypes.FarmFilter{
			NodeFreeHRU: &aggNode.maxFreeHRU,
			NodeFreeMRU: &aggNode.maxFreeMRU,
		}
		localFarms, _, err := mockClient.Farms(filter, l)
		assert.NoError(t, err)
		remoteFarms, _, err := proxyClient.Farms(filter, l)
		assert.NoError(t, err)

		require.Equal(t, len(localFarms), len(remoteFarms), serializeFilter(filter))

		sortPublicIPs(localFarms, remoteFarms)
		require.True(t, reflect.DeepEqual(localFarms, remoteFarms), serializeFilter(filter), cmp.Diff(localFarms, remoteFarms))

	})
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

func calcFarmsAggregates(data *mock.DBData) (res FarmsAggregate) {
	for _, farm := range data.Farms {
		res.farmNames = append(res.farmNames, farm.Name)
		res.stellarAddresses = append(res.stellarAddresses, farm.StellarAddress)
		res.pricingPolicyIDs = append(res.pricingPolicyIDs, farm.PricingPolicyID)
		res.certifications = append(res.certifications, farm.Certification)
		res.farmIDs = append(res.farmIDs, farm.FarmID)
		res.twinIDs = append(res.twinIDs, farm.TwinID)
	}

	farmIPs := make(map[uint32]uint64)
	farmTotalIPs := make(map[uint32]uint64)
	for _, publicIP := range data.PublicIPs {
		if publicIP.ContractID == 0 {
			farmIPs[data.FarmIDMap[publicIP.FarmID]] += 1
		}
		farmTotalIPs[data.FarmIDMap[publicIP.FarmID]] += 1
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

// randomFarmsFilter should contain a random generator for each field to have more robust tests
func randomFarmsFilter(agg *FarmsAggregate) proxytypes.FarmFilter {
	var f proxytypes.FarmFilter

	f.FreeIPs = farmRandFreeIPs(agg)
	f.TotalIPs = farmRandTotalIPs(agg)
	f.StellarAddress = farmRandStellarAddress(agg)
	f.PricingPolicyID = farmRandPricingPolicyID(agg)
	f.FarmID = farmRandFarmID(agg)
	f.TwinID = farmRandTwinID(agg)
	f.Name = farmRandName(agg)
	f.NameContains = farmRandNameContains(agg)
	f.CertificationType = farmRandCertificationType(agg)
	f.Dedicated = farmRandDedicated(agg)

	return f
}

func farmRandFreeIPs(agg *FarmsAggregate) *uint64 {
	if flip(.5) {
		return rndref(0, agg.maxFreeIPs)
	}

	return nil
}

func farmRandTotalIPs(agg *FarmsAggregate) *uint64 {
	if flip(.5) {
		return rndref(0, agg.maxTotalIPs)
	}

	return nil
}

func farmRandStellarAddress(agg *FarmsAggregate) *string {
	if flip(.5) {
		c := agg.stellarAddresses[rand.Intn(len(agg.stellarAddresses))]
		return &c
	}

	return nil
}

func farmRandPricingPolicyID(agg *FarmsAggregate) *uint32 {
	if flip(.5) {
		c := agg.pricingPolicyIDs[rand.Intn(len(agg.pricingPolicyIDs))]
		return &c
	}

	return nil
}

func farmRandFarmID(agg *FarmsAggregate) *uint32 {
	if flip(.5) {
		c := agg.farmIDs[rand.Intn(len(agg.farmIDs))]
		return &c
	}

	return nil
}

func farmRandTwinID(agg *FarmsAggregate) *uint32 {
	if flip(.5) {
		c := agg.twinIDs[rand.Intn(len(agg.twinIDs))]
		return &c
	}

	return nil
}

func farmRandName(agg *FarmsAggregate) *string {
	if flip(.5) {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		v := changeCase(c)
		return &v
	}

	return nil
}

func farmRandNameContains(agg *FarmsAggregate) *string {
	if flip(.5) {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]
		return &c
	}

	return nil
}

func farmRandCertificationType(agg *FarmsAggregate) *string {
	if flip(.5) {
		c := agg.certifications[rand.Intn(len(agg.certifications))]
		return &c
	}

	return nil
}

func farmRandDedicated(agg *FarmsAggregate) *bool {
	if flip(.5) {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	}

	return nil
}
