package test

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

const (
	FARM_TESTS = 2000
)

var farmFilterRandomValueGenerator = map[string]func(agg FarmsAggregate) interface{}{
	"FreeIPs": func(agg FarmsAggregate) interface{} {
		return rndref(0, agg.maxFreeIPs)
	},
	"TotalIPs": func(agg FarmsAggregate) interface{} {
		return rndref(0, agg.maxTotalIPs)
	},
	"StellarAddress": func(agg FarmsAggregate) interface{} {
		c := agg.stellarAddresses[rand.Intn(len(agg.stellarAddresses))]
		if len(c) == 0 {
			return nil
		}

		return &c
	},
	"PricingPolicyID": func(agg FarmsAggregate) interface{} {
		return &agg.pricingPolicyIDs[rand.Intn(len(agg.pricingPolicyIDs))]
	},
	"FarmID": func(agg FarmsAggregate) interface{} {
		return &agg.farmIDs[rand.Intn(len(agg.farmIDs))]
	},
	"TwinID": func(agg FarmsAggregate) interface{} {
		return &agg.twinIDs[rand.Intn(len(agg.twinIDs))]
	},
	"Name": func(agg FarmsAggregate) interface{} {
		name := changeCase(agg.farmNames[rand.Intn(len(agg.farmNames))])
		if len(name) == 0 {
			return nil
		}

		return &name
	},
	"Country": func(_ FarmsAggregate) interface{} {
		aggNode := calcNodesAggregates(&data)
		country := changeCase(aggNode.countries[rand.Intn(len(aggNode.countries))])
		if len(country) == 0 {
			return nil
		}

		return &country
	},
	"Region": func(agg FarmsAggregate) interface{} {
		c := changeCase(agg.regions[rand.Intn(len(agg.regions))])
		if len(c) == 0 {
			return nil
		}

		return &c
	},
	"NameContains": func(agg FarmsAggregate) interface{} {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		runesList := []rune(c)
		a, b := rand.Intn(len(runesList)), rand.Intn(len(runesList))
		if a > b {
			a, b = b, a
		}
		runesList = runesList[a : b+1]
		c = string(runesList)
		if len(c) == 0 {
			return nil
		}

		return &c
	},
	"CertificationType": func(agg FarmsAggregate) interface{} {
		return &agg.certifications[rand.Intn(len(agg.certifications))]
	},
	"Dedicated": func(agg FarmsAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"NodeFreeMRU": func(agg FarmsAggregate) interface{} {
		aggNode := calcNodesAggregates(&data)
		mru := uint64(rand.Int63n(int64(aggNode.maxFreeMRU)))
		return &mru
	},
	"NodeFreeHRU": func(agg FarmsAggregate) interface{} {
		aggNode := calcNodesAggregates(&data)
		hru := uint64(rand.Int63n(int64(aggNode.maxFreeHRU)))
		return &hru
	},
	"NodeFreeSRU": func(agg FarmsAggregate) interface{} {
		aggNode := calcNodesAggregates(&data)
		sru := uint64(rand.Int63n(int64(aggNode.maxFreeSRU)))
		return &sru
	},
	"NodeTotalCRU": func(agg FarmsAggregate) interface{} {
		aggNode := calcNodesAggregates(&data)
		cru := uint64(rand.Int63n(int64(aggNode.maxTotalCRU)))
		return &cru
	},
	"NodeStatus": func(agg FarmsAggregate) interface{} {
		randomLen := rand.Intn(len(statuses))
		return getRandomSliceFrom(statuses, randomLen)
	},
	"NodeAvailableFor": func(agg FarmsAggregate) interface{} {
		return &agg.rentersTwinIDs[rand.Intn(len(agg.rentersTwinIDs))]
	},
	"NodeCertified": func(agg FarmsAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"NodeRentedBy": func(agg FarmsAggregate) interface{} {
		return &agg.rentersTwinIDs[rand.Intn(len(agg.rentersTwinIDs))]
	},
	"NodeHasGPU": func(agg FarmsAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"NodeHasIpv6": func(_ FarmsAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
}

type FarmsAggregate struct {
	regions          []string
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
	t.Parallel()
	t.Run("farms pagination test", func(t *testing.T) {
		t.Parallel()

		one := uint64(1)
		f := proxytypes.FarmFilter{
			TotalIPs: &one,
		}
		l := proxytypes.Limit{
			Size:     100,
			Page:     1,
			RetCount: true,
		}
		for ; ; l.Page++ {
			want, wantCount, err := mockClient.Farms(context.Background(), f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.Farms(context.Background(), f, l)
			require.NoError(t, err)

			assert.Equal(t, wantCount, gotCount)

			sortPublicIPs(want, got)
			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

			if l.Page*l.Size >= uint64(wantCount) {
				break
			}
		}
	})

	t.Run("farms stress test", func(t *testing.T) {
		t.Parallel()

		agg := calcFarmsAggregates(&data)
		for i := 0; i < FARM_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: true,
			}
			f, err := randomFarmsFilter(&agg)
			require.NoError(t, err)

			want, wantCount, err := mockClient.Farms(context.Background(), f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.Farms(context.Background(), f, l)
			require.NoError(t, err)

			assert.Equal(t, wantCount, gotCount)

			sortPublicIPs(want, got)
			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
		}
	})
	t.Run("farms list node free hru", func(t *testing.T) {
		t.Parallel()

		aggNode := calcNodesAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		f := proxytypes.FarmFilter{
			NodeFreeHRU: &aggNode.maxFreeHRU,
		}

		want, wantCount, err := mockClient.Farms(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Farms(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		sortPublicIPs(want, got)
		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})
	t.Run("farms list node free hru, mru", func(t *testing.T) {
		t.Parallel()

		aggNode := calcNodesAggregates(&data)

		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		f := proxytypes.FarmFilter{
			NodeFreeHRU: &aggNode.maxFreeHRU,
			NodeFreeMRU: &aggNode.maxFreeMRU,
		}

		want, wantCount, err := mockClient.Farms(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Farms(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		sortPublicIPs(want, got)
		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})
}

// TestFarmFilter iterates over all FarmFilter fields, and for each one generates a random value, then runs a test between the mock client and the gridproxy client
func TestFarmFilter(t *testing.T) {
	t.Parallel()

	f := proxytypes.FarmFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()
	l := proxytypes.Limit{
		Size:     9999999,
		Page:     1,
		RetCount: true,
	}

	agg := calcFarmsAggregates(&data)

	for i := 0; i < v.NumField(); i++ {
		generator, ok := farmFilterRandomValueGenerator[v.Type().Field(i).Name]
		require.True(t, ok, "Filter field %s has no random value generator", v.Type().Field(i).Name)

		randomFieldValue := generator(agg)
		if randomFieldValue == nil {
			continue
		}

		if v.Field(i).Type().Kind() != reflect.Slice {
			v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
		}
		v.Field(i).Set(reflect.ValueOf(randomFieldValue))

		want, wantCount, err := mockClient.Farms(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Farms(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		sortPublicIPs(want, got)
		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
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

	for _, contract := range data.RentContracts {
		res.rentersTwinIDs = append(res.rentersTwinIDs, contract.TwinID)
	}

	for _, region := range data.Regions {
		res.regions = append(res.regions, region)
	}

	farmIPs := make(map[uint64]uint64)
	farmTotalIPs := make(map[uint64]uint64)
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

func randomFarmsFilter(agg *FarmsAggregate) (proxytypes.FarmFilter, error) {
	f := proxytypes.FarmFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()

	for i := 0; i < v.NumField(); i++ {
		if rand.Float32() > .5 {
			generate, ok := farmFilterRandomValueGenerator[v.Type().Field(i).Name]
			if !ok {
				return proxytypes.FarmFilter{}, fmt.Errorf("Filter field %s has no random value generator", v.Type().Field(i).Name)
			}

			randomFieldValue := generate(*agg)
			if randomFieldValue == nil {
				continue
			}

			if v.Field(i).Type().Kind() != reflect.Slice {
				v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
			}
			v.Field(i).Set(reflect.ValueOf(randomFieldValue))
		}
	}

	return f, nil
}

func sortPublicIPs(local, remote []proxytypes.Farm) {
	for id := range local {
		sort.Slice(local[id].PublicIps, func(i, j int) bool {
			return local[id].PublicIps[i].ID < local[id].PublicIps[j].ID
		})
	}

	for id := range remote {
		sort.Slice(remote[id].PublicIps, func(i, j int) bool {
			return remote[id].PublicIps[i].ID < remote[id].PublicIps[j].ID
		})
	}

}
