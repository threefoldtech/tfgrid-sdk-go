package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sethvargo/go-retry"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const maxDeploymentRetries = 5

func RunDeployer(ctx context.Context, cfg Config, output string) error {
	passedGroups := map[string][]vmOutput{}
	failedGroups := map[string]string{}

	tfPluginClient, err := setup(cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	deploymentStart := time.Now()

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = maxDeploymentRetries
	}

	for _, nodeGroup := range cfg.NodeGroups {
		log.Info().Str("Node group", nodeGroup.Name).Msg("Running deployment")
		trial := 1
		var failedDeployments []failedDeploymentsInfo

		if err := retry.Do(ctx, retry.WithMaxRetries(cfg.MaxRetries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
			if trial != 1 {
				log.Debug().Str("Node group", nodeGroup.Name).Int("Deployment trial", trial).Msg("Retrying to deploy")
			}

			// deploy node group
			nodesIDs, err := filterNodes(ctx, tfPluginClient, nodeGroup)
			if err != nil {
				return err
			}

			var groupDeployments groupDeploymentsInfo

			if failedDeployments == nil {
				groupDeployments = parseVMsGroup(cfg.Vms, nodeGroup.Name, nodesIDs, cfg.SSHKeys)
			} else {
				var updated bool

				fmt.Println("updaing networks")
				groupDeployments, updated = updateFailedNetworks(nodesIDs, failedDeployments)
				if !updated {
					fmt.Println("updaing vms")
					groupDeployments = updateFailedDeployments(tfPluginClient, nodesIDs, failedDeployments)
				}
			}

			info, failed := massDeploy(ctx, tfPluginClient, groupDeployments)
			if failed != nil {

				log.Debug().Err(err).Str("Node group", nodeGroup.Name).Msg("failed to deploy")

				trial++
				failedDeployments = failed
				return retry.RetryableError(fmt.Errorf("failed to deploy node group %s", nodeGroup.Name))
			}

			passedGroups[nodeGroup.Name] = info
			log.Info().Str("Node group", nodeGroup.Name).Msg("Done deploying")
			return nil
		}); err != nil {
			cancellationErr := tfPluginClient.CancelByProjectName(nodeGroup.Name)
			if cancellationErr != nil {
				log.Debug().Err(err).Send()
			}

			failedGroups[nodeGroup.Name] = err.Error()
		}
	}

	log.Info().Msgf("Deployment took %s", time.Since(deploymentStart))

	outData := struct {
		OK    map[string][]vmOutput `json:"ok"`
		Error map[string]string     `json:"error"`
	}{
		OK:    passedGroups,
		Error: failedGroups,
	}

	var outputBytes []byte
	if filepath.Ext(output) == ".json" {
		outputBytes, err = json.MarshalIndent(outData, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(outData)
	}
	if err != nil {
		return err
	}

	fmt.Println(string(outputBytes))

	if output == "" {
		return nil
	}

	return os.WriteFile(output, outputBytes, 0644)
}

func parseVMsGroup(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	vmsOfNodeGroup := []Vms{}
	for _, vm := range vms {
		if vm.NodeGroup == nodeGroup {
			vmsOfNodeGroup = append(vmsOfNodeGroup, vm)
		}
	}

	return buildDeployments(vmsOfNodeGroup, nodeGroup, nodesIDs, sshKeys)
}

func massDeploy(ctx context.Context, tfPluginClient deployer.TFPluginClient, deployments groupDeploymentsInfo) ([]vmOutput, []failedDeploymentsInfo) {
	networkError := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, deployments.networkDeployments)
	deploymentsError := tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, deployments.vmDeployments)

	fmt.Printf("networkError: %v\n", networkError)
	fmt.Printf("deploymentsError: %+v\n", deploymentsError)

	if networkError != nil || deploymentsError != nil {
		failedDeployments := getFailedDeployments(tfPluginClient, deployments.networkDeployments, deployments.vmDeployments)
		fmt.Println(len(failedDeployments))
		return nil, failedDeployments
	}

	vmsInfo := loadDeploymentsInfo(tfPluginClient, deployments.deploymentsInfo)

	return vmsInfo, nil
}

func buildDeployments(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	var vmDeployments []*workloads.Deployment
	var networkDeployments []*workloads.ZNet
	var deploymentsInfo []vmDeploymentInfo
	var nodesIDsIdx int

	// here we loop over all groups of vms within the same node group, and for every group
	// we loop over all it's vms and create network and vm deployment for it
	// the nodesIDsIdx is a counter used to get nodeID to be able to distribute load over all nodes
	for _, vmGroup := range vms {
		envVars := vmGroup.EnvVars
		if envVars == nil {
			envVars = map[string]string{}
		}
		envVars["SSH_KEY"] = sshKeys[vmGroup.SSHKey]

		for i := 0; i < int(vmGroup.Count); i++ {
			nodeID := uint32(nodesIDs[nodesIDsIdx])
			nodesIDsIdx = (nodesIDsIdx + 1) % len(nodesIDs)

			vmName := fmt.Sprintf("%s%d", vmGroup.Name, i)
			disks, mounts := parseDisks(vmName, vmGroup.SSDDisks)

			network := workloads.ZNet{
				Name:        fmt.Sprintf("%snetwork", vmName),
				Description: "network for mass deployment",
				Nodes:       []uint32{nodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess:  false,
				SolutionType: nodeGroup,
			}

			if !vmGroup.PublicIP4 && !vmGroup.Planetary {
				log.Warn().Msg("Planetary and public IP options are false. Setting planetary IP to true")
				vmGroup.Planetary = true
			}

			vm := workloads.VM{
				Name:        vmName,
				NetworkName: network.Name,
				Flist:       vmGroup.Flist,
				CPU:         int(vmGroup.FreeCPU),
				Memory:      int(vmGroup.FreeMRU * 1024), // Memory is in MB
				PublicIP:    vmGroup.PublicIP4,
				PublicIP6:   vmGroup.PublicIP6,
				Planetary:   vmGroup.Planetary,
				RootfsSize:  int(vmGroup.RootSize * 1024), // RootSize is in MB
				Entrypoint:  vmGroup.Entrypoint,
				EnvVars:     envVars,
				Mounts:      mounts,
			}
			deployment := workloads.NewDeployment(vm.Name, nodeID, nodeGroup, nil, network.Name, disks, nil, []workloads.VM{vm}, nil)

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, &network)
			deploymentsInfo = append(deploymentsInfo, vmDeploymentInfo{nodeID: nodeID, deploymentName: deployment.Name, vmName: vm.Name})
		}
	}
	return groupDeploymentsInfo{vmDeployments: vmDeployments, networkDeployments: networkDeployments, deploymentsInfo: deploymentsInfo}
}

func cancelContractsOfFailedDeployments(tfPluginClient deployer.TFPluginClient, networkDeployments []*workloads.ZNet, vmDeployments []*workloads.Deployment) {
	contracts := []uint64{}
	for _, network := range networkDeployments {
		for _, contract := range network.NodeDeploymentID {
			if contract != 0 {
				contracts = append(contracts, contract)
			}
		}
	}

	for _, vm := range vmDeployments {
		if vm.ContractID != 0 {
			contracts = append(contracts, vm.ContractID)
		}
	}

	err := tfPluginClient.BatchCancelContract(contracts)
	if err != nil {
		log.Debug().Err(err)
	}
}

func getFailedDeployments(tfPluginClient deployer.TFPluginClient, networkDeployments []*workloads.ZNet, vmDeployments []*workloads.Deployment) []failedDeploymentsInfo {
	failedDeployments := []failedDeploymentsInfo{}
	for i := 0; i < len(networkDeployments); i++ {
		var failedFlag bool
		network := networkDeployments[i]
		vm := vmDeployments[i]

		// check if the network is failed to be deployed
		for _, contract := range network.NodeDeploymentID {
			if contract == 0 {
				fmt.Println("network")
				failedFlag = true
				break
			}
		}

		// check if the vm deployment is failed to be deployed
		if vm.ContractID == 0 {

			fmt.Println("vm")
			failedFlag = true
		}

		if failedFlag {
			failedDeployments = append(failedDeployments, failedDeploymentsInfo{vm, network, vmDeploymentInfo{nodeID: vm.NodeID, deploymentName: vm.Name, vmName: vm.Vms[0].Name}})
		}
	}
	return failedDeployments
}

func loadDeploymentsInfo(tfPluginClient deployer.TFPluginClient, deployments []vmDeploymentInfo) []vmOutput {
	vmsInfo := []vmOutput{}
	var lock sync.Mutex
	var wg sync.WaitGroup

	for _, info := range deployments {
		wg.Add(1)

		go func(depInfo vmDeploymentInfo) {
			defer wg.Done()

			vmDeployment, err := tfPluginClient.State.LoadDeploymentFromGrid(depInfo.nodeID, depInfo.deploymentName)
			if err != nil {
				log.Debug().Err(err).
					Str("vm", depInfo.vmName).
					Str("deployment", depInfo.deploymentName).
					Uint32("node ID", depInfo.nodeID).
					Msg("couldn't load from state")
				return
			}

			vm := vmDeployment.Vms[0]
			vmInfo := vmOutput{
				Name:        vm.Name,
				NetworkName: vmDeployment.NetworkName,
				NodeID:      vmDeployment.NodeID,
				ContractID:  vmDeployment.ContractID,
				PublicIP4:   vm.ComputedIP,
				PublicIP6:   vm.ComputedIP6,
				PlanetaryIP: vm.PlanetaryIP,
				IP:          vm.IP,
				Mounts:      vm.Mounts,
			}

			lock.Lock()
			defer lock.Unlock()
			vmsInfo = append(vmsInfo, vmInfo)
		}(info)
	}

	wg.Wait()
	return vmsInfo
}

func parseDisks(name string, disks []Disk) (disksWorkloads []workloads.Disk, mountsWorkloads []workloads.Mount) {
	for _, disk := range disks {
		DiskWorkload := workloads.Disk{
			Name:   fmt.Sprintf("%sdisk", name),
			SizeGB: int(disk.Size),
		}

		disksWorkloads = append(disksWorkloads, DiskWorkload)
		mountsWorkloads = append(mountsWorkloads, workloads.Mount{DiskName: DiskWorkload.Name, MountPoint: disk.Mount})
	}
	return
}

/*
if network failed then

	we need to use new node and change node id for the vm deployment and then deploy the network and vm

else if vm deployment failed

	we need to cancel the network deployment then change node id for both vm and then deploy network and vm

if there's a network that is failed, all vms will fail, so we need to fix the networks first then try to deploy vms
*/
func updateFailedDeployments(tfPluginClient deployer.TFPluginClient, nodesIDs []int, failedDeployments []failedDeploymentsInfo) groupDeploymentsInfo {
	var newDeploymentsInfo groupDeploymentsInfo
	// cancel old network deployments which vm deployments failed
	contractsToBeCanceled := []uint64{}
	for _, deployment := range failedDeployments {
		// get network contracts of failed vm deployments
		for _, contract := range deployment.networkDeployment.NodeDeploymentID {
			if contract != 0 {
				contractsToBeCanceled = append(contractsToBeCanceled, contract)
			}
		}
	}

	fmt.Printf("%d vm deployments failed \n", len(contractsToBeCanceled))
	err := tfPluginClient.BatchCancelContract(contractsToBeCanceled)
	if err != nil {
		log.Debug().Err(err)
	}

	for idx, deployment := range failedDeployments {
		nodeID := uint32(nodesIDs[idx%len(nodesIDs)])

		fmt.Printf("updating contract of vm %s from node %d to node %d\n", deployment.vmDeployment.Name, deployment.vmDeployment.NodeID, nodeID)
		deployment.vmDeployment.NodeID = nodeID
		deployment.deploymentInfo.nodeID = nodeID
		deployment.networkDeployment.Nodes = []uint32{nodeID}

		newDeploymentsInfo.vmDeployments = append(newDeploymentsInfo.vmDeployments, deployment.vmDeployment)
		newDeploymentsInfo.deploymentsInfo = append(newDeploymentsInfo.deploymentsInfo, deployment.deploymentInfo)
		newDeploymentsInfo.networkDeployments = append(newDeploymentsInfo.networkDeployments, deployment.networkDeployment)
	}
	return newDeploymentsInfo
}

// update nodes for failed networks and update vm deployments accordingly
// returns the updated deployments and bool indicating if the networks were updated
func updateFailedNetworks(nodesIDs []int, failedDeployments []failedDeploymentsInfo) (groupDeploymentsInfo, bool) {
	var newDeploymentsInfo groupDeploymentsInfo
	var updated bool
	for idx, deployment := range failedDeployments {

		nodeID := uint32(nodesIDs[idx%len(nodesIDs)])
		for _, contract := range deployment.networkDeployment.NodeDeploymentID {
			if contract == 0 {
				updated = true

				deployment.vmDeployment.NodeID = nodeID
				deployment.deploymentInfo.nodeID = nodeID
				deployment.networkDeployment.Nodes = []uint32{nodeID}

				newDeploymentsInfo.networkDeployments = append(newDeploymentsInfo.networkDeployments, deployment.networkDeployment)
				break
			}
		}

		newDeploymentsInfo.vmDeployments = append(newDeploymentsInfo.vmDeployments, deployment.vmDeployment)
		newDeploymentsInfo.deploymentsInfo = append(newDeploymentsInfo.deploymentsInfo, deployment.deploymentInfo)

	}

	return newDeploymentsInfo, updated
}
