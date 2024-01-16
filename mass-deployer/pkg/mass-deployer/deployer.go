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
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type vmDeploymentInfo struct {
	nodeID         uint32
	vmName         string
	deploymentName string
}

type VMInfo struct {
	Name      string
	PublicIP4 string
	PublicIP6 string
	YggIP     string
	IP        string
	Mounts    []workloads.Mount
}

func RunDeployer(configFile []byte) error {
	cfg, err := parser.ParseConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	tfPluginClient, err := setup(cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	ctx := context.Background()

	groupsNodes := map[string][]int{}
	pass := map[string][]VMInfo{}
	fail := map[string]error{}

	for _, group := range cfg.NodeGroups {
		nodes, err := filterNodes(tfPluginClient, group, ctx)
		if err != nil {
			fail[group.Name] = err
			continue
		}

		nodesIDs := []int{}
		for _, node := range nodes {
			nodesIDs = append(nodesIDs, node.NodeID)
		}
		groupsNodes[group.Name] = nodesIDs
	}

	vmsWorkloads, disksWorkloads := parseVms(tfPluginClient, cfg.Vms, groupsNodes, cfg.SSHKeys)
	var lock sync.Mutex
	var wg sync.WaitGroup

	deploymentStart := time.Now()

	for group, vms := range vmsWorkloads {
		wg.Add(1)
		go func(group string, vms []workloads.VM) {
			defer wg.Done()
			info, err := massDeploy(tfPluginClient, ctx, vms, groupsNodes[group], disksWorkloads[group])

			lock.Lock()
			defer lock.Unlock()

			if err != nil {
				fail[group] = err
			} else {
				pass[group] = info
			}
		}(group, vms)
	}
	wg.Wait()

	fmt.Println("deployment took ", time.Since(deploymentStart))
	if len(pass) > 0 {
		fmt.Println("ok:")
	}
	for group, info := range pass {

		groupInfo, err := yaml.Marshal(info)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal json")
		}
		fmt.Printf("%s: \n%v\n", group, string(groupInfo))
	}

	if len(fail) > 0 {
		fmt.Println("error:")
	}
	for group, err := range fail {
		fmt.Printf("%s: %v\n", group, err)
	}
	return nil
}

func setup(conf parser.Config) (deployer.TFPluginClient, error) {
	network := conf.Network
	log.Printf("network: %s", network)

	mnemonic := conf.Mnemonic
	log.Printf("mnemonics: %s", mnemonic)

	return deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, false)
}

func massDeploy(tfPluginClient deployer.TFPluginClient, ctx context.Context, vms []workloads.VM, nodes []int, disks [][]workloads.Disk) ([]VMInfo, error) {
	networks := make([]*workloads.ZNet, len(vms))
	vmDeployments := make([]*workloads.Deployment, len(vms))

	var lock sync.Mutex
	var wg sync.WaitGroup

	nodesCount := len(nodes)
	deploymentInfo := []vmDeploymentInfo{}

	for i, vm := range vms {
		nodeID := nodes[i%nodesCount]
		wg.Add(1)

		go func(vm workloads.VM, i int, nodeID uint32) {
			defer wg.Done()

			network := workloads.ZNet{
				Name:        generateRandomString(10),
				Description: "network for mass deployment",
				Nodes:       []uint32{nodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess: false,
			}

			vm.NetworkName = network.Name

			deployment := workloads.NewDeployment(generateRandomString(10), nodeID, "", nil, network.Name, disks[i], nil, []workloads.VM{vm}, nil)

			lock.Lock()
			networks[i] = &network
			vmDeployments[i] = &deployment
			deploymentInfo = append(deploymentInfo, vmDeploymentInfo{nodeID: nodeID, deploymentName: deployment.Name, vmName: vm.Name})
			lock.Unlock()
		}(vm, i, uint32(nodeID))
	}
	wg.Wait()

	err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, networks)
	if err != nil {
		return []VMInfo{}, err
	}

	err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, vmDeployments)
	if err != nil {
		return []VMInfo{}, err
	}

	vmsInfo := []VMInfo{}
	for _, vmInfo := range deploymentInfo {
		vm, err := tfPluginClient.State.LoadVMFromGrid(vmInfo.nodeID, vmInfo.vmName, vmInfo.deploymentName)
		if err != nil {
			log.Debug().Err(err).Msgf("couldn't load vm %s of deployment %s from node %d", vmInfo.vmName, vmInfo.deploymentName, vmInfo.nodeID)
			continue
		}
		info := VMInfo{
			Name:      vm.Name,
			PublicIP4: vm.ComputedIP,
			PublicIP6: vm.ComputedIP6,
			YggIP:     vm.YggIP,
			IP:        vm.IP,
			Mounts:    vm.Mounts,
		}
		vmsInfo = append(vmsInfo, info)
	}

	return vmsInfo, nil
}

func filterNodes(tfPluginClient deployer.TFPluginClient, group parser.NodesGroup, ctx context.Context) ([]types.Node, error) {
	filter := types.NodeFilter{}
	statusUp := "up"
	filter.Status = &statusUp
	filter.TotalCRU = &group.FreeCPU
	filter.FreeMRU = &group.FreeMRU

	if group.FreeSSD > 0 {
		ssd := uint64(convertGBToBytes(int(group.FreeSSD)))
		filter.FreeSRU = &ssd
	}
	if group.FreeHDD > 0 {
		hdd := uint64(convertGBToBytes(int(group.FreeHDD)))
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

	freeSSD := []uint64{group.FreeSSD}
	if group.FreeSSD == 0 {
		freeSSD = nil
	}
	freeHDD := []uint64{group.FreeHDD}
	if group.FreeSSD == 0 {
		freeHDD = nil
	}

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, filter, freeSSD, freeHDD, nil, int(group.NodesCount))
	return nodes, err
}

func parseVms(tfPluginClient deployer.TFPluginClient, vms []parser.Vm, groups map[string][]int, sshKeys map[string]string) (map[string][]workloads.VM, map[string][][]workloads.Disk) {
	vmsWorkloads := map[string][]workloads.VM{}
	vmsDisks := map[string][][]workloads.Disk{}
	for _, vm := range vms {
		// make sure the group is valid
		if _, ok := groups[vm.Nodegroup]; !ok {
			continue
		}
		sshKey := sshKeys[vm.SSHKey]

		w := workloads.VM{
			Flist:      vm.Flist,
			CPU:        vm.FreeCPU,
			Memory:     vm.FreeMRU,
			PublicIP:   vm.Pubip4,
			PublicIP6:  vm.Pubip6,
			Planetary:  vm.Planetary,
			RootfsSize: convertGBToBytes(vm.Rootsize),
			Entrypoint: vm.Entrypoint,
			EnvVars:    map[string]string{"SSH_KEY": sshKey},
		}

		var disks []workloads.Disk
		var mounts []workloads.Mount
		for _, disk := range vm.SSDDisks {
			DiskWorkload := workloads.Disk{
				Name:   fmt.Sprintf("%sdisk", vm.Name),
				SizeGB: convertGBToBytes(disk.Capacity),
			}

			disks = append(disks, DiskWorkload)
			mounts = append(mounts, workloads.Mount{DiskName: DiskWorkload.Name, MountPoint: disk.Mount})
		}
		w.Mounts = mounts

		for i := 0; i < vm.Count; i++ {
			w.Name = fmt.Sprintf("%s%d", vm.Name, i)
			vmsWorkloads[vm.Nodegroup] = append(vmsWorkloads[vm.Nodegroup], w)
			vmsDisks[vm.Nodegroup] = append(vmsDisks[vm.Nodegroup], disks)
		}
	}
	return vmsWorkloads, vmsDisks
}
