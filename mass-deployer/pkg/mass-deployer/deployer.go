package deployer

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type groupDeploymentsInfo struct {
	vmDeployments      []*workloads.Deployment
	networkDeployments []*workloads.ZNet
	deploymentsInfo    []vmDeploymentInfo
}

type vmDeploymentInfo struct {
	nodeID         uint32
	vmName         string
	deploymentName string
}

func RunDeployer(cfg Config) error {
	ctx := context.Background()

	tfPluginClient, err := setup(cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	groupsNodes, failed := filterNodes(tfPluginClient, cfg.NodeGroups, ctx)
	failedGroups := failed
	passedGroups := map[string][]string{}

	groupsDeployments := parseVMs(tfPluginClient, cfg.Vms, groupsNodes, cfg.SSHKeys)

	deploymentStart := time.Now()

	for nodeGroup, deployments := range groupsDeployments {
		info, err := massDeploy(tfPluginClient, ctx, deployments)
		if err != nil {
			failedGroups[nodeGroup] = err
		} else {
			passedGroups[nodeGroup] = info
		}
	}

	log.Info().Msgf("deployment took %s", time.Since(deploymentStart))

	if len(passedGroups) > 0 {
		fmt.Println("ok:")
	}
	for group, info := range passedGroups {
		fmt.Printf("%s: \n%+v\n", group, info)
	}

	if len(failedGroups) > 0 {
		fmt.Println("error:")
	}
	for group, err := range failedGroups {
		fmt.Printf("%s: %v\n", group, err)
	}
	return nil
}

func setup(conf Config) (deployer.TFPluginClient, error) {
	network := conf.Network
	log.Debug().Msgf("network: %s", network)

	mnemonic := conf.Mnemonic
	log.Debug().Msgf("mnemonic: %s", mnemonic)

	return deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 30, false)
}

func filterNodes(tfPluginClient deployer.TFPluginClient, groups []NodesGroup, ctx context.Context) (map[string][]int, map[string]error) {
	failedGroups := map[string]error{}
	filteredNodes := map[string][]int{}
	for _, group := range groups {
		filter := types.NodeFilter{}
		statusUp := "up"
		filter.Status = &statusUp
		filter.TotalCRU = &group.FreeCPU
		filter.FreeMRU = convertMBToBytes(group.FreeMRU)

		if group.FreeSRU > 0 {
			filter.FreeSRU = convertGBToBytes(group.FreeSRU)
		}
		if group.FreeHRU > 0 {
			filter.FreeHRU = convertGBToBytes(group.FreeHRU)
		}
		if group.Regions != "" {
			filter.Region = &group.Regions
		}
		if group.Certified {
			certified := "Certified"
			filter.CertificationType = &certified
		}
		if group.Pubip4 {
			filter.IPv4 = &group.Pubip4
		}
		if group.Pubip6 {
			filter.IPv6 = &group.Pubip6
		}
		if group.Dedicated {
			filter.Dedicated = &group.Dedicated
		}

		freeSSD := []uint64{group.FreeSRU}
		if group.FreeSRU == 0 {
			freeSSD = nil
		}
		freeHDD := []uint64{group.FreeHRU}
		if group.FreeHRU == 0 {
			freeHDD = nil
		}

		nodes, err := deployer.FilterNodes(ctx, tfPluginClient, filter, freeSSD, freeHDD, nil, int(group.NodesCount))
		if err != nil {
			failedGroups[group.Name] = err
			continue
		}

		nodesIDs := []int{}
		for _, node := range nodes {
			nodesIDs = append(nodesIDs, node.NodeID)
		}
		filteredNodes[group.Name] = nodesIDs
	}
	return filteredNodes, failedGroups
}

func parseVMs(tfPluginClient deployer.TFPluginClient, vms []Vms, nodeGroups map[string][]int, sshKeys map[string]string) map[string]groupDeploymentsInfo {
	deploymentsInfo := map[string]groupDeploymentsInfo{}
	vmsOfNodeGroups := map[string][]Vms{}
	for _, vm := range vms {
		vmsOfNodeGroups[vm.Nodegroup] = append(vmsOfNodeGroups[vm.Nodegroup], vm)
	}

	for nodeGroup, vms := range vmsOfNodeGroups {
		deploymentsInfo[nodeGroup] = buildDeployments(vms, nodeGroups[nodeGroup], sshKeys)
	}
	return deploymentsInfo
}

func massDeploy(tfPluginClient deployer.TFPluginClient, ctx context.Context, deployments groupDeploymentsInfo) ([]string, error) {
	err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, deployments.networkDeployments)
	if err != nil {
		return []string{}, err
	}

	err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, deployments.vmDeployments)
	if err != nil {
		return []string{}, err
	}
	vmsInfo := loadDeploymentsInfo(tfPluginClient, deployments.deploymentsInfo)

	return vmsInfo, nil
}

func buildDeployments(vms []Vms, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	var vmDeployments []*workloads.Deployment
	var networkDeployments []*workloads.ZNet
	var deploymentsInfo []vmDeploymentInfo
	nodesIDsIdx := 0

	// here we loop over all groups of vms within the same node group, and for every group
	// we loop over all it's vms and create network and vm deployment for it
	// the nodesIDsIdx is a counter used to get nodeID to be able to distribute load over all nodes
	for _, vmGroup := range vms {
		for i := 0; i < int(vmGroup.Count); i++ {
			nodeID := uint32(nodesIDs[nodesIDsIdx%len(nodesIDs)])
			nodesIDsIdx++

			disks, mounts := parseDisks(vmGroup.Name, vmGroup.SSDDisks)

			network := workloads.ZNet{
				Name:        fmt.Sprintf("%s%dnetwork", vmGroup.Name, i),
				Description: "network for mass deployment",
				Nodes:       []uint32{nodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess: false,
			}
			w := workloads.VM{
				Name:        fmt.Sprintf("%s%d", vmGroup.Name, i),
				NetworkName: network.Name,
				Flist:       vmGroup.Flist,
				CPU:         int(vmGroup.FreeCPU),
				Memory:      int(vmGroup.FreeMRU),
				PublicIP:    vmGroup.Pubip4,
				PublicIP6:   vmGroup.Pubip6,
				Planetary:   vmGroup.Planetary,
				RootfsSize:  int(*convertGBToBytes(vmGroup.Rootsize)),
				Entrypoint:  vmGroup.Entrypoint,
				EnvVars:     map[string]string{"SSH_KEY": sshKeys[vmGroup.SSHKey]},
				Mounts:      mounts,
			}
			deployment := workloads.NewDeployment(w.Name, nodeID, "", nil, network.Name, disks, nil, []workloads.VM{w}, nil)

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, &network)
			deploymentsInfo = append(deploymentsInfo, vmDeploymentInfo{nodeID: nodeID, deploymentName: deployment.Name, vmName: w.Name})
		}
	}
	return groupDeploymentsInfo{vmDeployments: vmDeployments, networkDeployments: networkDeployments, deploymentsInfo: deploymentsInfo}
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

func convertGBToBytes(gb uint64) *uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return &bytes
}

func convertMBToBytes(mb uint64) *uint64 {
	bytes := mb * 1024 * 1024
	return &bytes
}
