package test

import (
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
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

type NodesAggregate struct {
	countries []string
	cities    []string
	farmNames []string
	farmIDs   []uint64
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
		return &statuses[rand.Intn(3)]
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
		return &country
	},
	"CountryContains": func(agg NodesAggregate) interface{} {
		c := agg.countries[rand.Intn(len(agg.countries))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]
		return &c
	},
	"City": func(agg NodesAggregate) interface{} {
		city := changeCase(agg.cities[rand.Intn(len(agg.cities))])
		return &city
	},
	"CityContains": func(agg NodesAggregate) interface{} {
		c := agg.cities[rand.Intn(len(agg.cities))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]
		return &c
	},
	"FarmName": func(agg NodesAggregate) interface{} {
		name := changeCase(agg.farmNames[rand.Intn(len(agg.farmNames))])
		return &name
	},
	"FarmNameContains": func(agg NodesAggregate) interface{} {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]
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
	"GpuDeviceName": func(agg NodesAggregate) interface{} {
		deviceNames := []string{"navi", "a", "hamada"}
		return &deviceNames[rand.Intn(len(deviceNames))]
	},
	"GpuVendorName": func(agg NodesAggregate) interface{} {
		vendorNames := []string{"advanced", "a", "hamada"}
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
}

func TestNode(t *testing.T) {
	t.Run("node pagination test", func(t *testing.T) {
		nodePaginationCheck(t, mockClient, gridProxyClient)
	})

	t.Run("single node test", func(t *testing.T) {
		singleNodeCheck(t, mockClient, gridProxyClient)
	})

	t.Run("node up test", func(t *testing.T) {
		f := proxytypes.NodeFilter{
			Status: &STATUS_UP,
		}

		l := proxytypes.Limit{
			Size:     999999999,
			Page:     1,
			RetCount: true,
		}

		want, wantCount, err := mockClient.Nodes(f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Nodes(f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})

	t.Run("node status test", func(t *testing.T) {
		for i := 1; i <= NODE_COUNT; i++ {
			if flip(.3) {
				want, err := mockClient.NodeStatus(uint32(i))
				require.NoError(t, err)

				got, err := gridProxyClient.NodeStatus(uint32(i))
				require.NoError(t, err)

				require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
			}
		}
	})

	t.Run("node stress test", func(t *testing.T) {
		agg := calcNodesAggregates(&data)
		for i := 0; i < NODE_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: true,
			}
			f, err := randomNodeFilter(&agg)
			require.NoError(t, err)

			want, wantCount, err := mockClient.Nodes(f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.Nodes(f, l)
			require.NoError(t, err)

			assert.Equal(t, wantCount, gotCount)

			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
		}
	})

	t.Run("node not found test", func(t *testing.T) {
		nodeID := 1000000000
		_, err := gridProxyClient.Node(uint32(nodeID))
		assert.Equal(t, err.Error(), ErrNodeNotFound.Error())
	})

	t.Run("nodes test without resources view", func(t *testing.T) {
		db := data.DB
		_, err := db.Exec("drop view nodes_resources_view ;")
		assert.NoError(t, err)

		singleNodeCheck(t, mockClient, gridProxyClient)
		assert.NoError(t, err)

		_, err = db.Exec("drop view nodes_resources_view ;")
		assert.NoError(t, err)

		nodePaginationCheck(t, mockClient, gridProxyClient)
	})

	t.Run("nodes test certification_type filter", func(t *testing.T) {
		certType := "Diy"
		nodes, _, err := gridProxyClient.Nodes(proxytypes.NodeFilter{CertificationType: &certType}, proxytypes.Limit{})
		require.NoError(t, err)

		for _, node := range nodes {
			assert.Equal(t, node.CertificationType, certType, "certification_type filter did not work")
		}

		notExistCertType := "noCert"
		nodes, _, err = gridProxyClient.Nodes(proxytypes.NodeFilter{CertificationType: &notExistCertType}, proxytypes.Limit{})
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("nodes test has_gpu filter", func(t *testing.T) {
		hasGPU := true
		nodes, _, err := gridProxyClient.Nodes(proxytypes.NodeFilter{HasGPU: &hasGPU}, proxytypes.Limit{})
		assert.NoError(t, err)

		for _, node := range nodes {
			assert.Equal(t, node.NumGPU, 1, "has_gpu filter did not work")
		}
	})

	t.Run("nodes test gpu vendor, device name filter", func(t *testing.T) {
		device := "navi"
		vendor := "advanced"
		nodes, _, err := gridProxyClient.Nodes(proxytypes.NodeFilter{GpuDeviceName: &device, GpuVendorName: &vendor}, proxytypes.Limit{})
		assert.NoError(t, err)

		localNodes, _, err := mockClient.Nodes(proxytypes.NodeFilter{GpuDeviceName: &device, GpuVendorName: &vendor}, proxytypes.Limit{})
		assert.NoError(t, err)

		assert.Equal(t, len(nodes), len(localNodes), "gpu_device_name, gpu_vendor_name filters did not work")
	})

	t.Run("nodes test gpu vendor, device id filter", func(t *testing.T) {
		device := "744c"
		vendor := "1002"
		nodes, _, err := gridProxyClient.Nodes(proxytypes.NodeFilter{GpuDeviceID: &device, GpuVendorID: &vendor}, proxytypes.Limit{})
		assert.NoError(t, err)

		localNodes, _, err := mockClient.Nodes(proxytypes.NodeFilter{GpuDeviceID: &device, GpuVendorID: &vendor}, proxytypes.Limit{})
		assert.NoError(t, err)

		assert.Equal(t, len(nodes), len(localNodes), "gpu_device_id, gpu_vendor_id filters did not work")
	})

	t.Run("nodes test gpu available", func(t *testing.T) {
		available := false
		nodes, _, err := gridProxyClient.Nodes(proxytypes.NodeFilter{GpuAvailable: &available}, proxytypes.Limit{})
		assert.NoError(t, err)

		localNodes, _, err := mockClient.Nodes(proxytypes.NodeFilter{GpuAvailable: &available}, proxytypes.Limit{})
		assert.NoError(t, err)

		assert.Equal(t, len(nodes), len(localNodes), "gpu_available filter did not work")
	})
}

// TestNodeFilter iterates over all NodeFilter fields, and for each one generates a random value, then runs a test between the mock client and the gridproxy client
func TestNodeFilter(t *testing.T) {
	f := proxytypes.NodeFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()
	l := proxytypes.Limit{
		Size:     9999999,
		Page:     1,
		RetCount: true,
	}

	agg := calcNodesAggregates(&data)

	for i := 0; i < v.NumField(); i++ {
		generator, ok := nodeFilterRandomValueGenerator[v.Type().Field(i).Name]
		require.True(t, ok, "Filter field %s has no random value generator", v.Type().Field(i).Name)

		randomFieldValue := generator(agg)

		if v.Field(i).Type().Kind() != reflect.Slice {
			v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
		}
		v.Field(i).Set(reflect.ValueOf(randomFieldValue))

		want, wantCount, err := mockClient.Nodes(f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Nodes(f, l)
		require.NoError(t, err, SerializeFilter(f))

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
	}
}

func singleNodeCheck(t *testing.T, localClient proxyclient.Client, proxyClient proxyclient.Client) {
	nodeID := rand.Intn(NODE_COUNT)
	want, err := mockClient.Node(uint32(nodeID))
	require.NoError(t, err)

	got, err := gridProxyClient.Node(uint32(nodeID))
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
}

func nodePaginationCheck(t *testing.T, localClient proxyclient.Client, proxyClient proxyclient.Client) {
	f := proxytypes.NodeFilter{
		Status: &STATUS_DOWN,
	}
	l := proxytypes.Limit{
		Size:     5,
		Page:     1,
		RetCount: true,
	}
	for ; ; l.Page++ {
		want, wantCount, err := mockClient.Nodes(f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Nodes(f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		if l.Page*l.Size >= uint64(wantCount) {
			break
		}
	}
}

func randomNodeFilter(agg *NodesAggregate) (proxytypes.NodeFilter, error) {
	f := proxytypes.NodeFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()

	for i := 0; i < v.NumField(); i++ {
		if rand.Float32() > .5 {
			_, ok := nodeFilterRandomValueGenerator[v.Type().Field(i).Name]
			if !ok {
				return proxytypes.NodeFilter{}, fmt.Errorf("Filter field %s has no random value generator", v.Type().Field(i).Name)
			}

			randomFieldValue := nodeFilterRandomValueGenerator[v.Type().Field(i).Name](*agg)
			if v.Field(i).Type().Kind() != reflect.Slice {
				v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
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
		free := calcFreeResources(total, data.NodeUsedResources[node.NodeID])
		res.maxFreeHRU = max(res.maxFreeHRU, free.HRU)
		res.maxFreeSRU = max(res.maxFreeSRU, free.SRU)
		res.maxFreeMRU = max(res.maxFreeMRU, free.MRU)
		res.freeMRUs = append(res.freeMRUs, free.MRU)
		res.freeSRUs = append(res.freeSRUs, free.SRU)
		res.freeHRUs = append(res.freeHRUs, free.HRU)

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
