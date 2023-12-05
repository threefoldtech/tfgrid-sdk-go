package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	totalVMCount = 50
	batchSize    = 25
)

func main() {
	tfPluginClients, err := setup()
	if err != nil {
		fmt.Println("failed to create new tfPluginClient: " + err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Minute)
	defer cancel()

	nodes := getNodes(ctx, tfPluginClients[0], totalVMCount)

	massSize := totalVMCount / len(tfPluginClients)
	var passed int
	var duration time.Duration

	var wg sync.WaitGroup
	var lock sync.Mutex

	// the actual deployment
	for i := 0; i < len(tfPluginClients); i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			nodesIDs := nodes[j*massSize : (j+1)*massSize]
			p, d := MassDeploy(ctx, tfPluginClients[j], nodesIDs)

			lock.Lock()
			passed += p
			duration = max(duration, d)
			lock.Unlock()
		}(i)
	}
	wg.Wait()

	fmt.Printf("deployment of %d vms passed\n", passed)
	fmt.Printf("deployment of %d vms took %s\n", totalVMCount, duration)
}

func MassDeploy(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodes []uint32) (int, time.Duration) {
	deploymentStartTime := time.Now()
	allNetworkDeployments := []*workloads.ZNet{}
	allVmDeployments := []*workloads.Deployment{}

	batchCnt := len(nodes) / batchSize

	for i := 0; i < batchCnt; i++ {
		batchIDs := nodes[i*batchSize : (i+1)*batchSize]

		networkDeployments := make([]*workloads.ZNet, len(batchIDs))
		vmDeployments := make([]*workloads.Deployment, len(batchIDs))

		for j := 0; j < batchSize; j++ {
			network := workloads.ZNet{
				Name:        generateRandomString(15),
				Description: "network for testing",
				Nodes:       batchIDs,
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess: false,
			}

			vm := workloads.VM{
				Name:        "vm",
				Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
				CPU:         2,
				Planetary:   true,
				Memory:      1024,
				Entrypoint:  "/sbin/zinit init",
				NetworkName: network.Name,
			}
			deployment := workloads.NewDeployment(generateRandomString(15), batchIDs[j], "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
			vmDeployments[j] = &deployment
			networkDeployments[j] = &network
		}

		err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, networkDeployments)
		if err != nil {
			fmt.Printf("failed to batch deploy networks\n")
			fmt.Printf("err: %s\n", err)
			continue
		}

		allNetworkDeployments = append(allNetworkDeployments, networkDeployments...)

		err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, vmDeployments)
		if err != nil {
			fmt.Printf("failed to deploy vms\n")
			fmt.Printf("err: %s\n", err)
			continue
		}
		allVmDeployments = append(allVmDeployments, vmDeployments...)

		fmt.Printf("Done deploying %d vms\n", batchSize)
	}

	elapsedTime := time.Since(deploymentStartTime)

	for i := 0; i < len(allVmDeployments); i++ {
		err := tfPluginClient.DeploymentDeployer.Cancel(ctx, allVmDeployments[i])
		if err != nil {
			fmt.Println(err)
		}

		err = tfPluginClient.NetworkDeployer.Cancel(ctx, allNetworkDeployments[i])
		if err != nil {
			fmt.Println(err)
		}

	}
	return len(allVmDeployments), elapsedTime
}

func getNodes(ctx context.Context, tfPluginClient deployer.TFPluginClient, totalVMCount int) []uint32 {
	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		nodeFilter,
		[]uint64{*convertGBToBytes(5)},
		nil,
		[]uint64{minRootfs},
		uint64(totalVMCount+200),
	)

	if err != nil || len(nodes) < totalVMCount {
		fmt.Println(err)
		fmt.Printf("no available nodes found, Only found %d\n", len(nodes))
		os.Exit(1)
	}

	nodesIDs := getReachableNodes(nodes, tfPluginClient, ctx)
	nodesIDs = getNodesWithValidFileSystem(nodesIDs, tfPluginClient, ctx)

	if len(nodesIDs) < totalVMCount {
		fmt.Printf("no available nodes found, Only found %d\n", len(nodesIDs))
		os.Exit(1)
	}

	fmt.Printf("Found free %d nodes!\n", len(nodes))
	return nodesIDs[:totalVMCount]
}
