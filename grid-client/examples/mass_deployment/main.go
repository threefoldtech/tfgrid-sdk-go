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
	totalVMCount = 500
	batchSize    = 250
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Minute)
	defer cancel()

	tfPluginClients, err := setup()
	if err != nil {
		fmt.Println("failed to create new tfPluginClient: " + err.Error())
		os.Exit(1)
	}

	filtrationStartTime := time.Now()
	nodes, err := getNodes(ctx, tfPluginClients[0], totalVMCount)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	filtrationDuration := time.Since(filtrationStartTime)

	var passed int
	massSize := totalVMCount / len(tfPluginClients)
	var deploymentDuration, cancelationDuration time.Duration
	var wg sync.WaitGroup
	var lock sync.Mutex

	// the actual deployment
	for i := 0; i < len(tfPluginClients); i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			nodesIDs := nodes[j*massSize : (j+1)*massSize]
			p, d, c := MassDeploy(ctx, tfPluginClients[j], nodesIDs)

			lock.Lock()
			passed += p
			deploymentDuration = max(deploymentDuration, d)
			cancelationDuration = max(cancelationDuration, c)
			lock.Unlock()
		}(i)
	}
	wg.Wait()

	fmt.Printf("nodes filtration took %s\n", filtrationDuration)
	fmt.Printf("deployment of %d vms passed\n", passed)
	fmt.Printf("deployment took %s\n", deploymentDuration)
	fmt.Printf("cancelation took %s\n", cancelationDuration)
}

func MassDeploy(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodes []uint32) (int, time.Duration, time.Duration) {
	deploymentStartTime := time.Now()
	allNetworkDeployments := []*workloads.ZNet{}
	allVmDeployments := []*workloads.Deployment{}

	batchCnt := len(nodes) / batchSize

	for i := 0; i < batchCnt; i++ {
		batchIDs := nodes[i*batchSize : (i+1)*batchSize]
		batchIDs = getReachableNodes(batchIDs, tfPluginClient, ctx)

		networkDeployments := make([]*workloads.ZNet, len(batchIDs))
		vmDeployments := make([]*workloads.Deployment, len(batchIDs))

		for j, nodeID := range batchIDs {
			network := workloads.ZNet{
				Name:        generateRandomString(15),
				Description: "network for testing",
				Nodes:       []uint32{nodeID},
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
			deployment := workloads.NewDeployment(generateRandomString(15), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
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
	}

	deploymentTime := time.Since(deploymentStartTime)

	cleanUpStartTime := time.Now()
	CancelDeployments(tfPluginClient)

	return len(allVmDeployments), deploymentTime, time.Since(cleanUpStartTime)
}

func CancelDeployments(tfPluginClient deployer.TFPluginClient) {
	contracts := tfPluginClient.State.CurrentNodeDeployments

	allContractsIDs := []uint64{}
	for _, contractsID := range contracts {
		allContractsIDs = append(allContractsIDs, contractsID...)
	}

	err := tfPluginClient.BatchCancelContract(allContractsIDs)
	if err != nil {
		fmt.Println(err)
	}
}
