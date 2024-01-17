package deployer

import (
	"context"
	"fmt"
	"math/rand"
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

type vmInfo struct {
	Name      string
	PublicIP4 string
	PublicIP6 string
	YggIP     string
	IP        string
	Mounts    []workloads.Mount
}

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
	passedGroups := map[string][]string{}
	failedGroups := map[string]error{}

	tfPluginClient, err := setup(cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	groupsNodes, failed := filterNodes(tfPluginClient, cfg.NodeGroups, ctx)
	failedGroups = failed

	groupsDeployments := parseVMs(tfPluginClient, cfg.Vms, groupsNodes, cfg.SSHKeys)
	var lock sync.Mutex
	var wg sync.WaitGroup

	deploymentStart := time.Now()

	for nodeGroup, deployemnts := range groupsDeployments {
		wg.Add(1)
		go func(group string, deployemnts groupDeploymentsInfo) {
			defer wg.Done()
			info, err := massDeploy(tfPluginClient, ctx, deployemnts)

			lock.Lock()
			defer lock.Unlock()

			if err != nil {
				failedGroups[group] = err
			} else {
				passedGroups[group] = info
			}
		}(nodeGroup, deployemnts)
	}
	wg.Wait()

	log.Info().Msgf("deployment took %s", time.Since(deploymentStart))

	if len(passedGroups) > 0 {
		fmt.Println("ok:")
	}
	for group, info := range passedGroups {
		fmt.Printf("%s: \n%v\n", group, info)
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
		filter.FreeMRU = &group.FreeMRU

		if group.FreeSRU > 0 {
			ssd := convertGBToBytes(group.FreeSRU)
			filter.FreeSRU = &ssd
		}
		if group.FreeHRU > 0 {
			hdd := convertGBToBytes(group.FreeHRU)
			filter.FreeHRU = &hdd
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
		deploymentsInfo[nodeGroup] = buildDeployments(tfPluginClient, vms, nodeGroups[nodeGroup], sshKeys)
	}
	return deploymentsInfo
}

func massDeploy(tfPluginClient deployer.TFPluginClient, ctx context.Context, deployemnts groupDeploymentsInfo) ([]string, error) {
	err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, deployemnts.networkDeployments)
	if err != nil {
		return []string{}, err
	}

	err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, deployemnts.vmDeployments)
	if err != nil {
		return []string{}, err
	}
	vmsInfo := loadDeploymentsInfo(tfPluginClient, deployemnts.deploymentsInfo)

	return vmsInfo, nil
}

func loadDeploymentsInfo(tfPluginClient deployer.TFPluginClient, deployemnts []vmDeploymentInfo) []string {
	vmsInfo := []string{}
	for _, info := range deployemnts {
		vm, err := tfPluginClient.State.LoadVMFromGrid(info.nodeID, info.vmName, info.deploymentName)
		if err != nil {
			log.Debug().Err(err).Msgf("couldn't load vm %s of deployment %s from node %d", info.vmName, info.deploymentName, info.nodeID)
			continue
		}
		info := vmInfo{
			Name:      vm.Name,
			PublicIP4: vm.ComputedIP,
			PublicIP6: vm.ComputedIP6,
			YggIP:     vm.YggIP,
			IP:        vm.IP,
			Mounts:    vm.Mounts,
		}
		groupInfo, err := yaml.Marshal(info)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal json")
		}
		vmsInfo = append(vmsInfo, string(groupInfo))
	}
	return vmsInfo
}

func buildDeployments(tfPluginClient deployer.TFPluginClient, vms []Vms, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	var vmDeployments []*workloads.Deployment
	var networkDeployments []*workloads.ZNet
	var deployemntsInfo []vmDeploymentInfo
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
				Name:        fmt.Sprintf("%s%d", vmGroup.Name, i),
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
				RootfsSize:  int(convertGBToBytes(vmGroup.Rootsize)),
				Entrypoint:  vmGroup.Entrypoint,
				EnvVars:     map[string]string{"SSH_KEY": sshKeys[vmGroup.SSHKey]},
				Mounts:      mounts,
			}
			deployment := workloads.NewDeployment(generateRandomString(10), nodeID, "", nil, network.Name, disks, nil, []workloads.VM{w}, nil)

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, &network)
			deployemntsInfo = append(deployemntsInfo, vmDeploymentInfo{nodeID: nodeID, deploymentName: deployment.Name, vmName: w.Name})
		}
	}
	return groupDeploymentsInfo{vmDeployments: vmDeployments, networkDeployments: networkDeployments, deploymentsInfo: deployemntsInfo}
}

func convertGBToBytes(gb uint64) uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return bytes
}

func parseDisks(name string, disks []Disk) (disksWorkloads []workloads.Disk, mountsWorkloads []workloads.Mount) {
	for _, disk := range disks {
		DiskWorkload := workloads.Disk{
			Name:   fmt.Sprintf("%sdisk", name),
			SizeGB: int(convertGBToBytes(disk.Size)),
		}

		disksWorkloads = append(disksWorkloads, DiskWorkload)
		mountsWorkloads = append(mountsWorkloads, workloads.Mount{DiskName: DiskWorkload.Name, MountPoint: disk.Mount})
	}
	return
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
