package deployer

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/sethvargo/go-retry"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func RunDeployer(cfg Config, ctx context.Context) error {
	passedGroups := map[string][]string{}
	failedGroups := map[string]error{}

	tfPluginClient, err := setup(cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	deploymentStart := time.Now()

	for _, nodeGroup := range cfg.NodeGroups {
		log.Info().Msgf("running deployer for node group %s", nodeGroup.Name)
		firstTrial := true

		if err := retry.Do(ctx, retry.WithMaxRetries(2, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
			if !firstTrial {
				log.Debug().Msgf("retrying to deploy node group %s", nodeGroup.Name)
			}

			info, err := deployNodeGroup(tfPluginClient, ctx, nodeGroup, cfg.Vms, cfg.SSHKeys)
			if err != nil {
				firstTrial = false
				log.Debug().Err(err).Msgf("failed to deploy node group %s", nodeGroup.Name)
				return retry.RetryableError(err)
			}

			passedGroups[nodeGroup.Name] = info
			log.Info().Msgf("done deploying node group %s", nodeGroup.Name)
			return nil
		}); err != nil {
			failedGroups[nodeGroup.Name] = err
		}
	}

	log.Info().Msgf("deployment took %s", time.Since(deploymentStart))

	if len(passedGroups) > 0 {
		fmt.Println("ok:")
	}
	for group, info := range passedGroups {
		fmt.Printf("%s:\n", group)
		for _, s := range info {
			fmt.Println(s)
		}
	}

	if len(failedGroups) > 0 {
		fmt.Fprintln(os.Stderr, "error:")
	}
	for group, err := range failedGroups {
		fmt.Fprintf(os.Stderr, "%s: %v\n", group, err)
	}
	return nil
}

func deployNodeGroup(tfPluginClient deployer.TFPluginClient, ctx context.Context, nodeGroup NodesGroup, vms []Vms, sshKeys map[string]string) ([]string, error) {
	nodesIDs, err := filterNodes(tfPluginClient, nodeGroup, ctx)
	if err != nil {
		return []string{}, err
	}

	groupsDeployments := parseGroupVMs(vms, nodeGroup.Name, nodesIDs, sshKeys)

	info, err := massDeploy(tfPluginClient, ctx, groupsDeployments)
	if err != nil {
		return []string{}, err
	}

	return info, nil
}

func parseGroupVMs(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	vmsOfNodeGroup := []Vms{}
	for _, vm := range vms {
		if vm.Nodegroup == nodeGroup {
			vmsOfNodeGroup = append(vmsOfNodeGroup, vm)
		}
	}

	return buildDeployments(vmsOfNodeGroup, nodeGroup, nodesIDs, sshKeys)
}

func massDeploy(tfPluginClient deployer.TFPluginClient, ctx context.Context, deployments groupDeploymentsInfo) ([]string, error) {
	err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, deployments.networkDeployments)
	if err != nil {
		cancelContractsOfFailedDeployments(tfPluginClient, deployments.networkDeployments, []*workloads.Deployment{})
		return []string{}, err
	}

	err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, deployments.vmDeployments)
	if err != nil {
		cancelContractsOfFailedDeployments(tfPluginClient, deployments.networkDeployments, deployments.vmDeployments)
		return []string{}, err
	}
	vmsInfo := loadDeploymentsInfo(tfPluginClient, deployments.deploymentsInfo)

	return vmsInfo, nil
}

func buildDeployments(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	var vmDeployments []*workloads.Deployment
	var networkDeployments []*workloads.ZNet
	var deploymentsInfo []vmDeploymentInfo
	nodesIDsIdx := 0

	// here we loop over all groups of vms within the same node group, and for every group
	// we loop over all it's vms and create network and vm deployment for it
	// the nodesIDsIdx is a counter used to get nodeID to be able to distribute load over all nodes
	for _, vmGroup := range vms {
		for i := 0; i < int(vmGroup.Count); i++ {
			nodeID := uint32(nodesIDs[nodesIDsIdx])
			nodesIDsIdx = (nodesIDsIdx + 1) % len(nodesIDs)

			disks, mounts := parseDisks(vmGroup.Name, vmGroup.SSDDisks)

			network := workloads.ZNet{
				Name:        fmt.Sprintf("%s%dnetwork", vmGroup.Name, i),
				Description: "network for mass deployment",
				Nodes:       []uint32{nodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess:  false,
				SolutionType: nodeGroup,
			}
			w := workloads.VM{
				Name:        fmt.Sprintf("%s%d", vmGroup.Name, i),
				NetworkName: network.Name,
				Flist:       vmGroup.Flist,
				CPU:         int(vmGroup.FreeCPU),
				Memory:      int(vmGroup.FreeMRU),
				PublicIP:    vmGroup.PublicIP4,
				PublicIP6:   vmGroup.PublicIP6,
				Planetary:   vmGroup.Planetary,
				RootfsSize:  int(vmGroup.Rootsize * 1024), // Rootsize is in MB
				Entrypoint:  vmGroup.Entrypoint,
				EnvVars:     map[string]string{"SSH_KEY": sshKeys[vmGroup.SSHKey]},
				Mounts:      mounts,
			}
			deployment := workloads.NewDeployment(w.Name, nodeID, nodeGroup, nil, network.Name, disks, nil, []workloads.VM{w}, nil)

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, &network)
			deploymentsInfo = append(deploymentsInfo, vmDeploymentInfo{nodeID: nodeID, deploymentName: deployment.Name, vmName: w.Name})
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

func loadDeploymentsInfo(tfPluginClient deployer.TFPluginClient, deployments []vmDeploymentInfo) []string {
	vmsInfo := []string{}
	var lock sync.Mutex
	var wg sync.WaitGroup

	for _, info := range deployments {
		wg.Add(1)

		go func(depInfo vmDeploymentInfo) {
			defer wg.Done()

			vm, err := tfPluginClient.State.LoadVMFromGrid(depInfo.nodeID, depInfo.vmName, depInfo.deploymentName)
			if err != nil {
				log.Debug().Err(err).Msgf("couldn't load vm %s of deployment %s from node %d", depInfo.vmName, depInfo.deploymentName, depInfo.nodeID)
				return
			}

			vmInfo := struct {
				Name      string
				PublicIP4 string
				PublicIP6 string
				YggIP     string
				IP        string
				Mounts    []workloads.Mount
			}{vm.Name, vm.ComputedIP, vm.ComputedIP6, vm.YggIP, vm.IP, vm.Mounts}

			groupInfo, err := yaml.Marshal(vmInfo)
			if err != nil {
				log.Debug().Err(err).Msg("failed to marshal json")
			}

			lock.Lock()
			defer lock.Unlock()
			vmsInfo = append(vmsInfo, string(groupInfo))
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
