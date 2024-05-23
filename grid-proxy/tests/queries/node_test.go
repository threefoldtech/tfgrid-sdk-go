package test

import (
	"context"
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
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

type NodesAggregate struct {
	regions   []string
	countries []string
	cities    []string
	farmNames []string
	farmIDs   []uint64
	nodeIDs   []uint64
	freeMRUs  []uint64
	freeSRUs  []uint64
	freeHRUs  []uint64

	maxFreeMRU  uint64
	maxFreeSRU  uint64
	maxFreeHRU  uint64
	maxFreeIPs  uint64
	nodeRenters []uint64
	twins       []uint64

	totalCRUs   []uint64
	maxTotalCRU uint64
	totalHRUs   []uint64
	maxTotalHRU uint64
	totalMRUs   []uint64
	maxTotalMRU uint64
	totalSRUs   []uint64
	maxTotalSRU uint64
}

var (
	NODE_COUNT      = 1000
	NODE_TESTS      = 2000
	ErrNodeNotFound = errors.New("node not found")
)

var nodeFilterRandomValueGenerator = map[string]func(agg NodesAggregate) interface{}{
	"Status": func(agg NodesAggregate) interface{} {
		randomLen := rand.Intn(len(statuses))
		return getRandomSliceFrom(statuses, randomLen)
	},
	"Healthy": func(_ NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"HasIpv6": func(_ NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"FreeMRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.freeMRUs[rand.Intn(len(agg.freeMRUs))]
		}
		return rndref(0, agg.maxFreeMRU)

	},
	"FreeHRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.freeHRUs[rand.Intn(len(agg.freeHRUs))]
		}
		return rndref(0, agg.maxFreeHRU)
	},
	"FreeSRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.freeSRUs[rand.Intn(len(agg.freeSRUs))]
		}
		return rndref(0, agg.maxFreeSRU)
	},
	"TotalMRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.totalMRUs[rand.Intn(len(agg.totalMRUs))]
		}
		return rndref(0, agg.maxTotalMRU)
	},
	"TotalHRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.totalHRUs[rand.Intn(len(agg.totalHRUs))]
		}
		return rndref(0, agg.maxTotalHRU)

	},
	"TotalSRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.totalSRUs[rand.Intn(len(agg.totalSRUs))]
		}
		return rndref(0, agg.maxTotalSRU)

	},
	"TotalCRU": func(agg NodesAggregate) interface{} {
		if flip(.1) {
			return &agg.totalCRUs[rand.Intn(len(agg.totalCRUs))]
		}
		return rndref(0, agg.maxTotalCRU)
	},
	"Country": func(agg NodesAggregate) interface{} {
		country := changeCase(agg.countries[rand.Intn(len(agg.countries))])
		if len(country) == 0 {
			return nil
		}

		return &country
	},
	"Region": func(agg NodesAggregate) interface{} {
		region := changeCase(agg.regions[rand.Intn(len(agg.regions))])
		if len(region) == 0 {
			return nil
		}

		return &region
	},
	"CountryContains": func(agg NodesAggregate) interface{} {
		c := agg.countries[rand.Intn(len(agg.countries))]
		if len(c) == 0 {
			return nil
		}

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
	"City": func(agg NodesAggregate) interface{} {
		city := changeCase(agg.cities[rand.Intn(len(agg.cities))])
		if len(city) == 0 {
			return nil
		}

		return &city
	},
	"CityContains": func(agg NodesAggregate) interface{} {
		c := agg.cities[rand.Intn(len(agg.cities))]
		if len(c) == 0 {
			return nil
		}

		runesList := []rune(c)
		a, b := rand.Intn(len(runesList)), rand.Intn(len(runesList))
		if a > b {
			a, b = b, a
		}
		runesList = runesList[a : b+1]
		c = string(runesList)
		return &c
	},
	"FarmName": func(agg NodesAggregate) interface{} {
		name := changeCase(agg.farmNames[rand.Intn(len(agg.farmNames))])
		if len(name) == 0 {
			return nil
		}

		return &name
	},
	"FarmNameContains": func(agg NodesAggregate) interface{} {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		if len(c) == 0 {
			return nil
		}

		runesList := []rune(c)
		a, b := rand.Intn(len(runesList)), rand.Intn(len(runesList))
		if a > b {
			a, b = b, a
		}
		runesList = runesList[a : b+1]
		c = string(runesList)
		return &c
	},
	"FarmIDs": func(agg NodesAggregate) interface{} {
		farmIDs := []uint64{}
		for _, id := range agg.farmIDs {
			if flip(float32(min(3, uint64(len(agg.farmIDs)))) / float32(len(agg.farmIDs))) {
				farmIDs = append(farmIDs, id)
			}
		}
		return farmIDs
	},
	"FreeIPs": func(agg NodesAggregate) interface{} {
		return rndref(0, agg.maxFreeIPs)
	},
	"IPv4": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"IPv6": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"Domain": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"InDedicatedFarm": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"Dedicated": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"Rentable": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"Rented": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"RentedBy": func(agg NodesAggregate) interface{} {
		c := agg.twins[rand.Intn(len(agg.twins))]
		if flip(.9) && len(agg.nodeRenters) != 0 {
			c = agg.nodeRenters[rand.Intn(len(agg.nodeRenters))]
		}
		return &c
	},
	"AvailableFor": func(agg NodesAggregate) interface{} {
		c := agg.twins[rand.Intn(len(agg.twins))]
		if flip(.1) && len(agg.nodeRenters) != 0 {
			c = agg.nodeRenters[rand.Intn(len(agg.nodeRenters))]
		}
		return &c
	},
	"NodeID": func(agg NodesAggregate) interface{} {
		v := uint64(rand.Intn(1100)) // 1000 is the total nodes + 100 for non-existed cases
		return &v
	},
	"TwinID": func(agg NodesAggregate) interface{} {
		v := uint64(rand.Intn(3500))
		return &v
	},
	"CertificationType": func(agg NodesAggregate) interface{} {
		certType := "Diy"
		if flip(.5) {
			certType = "noCert"
		}
		return &certType
	},
	"HasGPU": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"NumGPU": func(agg NodesAggregate) interface{} {
		v := uint64(rand.Intn(3))
		return &v
	},
	"OwnedBy": func(_ NodesAggregate) interface{} {
		v := uint64(rand.Intn(110))
		return &v
	},
	"GpuDeviceName": func(agg NodesAggregate) interface{} {
		deviceNames := []string{"geforce", "radeon", "a", "hamada"}
		return &deviceNames[rand.Intn(len(deviceNames))]
	},
	"GpuVendorName": func(agg NodesAggregate) interface{} {
		vendorNames := []string{"amd", "intel", "a", "hamada"}
		return &vendorNames[rand.Intn(len(vendorNames))]
	},
	"GpuVendorID": func(agg NodesAggregate) interface{} {
		vendorIDs := []string{"1002", "1", "a"}
		return &vendorIDs[rand.Intn(len(vendorIDs))]
	},
	"GpuDeviceID": func(agg NodesAggregate) interface{} {
		deviceIDs := []string{"744c", "1", "a"}
		return &deviceIDs[rand.Intn(len(deviceIDs))]
	},
	"GpuAvailable": func(agg NodesAggregate) interface{} {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"PriceMin": func(_ NodesAggregate) interface{} {
		v := rand.Float64() * 1000
		return &v
	},
	"PriceMax": func(_ NodesAggregate) interface{} {
		v := rand.Float64() * 1000
		return &v
	},
	"Excluded": func(agg NodesAggregate) interface{} {
		shuffledIds := make([]uint64, len(agg.nodeIDs))
		copy(shuffledIds, agg.nodeIDs)
		for i := len(shuffledIds) - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			shuffledIds[i], shuffledIds[j] = shuffledIds[j], shuffledIds[i]
		}

		num := rand.Intn(10)
		return shuffledIds[:num]
	},
}

func TestNode(t *testing.T) {
	t.Parallel()
	t.Run("node pagination test", func(t *testing.T) {
		nodePaginationCheck(t, mockClient, gridProxyClient)
	})

	t.Run("single node test", func(t *testing.T) {
		singleNodeCheck(t, mockClient, gridProxyClient)
	})

	t.Run("node up test", func(t *testing.T) {
		t.Parallel()

		f := types.NodeFilter{
			Status: []string{STATUS_UP},
		}

		l := types.Limit{
			Size:     999999999,
			Page:     1,
			RetCount: true,
		}

		want, wantCount, err := mockClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})

	t.Run("node status test", func(t *testing.T) {
		t.Parallel()

		for i := 1; i <= NODE_COUNT; i++ {
			if flip(.3) {
				want, errWant := mockClient.NodeStatus(context.Background(), uint32(i))
				got, errGot := gridProxyClient.NodeStatus(context.Background(), uint32(i))

				if errGot != nil && errWant != nil {
					require.True(t, errors.As(errWant, &errGot), fmt.Sprintf("errors should match: want error %s, got error %s", errWant, errGot))
				} else {
					require.True(t, errWant == errGot)
				}

				require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
			}
		}
	})

	t.Run("node stress test", func(t *testing.T) {
		t.Parallel()

		agg := calcNodesAggregates(&data)
		for i := 0; i < NODE_TESTS; i++ {
			l := types.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: true,
			}
			f, err := randomNodeFilter(&agg)
			require.NoError(t, err)

			want, wantCount, err := mockClient.Nodes(context.Background(), f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.Nodes(context.Background(), f, l)
			require.NoError(t, err)

			assert.Equal(t, wantCount, gotCount)

			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
		}
	})

	t.Run("node not found test", func(t *testing.T) {
		t.Parallel()

		nodeID := 1000000000
		_, err := gridProxyClient.Node(context.Background(), uint32(nodeID))
		assert.Equal(t, err.Error(), ErrNodeNotFound.Error())
	})

	t.Run("nodes test certification_type filter", func(t *testing.T) {
		t.Parallel()

		certType := "Diy"
		nodes, _, err := gridProxyClient.Nodes(context.Background(), types.NodeFilter{CertificationType: &certType}, types.DefaultLimit())
		require.NoError(t, err)

		for _, node := range nodes {
			assert.Equal(t, node.CertificationType, certType, "certification_type filter did not work")
		}

		notExistCertType := "noCert"
		nodes, _, err = gridProxyClient.Nodes(context.Background(), types.NodeFilter{CertificationType: &notExistCertType}, types.DefaultLimit())
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("nodes test has_gpu filter", func(t *testing.T) {
		t.Parallel()

		l := proxytypes.DefaultLimit()
		hasGPU := true
		f := proxytypes.NodeFilter{
			HasGPU: &hasGPU,
		}

		_, wantCount, err := mockClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		_, gotCount, err := gridProxyClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)
	})

	t.Run("nodes test gpu vendor, device name filter", func(t *testing.T) {
		t.Parallel()

		device := "navi"
		vendor := "advanced"
		nodes, _, err := gridProxyClient.Nodes(context.Background(), types.NodeFilter{GpuDeviceName: &device, GpuVendorName: &vendor}, types.DefaultLimit())
		assert.NoError(t, err)

		localNodes, _, err := mockClient.Nodes(context.Background(), types.NodeFilter{GpuDeviceName: &device, GpuVendorName: &vendor}, types.DefaultLimit())
		assert.NoError(t, err)

		assert.Equal(t, len(nodes), len(localNodes), "gpu_device_name, gpu_vendor_name filters did not work")
	})

	t.Run("nodes test gpu vendor, device id filter", func(t *testing.T) {
		t.Parallel()

		device := "744c"
		vendor := "1002"
		nodes, _, err := gridProxyClient.Nodes(context.Background(), types.NodeFilter{GpuDeviceID: &device, GpuVendorID: &vendor}, types.DefaultLimit())
		assert.NoError(t, err)

		localNodes, _, err := mockClient.Nodes(context.Background(), types.NodeFilter{GpuDeviceID: &device, GpuVendorID: &vendor}, types.DefaultLimit())
		assert.NoError(t, err)

		assert.Equal(t, len(nodes), len(localNodes), "gpu_device_id, gpu_vendor_id filters did not work")
	})

	t.Run("nodes test gpu available", func(t *testing.T) {
		t.Parallel()

		available := false
		nodes, _, err := gridProxyClient.Nodes(context.Background(), types.NodeFilter{GpuAvailable: &available}, types.DefaultLimit())
		assert.NoError(t, err)

		localNodes, _, err := mockClient.Nodes(context.Background(), types.NodeFilter{GpuAvailable: &available}, types.DefaultLimit())
		assert.NoError(t, err)

		assert.Equal(t, len(nodes), len(localNodes), "gpu_available filter did not work")
	})

	t.Run("node staking discount", func(t *testing.T) {
		t.Parallel()

		limits := proxytypes.DefaultLimit()
		limits.Balance = 9999999999 // in usd

		got, _, err := gridProxyClient.Nodes(context.Background(), types.NodeFilter{}, limits)
		assert.NoError(t, err)

		want, _, err := mockClient.Nodes(context.Background(), types.NodeFilter{}, limits)
		assert.NoError(t, err)

		require.True(t, reflect.DeepEqual(want, got), "failed on testing staking discount", fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})
}

// TestNodeFilter iterates over all NodeFilter fields, and for each one generates a random value, then runs a test between the mock client and the gridproxy client
func TestNodeFilter(t *testing.T) {
	t.Parallel()

	f := types.NodeFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()
	l := types.Limit{
		Size:     9999999,
		Page:     1,
		RetCount: true,
	}

	agg := calcNodesAggregates(&data)

	for i := 0; i < v.NumField(); i++ {
		generator, ok := nodeFilterRandomValueGenerator[v.Type().Field(i).Name]
		require.True(t, ok, "Filter field %s has no random value generator", v.Type().Field(i).Name)

		randomFieldValue := generator(agg)
		if randomFieldValue == nil {
			continue
		}

		if v.Field(i).Type().Kind() != reflect.Slice {
			v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
		}

		v.Field(i).Set(reflect.ValueOf(randomFieldValue))

		want, wantCount, err := mockClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Nodes(context.Background(), f, l)
		require.NoError(t, err, SerializeFilter(f))

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
	}
}

func singleNodeCheck(t *testing.T, localClient proxyclient.Client, proxyClient proxyclient.Client) {
	t.Parallel()
	nodeID := rand.Intn(NODE_COUNT)
	want, errWant := mockClient.Node(context.Background(), uint32(nodeID))

	got, errGot := gridProxyClient.Node(context.Background(), uint32(nodeID))

	if errGot != nil && errWant != nil {
		require.True(t, errors.As(errWant, &errGot))
	} else {
		require.True(t, errWant == errGot)
	}

	require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
}

func nodePaginationCheck(t *testing.T, localClient proxyclient.Client, proxyClient proxyclient.Client) {
	f := types.NodeFilter{
		Status: []string{STATUS_DOWN},
	}
	l := types.Limit{
		Size:     100,
		Page:     1,
		RetCount: true,
	}
	for ; ; l.Page++ {
		want, wantCount, err := mockClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Nodes(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		if l.Page*l.Size >= uint64(wantCount) {
			break
		}
	}
}

func randomNodeFilter(agg *NodesAggregate) (types.NodeFilter, error) {
	f := types.NodeFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()

	for i := 0; i < v.NumField(); i++ {
		if rand.Float32() > .5 {
			_, ok := nodeFilterRandomValueGenerator[v.Type().Field(i).Name]
			if !ok {
				return types.NodeFilter{}, fmt.Errorf("Filter field %s has no random value generator", v.Type().Field(i).Name)
			}

			randomFieldValue := nodeFilterRandomValueGenerator[v.Type().Field(i).Name](*agg)
			if v.Field(i).Type().Kind() != reflect.Slice {
				v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
			}
			if randomFieldValue == nil {
				continue
			}

			v.Field(i).Set(reflect.ValueOf(randomFieldValue))
		}
	}

	return f, nil
}

func calcNodesAggregates(data *mock.DBData) (res NodesAggregate) {
	cities := make(map[string]struct{})
	countries := make(map[string]struct{})
	for _, node := range data.Nodes {
		cities[node.City] = struct{}{}
		countries[node.Country] = struct{}{}
		total := data.NodeTotalResources[node.NodeID]
		free := mock.CalcFreeResources(total, data.NodeUsedResources[node.NodeID])
		freeHRU := free.HRU
		if int64(freeHRU) < 0 {
			freeHRU = 0
		}
		freeMRU := free.MRU
		if int64(freeMRU) < 0 {
			freeMRU = 0
		}

		freeSRU := free.SRU
		if int64(freeSRU) < 0 {
			freeSRU = 0
		}

		res.maxFreeHRU = max(res.maxFreeHRU, freeHRU)
		res.maxFreeSRU = max(res.maxFreeSRU, freeSRU)
		res.maxFreeMRU = max(res.maxFreeMRU, freeMRU)
		res.freeMRUs = append(res.freeMRUs, freeMRU)
		res.freeSRUs = append(res.freeSRUs, freeSRU)
		res.freeHRUs = append(res.freeHRUs, freeHRU)
		res.nodeIDs = append(res.nodeIDs, node.NodeID)

		res.maxTotalMRU = max(res.maxTotalMRU, total.MRU)
		res.totalMRUs = append(res.totalMRUs, total.MRU)
		res.maxTotalCRU = max(res.maxTotalCRU, total.CRU)
		res.totalCRUs = append(res.totalCRUs, total.CRU)
		res.maxTotalSRU = max(res.maxTotalSRU, total.SRU)
		res.totalSRUs = append(res.totalSRUs, total.SRU)
		res.maxTotalHRU = max(res.maxTotalHRU, total.HRU)
		res.totalHRUs = append(res.totalHRUs, total.HRU)
	}
	for _, contract := range data.RentContracts {
		if contract.State == "Deleted" {
			continue
		}
		res.nodeRenters = append(res.nodeRenters, contract.TwinID)
	}
	for _, twin := range data.Twins {
		res.twins = append(res.twins, twin.TwinID)
	}

	for city := range cities {
		res.cities = append(res.cities, city)
	}

	for country := range countries {
		res.countries = append(res.countries, country)
	}
	for _, farm := range data.Farms {
		res.farmNames = append(res.farmNames, farm.Name)
		res.farmIDs = append(res.farmIDs, farm.FarmID)
	}

	farmIPs := make(map[uint64]uint64)
	for _, publicIP := range data.PublicIPs {
		if publicIP.ContractID == 0 {
			farmIPs[data.FarmIDMap[publicIP.FarmID]] += 1
		}
	}

	for _, cnt := range farmIPs {
		res.maxFreeIPs = max(res.maxFreeIPs, cnt)
	}

	for _, region := range data.Regions {
		res.regions = append(res.regions, region)
	}

	sort.Slice(res.countries, func(i, j int) bool {
		return res.countries[i] < res.countries[j]
	})

	sort.Slice(res.cities, func(i, j int) bool {
		return res.cities[i] < res.cities[j]
	})

	sort.Slice(res.farmNames, func(i, j int) bool {
		return res.farmNames[i] < res.farmNames[j]
	})

	sort.Slice(res.farmIDs, func(i, j int) bool {
		return res.farmIDs[i] < res.farmIDs[j]
	})

	sort.Slice(res.freeMRUs, func(i, j int) bool {
		return res.freeMRUs[i] < res.freeMRUs[j]
	})

	sort.Slice(res.freeSRUs, func(i, j int) bool {
		return res.freeSRUs[i] < res.freeSRUs[j]
	})

	sort.Slice(res.freeHRUs, func(i, j int) bool {
		return res.freeHRUs[i] < res.freeHRUs[j]
	})

	sort.Slice(res.nodeRenters, func(i, j int) bool {
		return res.nodeRenters[i] < res.nodeRenters[j]
	})

	sort.Slice(res.twins, func(i, j int) bool {
		return res.twins[i] < res.twins[j]
	})

	return
}
