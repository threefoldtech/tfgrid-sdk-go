package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"gorm.io/gorm"
)

var (
	nodesMRU               = make(map[uint64]uint64)
	nodesSRU               = make(map[uint64]uint64)
	nodesHRU               = make(map[uint64]uint64)
	nodeUP                 = make(map[uint64]bool)
	createdNodeContracts   = make([]uint64, 0)
	dedicatedFarms         = make(map[uint64]struct{})
	availableRentNodes     = make(map[uint64]struct{})
	availableRentNodesList = make([]uint64, 0)
	renter                 = make(map[uint64]uint64)
	billCnt                = 1
	contractCnt            = uint64(1)
	createBatchSize        = 5000
)

const (
	contractCreatedRatio = .1 // from devnet
	usedPublicIPsRatio   = .9
	nodeUpRatio          = .5
	nodeCount            = 1000
	farmCount            = 100
	normalUsers          = 2000
	publicIPCount        = 1000
	twinCount            = nodeCount + farmCount + normalUsers
	contractCount        = 3000
	// rentContractCount    = 100
	nameContractCount = 300

	maxContractHRU = 1024 * 1024 * 1024 * 300
	maxContractSRU = 1024 * 1024 * 1024 * 300
	maxContractMRU = 1024 * 1024 * 1024 * 16
	maxContractCRU = 16
	minContractHRU = 0
	minContractSRU = 1024 * 1024 * 256
	minContractMRU = 1024 * 1024 * 256
	minContractCRU = 1
)

func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile("./schema.sql")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		panic(err)
	}
	return nil
}

func generateTwins(db *gorm.DB) error {
	twins := make([]mock.Twin, 0, twinCount)
	for i := uint64(1); i <= twinCount; i++ {
		twins = append(twins, mock.Twin{
			ID:          fmt.Sprintf("twin-%d", i),
			AccountID:   fmt.Sprintf("account-id-%d", i),
			Relay:       fmt.Sprintf("relay-%d", i),
			PublicKey:   fmt.Sprintf("public-key-%d", i),
			TwinID:      i,
			GridVersion: 3,
		})
	}
	db.CreateInBatches(twins, createBatchSize)
	return db.Error
}

func generatePublicIPs(db *gorm.DB) error {
	ipsMap := map[uint64][]mock.PublicIp{}
	contractIPs := map[uint64]uint32{}
	totalIPs := map[uint64]uint32{}
	freeIPs := map[uint64]uint32{}
	for i := uint64(1); i <= publicIPCount; i++ {
		contract_id := uint64(0)
		if flip(usedPublicIPsRatio) {
			contract_id = createdNodeContracts[rnd(0, uint64(len(createdNodeContracts))-1)]
		}
		ip, err := randomIPv4()
		if err != nil {
			return fmt.Errorf("failed to create random ipv4: %w", err)
		}

		farmID := rnd(1, farmCount)
		publicIP := mock.PublicIp{
			ID:         fmt.Sprintf("public-ip-%d", i),
			Gateway:    ip.String(),
			IP:         IPv4Subnet(ip).String(),
			ContractID: contract_id,
			FarmID:     fmt.Sprintf("farm-%d", farmID),
		}
		ipsMap[farmID] = append(ipsMap[farmID], publicIP)

		if publicIP.ContractID != 0 {
			contractIPs[contract_id]++
		}

		if publicIP.ContractID == 0 {
			freeIPs[farmID]++
		}
		totalIPs[farmID]++

	}

	for farmID, ips := range ipsMap {
		b, err := json.Marshal(ips)
		if err != nil {
			return err
		}

		if err := db.Create(ips).Error; err != nil {
			return err
		}

		if err := db.Model(mock.FarmsCache{}).Where("farm_id = ?", farmID).Update("ips", string(b)).Error; err != nil {
			return err
		}

		if err := db.Model(mock.FarmsCache{}).Where("farm_id = ?", farmID).Updates(map[string]interface{}{
			"total_ips": gorm.Expr("total_ips + ?", totalIPs[farmID]),
			"free_ips":  gorm.Expr("free_ips + ?", freeIPs[farmID]),
		}).Error; err != nil {
			return err
		}
	}

	for contractID, ips := range contractIPs {
		if err := db.Model(mock.GenericContract{}).Where("contract_id = ?", contractID).Updates(map[string]interface{}{
			"number_of_public_i_ps": gorm.Expr("number_of_public_i_ps + ?", ips),
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

func generateFarms(db *gorm.DB) error {
	farms := make([]mock.Farm, 0, farmCount)
	cache := make([]mock.FarmsCache, 0, farmCount)
	for i := uint64(1); i <= farmCount; i++ {
		farm := mock.Farm{
			ID:              fmt.Sprintf("farm-%d", i),
			FarmID:          i,
			Name:            fmt.Sprintf("farm-name-%d", i),
			Certification:   "Diy",
			DedicatedFarm:   flip(.1),
			TwinID:          i,
			PricingPolicyID: 1,
			GridVersion:     3,
			StellarAddress:  "",
		}

		if farm.DedicatedFarm {
			dedicatedFarms[farm.FarmID] = struct{}{}
		}

		cache = append(cache, mock.FarmsCache{
			ID:       fmt.Sprintf("farm-%d", i),
			FarmID:   i,
			FreeIPs:  0,
			TotalIPs: 0,
			IPs:      "[]",
		})

		farms = append(farms, farm)
	}

	if res := db.Create(farms); res.Error != nil {
		return res.Error
	}

	if res := db.Create(cache); res.Error != nil {
		return res.Error
	}

	return nil
}

func generateContracts(db *gorm.DB) error {
	nodeContracts := make([]mock.GenericContract, 0, contractCount)
	contractResources := make([]mock.ContractResources, 0, contractCount)
	billReports := make([]*mock.ContractBillReport, 0)
	usedHRU := map[uint64]uint64{}
	usedMRU := map[uint64]uint64{}
	usedSRU := map[uint64]uint64{}
	nonDeletedContracts := uint32(0)
	for i := uint64(1); i <= contractCount; i++ {
		nodeID := rnd(1, nodeCount)
		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}
		if state != "Deleted" && (minContractHRU > nodesHRU[nodeID] || minContractMRU > nodesMRU[nodeID] || minContractSRU > nodesSRU[nodeID]) {
			i--
			continue
		}
		twinID := rnd(1100, 3100)
		if renter, ok := renter[nodeID]; ok {
			twinID = renter
		}
		if _, ok := availableRentNodes[nodeID]; ok {
			i--
			continue
		}
		contract := mock.GenericContract{
			ID:                fmt.Sprintf("node-contract-%d", contractCnt),
			TwinID:            twinID,
			ContractID:        contractCnt,
			State:             state,
			CreatedAt:         uint64(time.Now().Unix()),
			NodeID:            nodeID,
			DeploymentData:    fmt.Sprintf("deployment-data-%d", contractCnt),
			DeploymentHash:    fmt.Sprintf("deployment-hash-%d", contractCnt),
			NumberOfPublicIPs: 0,
			GridVersion:       3,
			ResourcesUsedID:   fmt.Sprintf("contract-resources-%d", contractCnt),
			Name:              "",
			Type:              "node",
		}
		nodeContracts = append(nodeContracts, contract)

		cru := rnd(minContractCRU, maxContractCRU)
		hru := rnd(minContractHRU, min(maxContractHRU, nodesHRU[nodeID]))
		sru := rnd(minContractSRU, min(maxContractSRU, nodesSRU[nodeID]))
		mru := rnd(minContractMRU, min(maxContractMRU, nodesMRU[nodeID]))
		contractRes := mock.ContractResources{
			ID:         fmt.Sprintf("contract-resources-%d", contractCnt),
			HRU:        hru,
			SRU:        sru,
			CRU:        cru,
			MRU:        mru,
			ContractID: fmt.Sprintf("node-contract-%d", contractCnt),
		}
		contractResources = append(contractResources, contractRes)

		if contract.State != "Deleted" {
			nodesHRU[nodeID] -= hru
			nodesSRU[nodeID] -= sru
			nodesMRU[nodeID] -= mru
			createdNodeContracts = append(createdNodeContracts, contractCnt)

			usedHRU[nodeID] += hru
			usedMRU[nodeID] += mru
			usedSRU[nodeID] += sru
			nonDeletedContracts++

		}

		billings := rnd(0, 10)
		for j := uint64(0); j < billings; j++ {
			bill := mock.ContractBillReport{
				ID:               fmt.Sprintf("contract-bill-report-%d", billCnt),
				ContractID:       contractCnt,
				DiscountReceived: "Default",
				AmountBilled:     rnd(0, 100000),
				Timestamp:        uint64(time.Now().UnixNano()),
			}
			billCnt++

			billReports = append(billReports, &bill)
		}
		contractCnt++
	}

	if err := db.CreateInBatches(&nodeContracts, createBatchSize).Error; err != nil {
		return err
	}

	if err := db.CreateInBatches(&contractResources, createBatchSize).Error; err != nil {
		return err
	}

	if err := db.CreateInBatches(billReports, createBatchSize).Error; err != nil {
		return err
	}

	for nodeID, hru := range usedHRU {
		sru := usedSRU[nodeID]
		mru := usedMRU[nodeID]
		if err := db.Model(mock.NodesCache{}).Where("node_id = ?", nodeID).Updates(map[string]interface{}{
			"free_sru":       gorm.Expr("free_sru - ?", sru),
			"free_mru":       gorm.Expr("free_mru - ?", mru),
			"free_hru":       gorm.Expr("free_hru - ?", hru),
			"node_contracts": gorm.Expr("node_contracts + ?", nonDeletedContracts),
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
func generateNameContracts(db *gorm.DB) error {
	nameContracts := make([]mock.GenericContract, 0, nameContractCount)
	billReports := make([]mock.ContractBillReport, 0)
	for i := uint64(1); i <= nameContractCount; i++ {
		nodeID := rnd(1, nodeCount)
		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}
		twinID := rnd(1100, 3100)
		if renter, ok := renter[nodeID]; ok {
			twinID = renter
		}
		if _, ok := availableRentNodes[nodeID]; ok {
			i--
			continue
		}
		contract := mock.GenericContract{
			ID:          fmt.Sprintf("node-contract-%d", contractCnt),
			TwinID:      twinID,
			ContractID:  contractCnt,
			State:       state,
			CreatedAt:   uint64(time.Now().Unix()),
			GridVersion: 3,
			Name:        uuid.NewString(),
			Type:        "name",
		}
		nameContracts = append(nameContracts, contract)

		billings := rnd(0, 10)
		for j := uint64(0); j < billings; j++ {
			billing := mock.ContractBillReport{
				ID:               fmt.Sprintf("contract-bill-report-%d", billCnt),
				ContractID:       contractCnt,
				DiscountReceived: "Default",
				AmountBilled:     rnd(0, 100000),
				Timestamp:        uint64(time.Now().UnixNano()),
			}
			billCnt++
			billReports = append(billReports, billing)
		}
		contractCnt++
	}

	if err := db.Create(nameContracts).Error; err != nil {
		return err
	}

	if err := db.Create(billReports).Error; err != nil {
		return err
	}

	return nil
}
func generateRentContracts(db *gorm.DB) error {
	rentContracts := make([]mock.GenericContract, 0, len(availableRentNodesList))
	billReports := make([]mock.ContractBillReport, 0)
	for _, nodeID := range availableRentNodesList {
		delete(availableRentNodes, nodeID)
		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(0.9) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}
		contract := mock.GenericContract{
			ID:          fmt.Sprintf("rent-contract-%d", contractCnt),
			TwinID:      rnd(1100, 3100),
			ContractID:  contractCnt,
			State:       state,
			CreatedAt:   uint64(time.Now().Unix()),
			NodeID:      nodeID,
			GridVersion: 3,
			Type:        "rent",
		}
		if state != "Deleted" {
			renter[nodeID] = contract.TwinID
			if err := db.Model(mock.NodesCache{}).Where("node_id = ?", nodeID).Updates(map[string]interface{}{
				"renter":           contract.TwinID,
				"rent_contract_id": contract.ContractID,
			}).Error; err != nil {
				return err
			}
		}
		rentContracts = append(rentContracts, contract)

		billings := rnd(0, 10)
		for j := uint64(0); j < billings; j++ {
			billing := mock.ContractBillReport{
				ID:               fmt.Sprintf("contract-bill-report-%d", billCnt),
				ContractID:       contractCnt,
				DiscountReceived: "Default",
				AmountBilled:     rnd(0, 100000),
				Timestamp:        uint64(time.Now().UnixNano()),
			}

			billCnt++
			billReports = append(billReports, billing)
		}
		contractCnt++
	}

	if err := db.Create(rentContracts).Error; err != nil {
		return err
	}

	if err := db.Create(&billReports).Error; err != nil {
		return err
	}

	return nil
}

func generateNodes(db *gorm.DB) error {
	const NodeCount = 1000
	powerState := []string{"Up", "Down"}
	locations := make([]mock.Location, 0, NodeCount)
	nodes := make([]mock.Node, 0, NodeCount)
	caches := make([]mock.NodesCache, 0, NodeCount)
	publicConfigs := make([]mock.PublicConfig, 0)

	for i := uint64(1); i <= NodeCount; i++ {
		mru := rnd(4, 256) * 1024 * 1024 * 1024
		hru := rnd(100, 30*1024) * 1024 * 1024 * 1024 // 100GB -> 30TB
		sru := rnd(200, 30*1024) * 1024 * 1024 * 1024 // 100GB -> 30TB
		cru := rnd(4, 128)
		up := flip(nodeUpRatio)
		updatedAt := time.Now().Unix() - int64(rnd(60*40*3, 60*60*24*30*12))

		if up {
			updatedAt = time.Now().Unix() - int64(rnd(0, 60*40*1))
		}
		nodesMRU[i] = mru - max(2*uint64(gridtypes.Gigabyte), mru/10)
		nodesSRU[i] = sru - 100*uint64(gridtypes.Gigabyte)
		nodesHRU[i] = hru
		nodeUP[i] = up
		location := mock.Location{
			ID:        fmt.Sprintf("location-%d", i),
			Longitude: fmt.Sprintf("location--long-%d", i),
			Latitude:  fmt.Sprintf("location-lat-%d", i),
		}
		locations = append(locations, location)

		node := mock.Node{
			ID:              fmt.Sprintf("node-%d", i),
			LocationID:      fmt.Sprintf("location-%d", i),
			NodeID:          i,
			FarmID:          i%100 + 1,
			TwinID:          i + 100 + 1,
			Country:         "Belgium",
			City:            "Unknown",
			Uptime:          1000,
			UpdatedAt:       uint64(updatedAt),
			Created:         uint64(time.Now().Unix()),
			CreatedAt:       uint64(time.Now().Unix()),
			FarmingPolicyID: 1,
			GridVersion:     3,
			Certification:   "Diy",
			Secure:          false,
			Virtualized:     false,
			SerialNumber:    "",
			Power: mock.NodePower{
				State:  powerState[rand.Intn(len(powerState))],
				Target: powerState[rand.Intn(len(powerState))],
			},
			ExtraFee: 0,
			TotalHRU: hru,
			TotalSRU: sru,
			TotalMRU: mru,
			TotalCRU: cru,
		}
		nodes = append(nodes, node)

		isDedicatedFarm := true
		if _, ok := dedicatedFarms[node.FarmID]; !ok {
			isDedicatedFarm = false
		}

		cache := mock.NodesCache{
			ID:             fmt.Sprintf("node-%d", i),
			NodeID:         i,
			NodeTwinID:     node.TwinID,
			FreeHRU:        hru,
			FreeSRU:        sru - 107374182400,
			FreeMRU:        mru - max(mru/10, 2147483648),
			FreeCRU:        cru,
			Renter:         0,
			RentContractID: 0,
			NodeContracts:  0,
			FarmID:         node.FarmID,
			DedicatedFarm:  isDedicatedFarm,
			FreeGPUs:       0,
		}
		caches = append(caches, cache)

		if _, ok := dedicatedFarms[node.FarmID]; ok {
			availableRentNodes[i] = struct{}{}
			availableRentNodesList = append(availableRentNodesList, i)
		}

		if flip(.1) {
			publicConfigs = append(publicConfigs, mock.PublicConfig{
				ID:     fmt.Sprintf("public-config-%d", i),
				IPv4:   "185.16.5.2/24",
				GW4:    "185.16.5.2",
				IPv6:   "::1/64",
				GW6:    "::1",
				Domain: "hamada.com",
				NodeID: fmt.Sprintf("node-%d", i),
			})
		}

	}

	if err := db.Create(locations).Error; err != nil {
		return err
	}

	if err := db.Create(nodes).Error; err != nil {
		return err
	}

	if err := db.Create(caches).Error; err != nil {
		return err
	}

	if err := db.Create(publicConfigs).Error; err != nil {
		return err
	}

	return nil
}

func generateNodeGPUs(db *gorm.DB) error {
	for i := 0; i <= 10; i++ {
		g := mock.NodeGPU{
			NodeTwinID: uint64(i + 100),
			Vendor:     "Advanced Micro Devices, Inc. [AMD/ATI]",
			Device:     "Navi 31 [Radeon RX 7900 XT/7900 XTX",
			Contract:   i % 2,
			ID:         "0000:0e:00.0/1002/744c",
		}

		if err := db.Create(&g).Error; err != nil {
			return err
		}

		if g.Contract != 0 {
			if err := db.Model(mock.NodesCache{}).Where("node_twin_id = ?", g.NodeTwinID).Updates(map[string]interface{}{
				"free_gpus": gorm.Expr("free_gpus + 1"),
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func generateData(gormDB *gorm.DB) error {
	if err := generateTwins(gormDB); err != nil {
		return err
	}
	if err := generateFarms(gormDB); err != nil {
		return err
	}
	if err := generateNodes(gormDB); err != nil {
		return err
	}
	if err := generateRentContracts(gormDB); err != nil {
		return err
	}
	if err := generateContracts(gormDB); err != nil {
		return err
	}
	if err := generateNameContracts(gormDB); err != nil {
		return err
	}
	if err := generatePublicIPs(gormDB); err != nil {
		return err
	}
	if err := generateNodeGPUs(gormDB); err != nil {
		return err
	}
	return nil
}
