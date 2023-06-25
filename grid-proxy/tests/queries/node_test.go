package test

import (
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
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/mock_client"
)

type NodesAggregate struct {
	countries []string
	cities    []string
	farmNames []string
	farmIDs   []uint32
	freeMRUs  []uint64
	freeSRUs  []uint64
	freeHRUs  []uint64

	maxFreeMRU  uint64
	maxFreeSRU  uint64
	maxFreeHRU  uint64
	maxFreeIPs  uint64
	nodeRenters []uint32
	twins       []uint32

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

func TestNode(t *testing.T) {
	t.Run("node pagination test", func(t *testing.T) {
		t.Parallel()

		nodePaginationCheck(t, mockClient, proxyClient)
	})

	t.Run("single node test", func(t *testing.T) {
		t.Parallel()

		singleNodeCheck(t, mockClient, proxyClient)
	})

	t.Run("node up test", func(t *testing.T) {
		t.Parallel()

		f := proxytypes.NodeFilter{
			Status: &STATUS_UP,
		}

		l := proxytypes.Limit{
			Size:     999999999,
			Page:     1,
			RetCount: true,
		}

		localNodes, localCount, err := mockClient.Nodes(f, l)
		assert.NoError(t, err)
		remoteNodes, remoteCount, err := proxyClient.Nodes(f, l)
		assert.NoError(t, err)

		require.Equal(t, localCount, remoteCount, serializeFilter(f))
		require.Equal(t, len(localNodes), len(remoteNodes), serializeFilter(f))

		require.True(t, reflect.DeepEqual(localNodes, remoteNodes), serializeFilter(f), cmp.Diff(localNodes, remoteNodes))
	})

	t.Run("node status test", func(t *testing.T) {
		t.Parallel()

		for i := 1; i <= NODE_COUNT; i++ {
			if flip(.3) {
				localNodeStatus, err := mockClient.NodeStatus(uint32(i))
				assert.NoError(t, err)
				remoteNodeStatus, err := proxyClient.NodeStatus(uint32(i))
				assert.NoError(t, err)

				require.True(t, reflect.DeepEqual(localNodeStatus, remoteNodeStatus), cmp.Diff(localNodeStatus, remoteNodeStatus))
			}
		}
	})

	t.Run("node stress test", func(t *testing.T) {
		t.Parallel()

		agg, err := calcNodesAggregates(&dbData)
		assert.NoError(t, err)

		for i := 0; i < NODE_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: false,
			}
			f := randomNodeFilter(&agg)

			localNodes, _, err := mockClient.Nodes(f, l)
			assert.NoError(t, err)

			remoteNodes, _, err := proxyClient.Nodes(f, l)
			assert.NoError(t, err)

			assert.Equal(t, len(localNodes), len(remoteNodes), serializeFilter(f))

			require.True(t, reflect.DeepEqual(localNodes, remoteNodes), serializeFilter(f), cmp.Diff(localNodes, remoteNodes))
		}
	})

	t.Run("node not found test", func(t *testing.T) {
		t.Parallel()

		nodeID := 1000000000
		_, err := proxyClient.Node(uint32(nodeID))
		assert.Equal(t, err.Error(), ErrNodeNotFound.Error())
	})

	t.Run("nodes test without resources view", func(t *testing.T) {
		// this can't be run in parallel because it deletes a view, hence will make other tests fail

		db := dbData.DB
		_, err := db.Exec("drop view nodes_resources_view ;")
		assert.NoError(t, err)

		singleNodeCheck(t, mockClient, proxyClient)

		_, err = db.Exec("drop view nodes_resources_view ;")
		assert.NoError(t, err)

		nodePaginationCheck(t, mockClient, proxyClient)
	})

}

func singleNodeCheck(t *testing.T, localClient proxyclient.Client, proxyClient proxyclient.Client) {
	nodeID := rand.Intn(NODE_COUNT)

	localNode, err := localClient.Node(uint32(nodeID))
	assert.NoError(t, err)

	remoteNode, err := proxyClient.Node(uint32(nodeID))
	assert.NoError(t, err)

	assert.True(t, reflect.DeepEqual(localNode, remoteNode), cmp.Diff(localNode, remoteNode))
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
		localNodes, localCount, err := localClient.Nodes(f, l)
		assert.NoError(t, err)

		remoteNodes, remoteCount, err := proxyClient.Nodes(f, l)
		assert.NoError(t, err)

		require.Equal(t, localCount, remoteCount, serializeFilter(f))
		require.Equal(t, len(localNodes), len(remoteNodes), serializeFilter(f))

		require.True(t, reflect.DeepEqual(localNodes, remoteNodes), serializeFilter(f), cmp.Diff(localNodes, remoteNodes))

		if l.Page*l.Size >= uint64(localCount) {
			break
		}
	}
}

func calcNodesAggregates(data *mock.DBData) (NodesAggregate, error) {
	res := NodesAggregate{}
	cities := make(map[string]struct{})
	countries := make(map[string]struct{})
	for _, node := range data.Nodes {
		cities[node.City] = struct{}{}
		countries[node.Country] = struct{}{}
		total := data.NodeTotalResources[node.NodeID]
		free, err := mock.CalculateFreeResources(total, data.NodeUsedResources[node.NodeID])
		if err != nil {
			return NodesAggregate{}, err
		}

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
		res.countries = append(res.cities, country)
	}
	for _, farm := range data.Farms {
		res.farmNames = append(res.farmNames, farm.Name)
		res.farmIDs = append(res.farmIDs, farm.FarmID)
	}

	farmIPs := make(map[uint32]uint64)
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

	return res, nil
}

func randomNodeFilter(agg *NodesAggregate) proxytypes.NodeFilter {
	var f proxytypes.NodeFilter

	f.Status = nodeRandStatus(agg)
	f.FreeMRU = nodeRandFreeMRU(agg)
	f.FreeHRU = nodeRandFreeHRU(agg)
	f.FreeSRU = nodeRandFreeSRU(agg)
	f.TotalCRU = nodeRandTotalCRU(agg)
	f.TotalMRU = nodeRandTotalMRU(agg)
	f.TotalSRU = nodeRandTotalSRU(agg)
	f.TotalHRU = nodeRandTotalHRU(agg)
	f.Country = nodeRandCountry(agg)
	f.CountryContains = nodeRandCountryContains(agg)
	f.City = nodeRandCity(agg)
	f.CityContains = nodeRandCityContains(agg)
	f.FarmName = nodeRandFarmName(agg)
	f.FarmNameContains = nodeRandFarmNameContains(agg)
	f.FarmIDs = nodeRandFarmIDs(agg)
	f.FreeIPs = nodeRandFreeIPs(agg)
	f.IPv4 = nodeRandIPv4(agg)
	f.IPv6 = nodeRandIPv6(agg)
	f.Domain = nodeRandDomain(agg)
	f.NodeID = nodeRandNodeID(agg)
	f.TwinID = nodeRandTwinID(agg)
	f.Rentable = nodeRandRentable(agg)
	f.RentedBy = nodeRandRentedBy(agg)
	f.AvailableFor = nodeRandAvailableFor(agg)
	f.Rented = nodeRandRented(agg)
	f.CertificationType = nodeRandCertificationType(agg)
	f.HasGPU = nodeRandHasGPU(agg)

	return f
}

func nodeRandStatus(agg *NodesAggregate) *string {
	if flip(.5) {
		status := "down"
		if flip(.5) {
			status = "up"
		}
		return &status
	}

	return nil
}

func nodeRandFreeMRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.freeMRUs[rand.Intn(len(agg.freeMRUs))]
			return &c
		} else {
			return rndref(0, agg.maxFreeMRU)
		}
	}

	return nil
}

func nodeRandFreeHRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.freeHRUs[rand.Intn(len(agg.freeHRUs))]
			return &c
		} else {
			return rndref(0, agg.maxFreeHRU)
		}
	}

	return nil
}

func nodeRandFreeSRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.freeSRUs[rand.Intn(len(agg.freeSRUs))]
			return &c
		} else {
			return rndref(0, agg.maxFreeSRU)
		}
	}

	return nil
}

func nodeRandTotalCRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.totalCRUs[rand.Intn(len(agg.totalCRUs))]
			return &c
		} else {
			return rndref(0, agg.maxTotalCRU)
		}
	}

	return nil
}
func nodeRandTotalMRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.totalMRUs[rand.Intn(len(agg.totalMRUs))]
			return &c
		} else {
			return rndref(0, agg.maxTotalMRU)
		}
	}

	return nil
}
func nodeRandTotalSRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.totalSRUs[rand.Intn(len(agg.totalSRUs))]
			return &c
		} else {
			return rndref(0, agg.maxTotalSRU)
		}
	}

	return nil
}

func nodeRandTotalHRU(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		if flip(.1) {
			c := agg.totalHRUs[rand.Intn(len(agg.totalHRUs))]
			return &c
		} else {
			return rndref(0, agg.maxTotalHRU)
		}
	}

	return nil
}

func nodeRandCountry(agg *NodesAggregate) *string {
	if flip(.5) {
		c := agg.countries[rand.Intn(len(agg.countries))]
		v := changeCase(c)
		return &v
	}

	return nil
}

func nodeRandCountryContains(agg *NodesAggregate) *string {
	if flip(.5) {
		c := agg.countries[rand.Intn(len(agg.countries))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]

		return &c
	}

	return nil
}

func nodeRandCity(agg *NodesAggregate) *string {
	if flip(.5) {
		c := agg.cities[rand.Intn(len(agg.cities))]
		v := changeCase(c)
		return &v
	}

	return nil
}

func nodeRandCityContains(agg *NodesAggregate) *string {
	if flip(.5) {
		c := agg.cities[rand.Intn(len(agg.cities))]
		a, b := rand.Intn(len(c)), rand.Intn(len(c))
		if a > b {
			a, b = b, a
		}
		c = c[a : b+1]

		return &c
	}

	return nil
}

func nodeRandFarmName(agg *NodesAggregate) *string {
	if flip(.5) {
		c := agg.farmNames[rand.Intn(len(agg.farmNames))]
		v := changeCase(c)
		return &v
	}

	return nil
}

func nodeRandFarmNameContains(agg *NodesAggregate) *string {
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

func nodeRandFarmIDs(agg *NodesAggregate) []uint32 {
	if flip(.5) {
		ids := []uint32{}
		for _, id := range agg.farmIDs {
			if flip(float32(min(3, uint64(len(agg.farmIDs)))) / float32(len(agg.farmIDs))) {
				ids = append(ids, uint32(id))
			}
		}
		return ids
	}

	return nil
}

func nodeRandFreeIPs(agg *NodesAggregate) *uint64 {
	if flip(.5) {
		return rndref(0, agg.maxFreeIPs)
	}

	return nil
}

func nodeRandIPv4(agg *NodesAggregate) *bool {
	if flip(.5) {
		v := true
		return &v
	}

	return nil
}

func nodeRandIPv6(agg *NodesAggregate) *bool {
	if flip(.5) {
		v := true
		return &v
	}

	return nil
}

func nodeRandDomain(agg *NodesAggregate) *bool {
	if flip(.5) {
		v := true
		return &v
	}

	return nil
}

func nodeRandNodeID(agg *NodesAggregate) *uint32 {
	if flip(.5) {
		v := uint32(rand.Intn(1100)) // 1000 is the total nodes + 100 for non-existed cases
		return &v
	}

	return nil
}

func nodeRandTwinID(agg *NodesAggregate) *uint32 {
	if flip(.5) {
		v := uint32(rand.Intn(3500))
		return &v
	}

	return nil
}

func nodeRandRentable(agg *NodesAggregate) *bool {
	if flip(.5) {
		v := true
		return &v
	}

	return nil
}

func nodeRandRentedBy(agg *NodesAggregate) *uint32 {
	if flip(.5) {
		c := agg.twins[rand.Intn(len(agg.twins))]
		if flip(.9) && len(agg.nodeRenters) != 0 {
			c = agg.nodeRenters[rand.Intn(len(agg.nodeRenters))]
		}
		return &c
	}

	return nil
}

func nodeRandAvailableFor(agg *NodesAggregate) *uint32 {
	if flip(.5) {
		c := agg.twins[rand.Intn(len(agg.twins))]
		if flip(.1) && len(agg.nodeRenters) != 0 {
			c = agg.nodeRenters[rand.Intn(len(agg.nodeRenters))]
		}
		return &c
	}

	return nil
}

func nodeRandRented(agg *NodesAggregate) *bool {
	if flip(.5) {
		v := true
		return &v
	}

	return nil
}

func nodeRandCertificationType(agg *NodesAggregate) *string {
	if flip(.5) {
		v := "Diy"
		if flip(.5) {
			v = "noCert"
		}

		return &v
	}

	return nil
}

func nodeRandHasGPU(agg *NodesAggregate) *bool {
	if flip(.5) {
		v := true
		return &v
	}

	return nil
}
