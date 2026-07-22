package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"gorm.io/gorm/logger"
)

// TestPostgresDatabase_GetNode tests the GetNode function.
func TestPostgresDatabase_GetNode(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)

	if err != nil {
		t.Skipf("Can't connect to testdb %e", err)
	}
	ctx := context.Background()

	t.Run("Node exists", func(t *testing.T) {
		nodeID := uint32(118) // Node ID from the fixture data

		node, err := dbTest.GetNode(ctx, nodeID)

		assert.NoError(t, err)
		assert.Equal(t, "node-118", node.ID)
		assert.Equal(t, int64(118), node.NodeID)
		assert.Equal(t, int64(52), node.FarmID)
		assert.Equal(t, "United States", node.Country)
		assert.Equal(t, "Los Angeles", node.City)
		assert.Equal(t, int64(1000), node.Uptime)
		assert.Equal(t, int64(1730904704), node.Created)
		assert.Equal(t, "Diy", node.Certification)
	})

	t.Run("Node does not exist", func(t *testing.T) {
		nonExistentNodeID := uint32(99999) // Node ID that doesn’t exist in the fixture data

		node, err := dbTest.GetNode(ctx, nonExistentNodeID)

		assert.ErrorIs(t, err, ErrNodeNotFound)
		assert.Equal(t, Node{}, node)
	})
}

// TestPostgresDatabase_GetFarm tests the GetFarm function.
func TestPostgresDatabase_GetFarm(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Farm exists Dedicated", func(t *testing.T) {
		farmID := uint32(3) // Farm ID from the fixture data

		farm, err := dbTest.GetFarm(ctx, farmID)

		assert.NoError(t, err)
		assert.Equal(t, int(3), farm.FarmID)
		assert.Equal(t, "farm-name-3", farm.Name)
		assert.Equal(t, int(5), farm.TwinID)
		assert.Equal(t, int(1), farm.PricingPolicyID)
		//assert.True(t, farm.Dedicated)
		assert.Equal(t, "Diy", farm.Certification)
	})

	t.Run("Farm exists not Dedicated", func(t *testing.T) {
		farmID := uint32(4) // Farm ID from the fixture data

		farm, err := dbTest.GetFarm(ctx, farmID)

		assert.NoError(t, err)
		assert.Equal(t, int(4), farm.FarmID)
		assert.Equal(t, "farm-name-4", farm.Name)
		assert.Equal(t, int(6), farm.TwinID)
		assert.Equal(t, int(1), farm.PricingPolicyID)
		//assert.False(t, farm.Dedicated)
		assert.Equal(t, "Diy", farm.Certification)
	})

	t.Run("Farm does not exist", func(t *testing.T) {
		nonExistentFarmID := uint32(999) // Farm ID that doesn’t exist in the fixture data

		farm, _ := dbTest.GetFarm(ctx, nonExistentFarmID)

		//assert.ErrorIs(t, err, ErrFarmNotFound)
		assert.Equal(t, Farm{}, farm)
	})
}

// TestPostgresDatabase_GetNodes tests the GetNodes function.
func TestPostgresDatabase_GetNodes(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve all nodes", func(t *testing.T) {
		filter := types.NodeFilter{}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		nodes, count, err := dbTest.GetNodes(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(600), count)
		assert.NotEmpty(t, nodes)
	})

	t.Run("Filter by farm id", func(t *testing.T) {
		farmId := []uint64{100}
		filter := types.NodeFilter{
			FarmIDs: farmId,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		_, count, err := dbTest.GetNodes(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(3), count)
	})

	t.Run("Filter by farm id and add limit to 2 nodes", func(t *testing.T) {
		farmId := []uint64{100}
		filter := types.NodeFilter{
			FarmIDs: farmId,
		}
		limit := types.Limit{
			Size: 2,
			Page: 1,
		}

		nodes, _, err := dbTest.GetNodes(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, 2, len(nodes))
	})

	t.Run("Filter by country presence", func(t *testing.T) {
		country := "Egypt"
		filter := types.NodeFilter{
			Country: &country,
		}
		limit := types.Limit{
			Size: 999999999999,
			Page: 1,
		}

		nodes, _, err := dbTest.GetNodes(ctx, filter, limit)

		assert.NoError(t, err)
		for _, node := range nodes {
			assert.Equal(t, country, node.Country)
		}
	})
}

// TestPostgresDatabase_GetNodes tests the GetFarms function.
func TestPostgresDatabase_GetFarms(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve all farms", func(t *testing.T) {
		filter := types.FarmFilter{}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		nodes, count, err := dbTest.GetFarms(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(100), count)
		assert.NotEmpty(t, nodes)
	})

}

// TestPostgresDatabase_GetTwins tests the GetTwins function.
func TestPostgresDatabase_GetTwins(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve all twins", func(t *testing.T) {
		filter := types.TwinFilter{}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		_, count, err := dbTest.GetTwins(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(1300), count)
	})

	t.Run("Retrieve twin with twin id", func(t *testing.T) {
		twinId := uint64(1)
		filter := types.TwinFilter{
			TwinID: &twinId,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		twin, count, err := dbTest.GetTwins(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(1), count)
		assert.Equal(t, twinId, twinId, twin[0].TwinID)
		assert.Equal(t, "account-id-1", twin[0].AccountID)
		assert.Equal(t, "relay-1", twin[0].Relay)
		assert.Equal(t, "public-key-1", twin[0].PublicKey)
	})
}

// TestPostgresDatabase_GetContracts tests the GetContracts function.
func TestPostgresDatabase_GetContracts(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve all contracts", func(t *testing.T) {
		filter := types.ContractFilter{}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		_, count, err := dbTest.GetContracts(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(61), count)
	})

	t.Run("Retrieve contract with id", func(t *testing.T) {
		id := uint64(37)
		filter := types.ContractFilter{
			ContractID: &id,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		contract, count, err := dbTest.GetContracts(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(1), count)
		assert.Equal(t, uint(37), contract[0].ContractID)
		assert.Equal(t, uint(303), contract[0].NodeID)
		assert.Equal(t, uint(1), contract[0].NumberOfPublicIps)
	})

	t.Run("Retrieve contract with id", func(t *testing.T) {
		id := uint64(52)
		filter := types.ContractFilter{
			ContractID: &id,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		contract, count, err := dbTest.GetContracts(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(1), count)
		assert.Equal(t, uint(52), contract[0].ContractID)
		assert.Equal(t, "401c4f28-4a84-47ff-9a91-50426305db00", contract[0].Name)
	})

	t.Run("Retrieve contract with twin id", func(t *testing.T) {
		id := uint64(2705)
		filter := types.ContractFilter{
			TwinID: &id,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		contract, count, err := dbTest.GetContracts(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(1), count)
		assert.Equal(t, uint(52), contract[0].ContractID)
		assert.Equal(t, "401c4f28-4a84-47ff-9a91-50426305db00", contract[0].Name)
	})

	t.Run("Retrieve contract with type Created", func(t *testing.T) {
		contractType := []string{"Created"}
		filter := types.ContractFilter{
			State: contractType,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}

		contracts, count, err := dbTest.GetContracts(ctx, filter, limit)

		assert.NoError(t, err)
		//name contracts + node contracts
		assert.Equal(t, uint(29), count)
		for _, contract := range contracts {
			assert.Equal(t, contractType[0], contract.State)
		}
	})

}

// TestPostgresDatabase_GetContract tests the GetContract function.
func TestPostgresDatabase_GetContract(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve contract with id", func(t *testing.T) {
		id := uint32(37)
		contract, err := dbTest.GetContract(ctx, id)

		assert.NoError(t, err)
		assert.NotNil(t, contract)
		assert.Equal(t, uint(37), contract.ContractID)
		assert.Equal(t, uint(303), contract.NodeID)
		assert.Equal(t, uint(1), contract.NumberOfPublicIps)
	})

	t.Run("Retrieve contract with id", func(t *testing.T) {
		id := uint32(52)
		contract, err := dbTest.GetContract(ctx, id)

		assert.NoError(t, err)
		assert.NotNil(t, contract)
		assert.Equal(t, uint(52), contract.ContractID)
		assert.Equal(t, "401c4f28-4a84-47ff-9a91-50426305db00", contract.Name)
	})

	t.Run("Retrieve contract with id not found", func(t *testing.T) {
		id := uint32(999)
		contract, err := dbTest.GetContract(ctx, id)

		assert.ErrorIs(t, err, ErrContractNotFound)
		assert.Equal(t, DBContract{}, contract)

	})

}

// TestPostgresDatabase_GetContractBills tests the GetContractBills function.
func TestPostgresDatabase_GetContractBills(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve contract bills with id", func(t *testing.T) {
		id := uint32(37)
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		contractBills, count, err := dbTest.GetContractBills(ctx, id, limit)

		assert.NoError(t, err)
		assert.NotNil(t, contractBills)
		assert.Equal(t, uint(9), count)
		//assert.Equal(t, uint64(37), contractBills[0].ContractId)
	})

	t.Run("Retrieve contract bills with id not found", func(t *testing.T) {
		id := uint32(999)
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		_, count, err := dbTest.GetContractBills(ctx, id, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(0), count)
	})

}

// TestPostgresDatabase_GetContractsLatestBillReports tests the GetContractsLatestBillReports function.
func TestPostgresDatabase_GetContractsLatestBillReports(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve contract bills with id", func(t *testing.T) {
		id := []uint32{37}
		contractBills, err := dbTest.GetContractsLatestBillReports(ctx, id, 2)

		assert.NoError(t, err)
		assert.NotNil(t, contractBills)
		assert.Equal(t, 2, len(contractBills))
		//assert.Equal(t, uint64(37), contractBills[0].ContractId)
	})

}

// TestPostgresDatabase_GetContractsTotalBilledAmount tests the GetContractsTotalBilledAmount function.
func TestPostgresDatabase_GetContractsTotalBilledAmount(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve contract bills with one id", func(t *testing.T) {
		id := []uint32{37}
		totalBills, err := dbTest.GetContractsTotalBilledAmount(ctx, id)

		assert.NoError(t, err)
		assert.Equal(t, uint64(825687), totalBills)
		//assert.Equal(t, uint64(37), contractBills[0].ContractId)
	})

	t.Run("Retrieve contract bills with multiple ids", func(t *testing.T) {
		id := []uint32{1, 37}
		totalBills, err := dbTest.GetContractsTotalBilledAmount(ctx, id)

		assert.NoError(t, err)
		assert.Equal(t, uint64(176890+825687), totalBills)
	})

}

// TestPostgresDatabase_GetPublicIps tests the GetPublicIps function.
func TestPostgresDatabase_GetPublicIps(t *testing.T) {

	dbTest, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "testdb", 80, logger.Error)
	if err != nil {
		t.Skipf("Can't connect to testdb: %v", err)
	}
	ctx := context.Background()

	t.Run("Retrieve Public Ips with Ip address", func(t *testing.T) {
		ip := "181.66.234.17/24"
		filter := types.PublicIpFilter{
			Ip: &ip,
		}
		limit := types.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		publicIp, count, err := dbTest.GetPublicIps(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(1), count)
		assert.Equal(t, uint64(93), publicIp[0].FarmID)
		assert.Equal(t, uint64(16), publicIp[0].ContractID)
	})

	t.Run("Retrieve Public Ips with farm id", func(t *testing.T) {
		farmIds := []uint64{93}
		filter := types.PublicIpFilter{
			FarmIDs: farmIds,
		}
		limit := types.Limit{
			Size:      999999999999,
			Page:      1,
			RetCount:  true,
			SortBy:    "contract_id",
			SortOrder: types.SortOrderAsc,
		}
		publicIp, count, err := dbTest.GetPublicIps(ctx, filter, limit)

		assert.NoError(t, err)
		assert.Equal(t, uint(2), count)
		assert.Equal(t, uint64(93), publicIp[0].FarmID)
		assert.Equal(t, uint64(93), publicIp[1].FarmID)
		assert.Equal(t, uint64(6), publicIp[0].ContractID)
		assert.Equal(t, uint64(16), publicIp[1].ContractID)
	})

}
