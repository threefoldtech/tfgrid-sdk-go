package main

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"sync"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var (
	statusUp  = "up"
	minRootfs = *convertGBToBytes(2)
	twn       = uint64(5432)
)

var nodeFilter = types.NodeFilter{
	Status:       &statusUp,
	AvailableFor: &twn,
	FreeSRU:      convertGBToBytes(15),
	FreeMRU:      convertGBToBytes(1),
}

func setup() ([]deployer.TFPluginClient, error) {
	// mainnet
	network := "main"
	mnemonics := []string{"<mnemonics goes here>"}

	fmt.Printf("network: %s\n", network)
	fmt.Printf("mnemonics: %s\n", mnemonics)

	tfPluginClients := []deployer.TFPluginClient{}
	for _, mnemonic := range mnemonics {
		tfPluginClient, err := deployer.NewTFPluginClient(
			mnemonic, "sr25519", network, "", "", "", 30, false)
		if err != nil {
			return []deployer.TFPluginClient{}, err
		}

		tfPluginClients = append(tfPluginClients, tfPluginClient)
	}
	return tfPluginClients, nil
}

func getNodes(ctx context.Context, tfPluginClient deployer.TFPluginClient, totalVMCount int) ([]uint32, error) {
	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		nodeFilter,
		[]uint64{*convertGBToBytes(5)},
		nil,
		[]uint64{minRootfs},
		uint64(totalVMCount+200),
	)
	if err != nil {
		return []uint32{}, err
	}

	if len(nodes) < totalVMCount {
		return []uint32{}, fmt.Errorf("no available nodes found, Only found %d\n", len(nodes))
	}

	nodesIDs := make([]uint32, len(nodes))
	for i, node := range nodes {
		nodesIDs[i] = uint32(node.NodeID)
	}

	nodesIDs = getNodesWithValidFileSystem(nodesIDs, tfPluginClient, ctx)

	if len(nodesIDs) < totalVMCount {
		return []uint32{}, fmt.Errorf("no available nodes found, Only found %d\n", len(nodesIDs))
	}

	fmt.Printf("Found free %d nodes!\n", len(nodes))
	return nodesIDs[:totalVMCount], nil
}

func getReachableNodes(nodes []uint32, tfPluginClient deployer.TFPluginClient, ctx context.Context) []uint32 {
	nodesIDs := []uint32{}
	var wg sync.WaitGroup
	var lock sync.Mutex

	// skip any node that can't be reached
	for _, node := range nodes {
		wg.Add(1)

		go func(nodeID uint32) {
			defer wg.Done()

			client, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID)
			if err != nil {
				fmt.Printf("failed to get node '%d' client\n", nodeID)
				return
			}
			err = client.IsNodeUp(ctx)
			if err != nil {
				fmt.Printf("failed to ping node '%d'\n", nodeID)
				return
			}

			lock.Lock()
			nodesIDs = append(nodesIDs, nodeID)
			lock.Unlock()
		}(node)
	}
	wg.Wait()
	return nodesIDs
}

func getNodesWithValidFileSystem(nodes []uint32, tfPluginClient deployer.TFPluginClient, ctx context.Context) []uint32 {
	brokenNodes := []int{
		958, 1116, 721, 1097, 1107, 2597, 3263, 1118, 1126, 1226, 1398,
		1361, 1334, 1335, 1941, 1744, 1090, 1732, 1719, 1296,
	}
	var validNodes []uint32

	for _, node := range nodes {
		if !slices.Contains(brokenNodes, int(node)) {
			validNodes = append(validNodes, node)
		}
	}
	return validNodes
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func convertGBToBytes(gb uint64) *uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return &bytes
}
