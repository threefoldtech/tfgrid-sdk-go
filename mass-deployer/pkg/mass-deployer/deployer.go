package deployer

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type Deployer struct {
	TFPluginClient deployer.TFPluginClient
}

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

func NewDeployer(conf parser.Config) (Deployer, error) {
	network := conf.Network
	log.Printf("network: %s", network)

	mnemonic := conf.Mnemonic
	log.Printf("mnemonics: %s", mnemonic)

	tf, err := deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 30, false)
	return Deployer{tf}, err
}

func (d Deployer) FilterNodes(group parser.NodesGroup, ctx context.Context) ([]types.Node, error) {
	filter := types.NodeFilter{}
	statusUp := "up"
	filter.Status = &statusUp

	if group.FreeCPU > 0 {
		filter.TotalCRU = &group.FreeCPU
	}
	if group.FreeMRU > 0 {
		filter.FreeMRU = &group.FreeMRU
	}
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

	if group.NodesCount == 0 {
		group.NodesCount = 1
	}

	nodes, err := deployer.FilterNodes(ctx, d.TFPluginClient, filter, freeSSD, freeHDD, nil, group.NodesCount)
	return nodes, err
}

func (d Deployer) ParseVms(vms []parser.Vm, groups map[string][]int, sshKeys map[string]string) (map[string][]workloads.VM, map[string][][]workloads.Disk) {
	vmsWorkloads := map[string][]workloads.VM{}
	vmsDisks := map[string][][]workloads.Disk{}
	for _, vm := range vms {
		// make sure the group has vaild nodes
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
		for _, disk := range vm.SSHDisks {
			DiskWorkload := workloads.Disk{
				Name:   fmt.Sprintf("%sdisk", vm.Name),
				SizeGB: convertGBToBytes(disk.Capacity),
			}

			disks = append(disks, DiskWorkload)
			mounts = append(mounts, workloads.Mount{DiskName: DiskWorkload.Name, MountPoint: disk.Mount})
		}
		w.Mounts = mounts

		if vm.Count == 0 { // if vms count is not specified so it's one vm
			vm.Count++
		}

		for i := 0; i < vm.Count; i++ {
			w.Name = fmt.Sprintf("%s%d", vm.Name, i)
			vmsWorkloads[vm.Nodegroup] = append(vmsWorkloads[vm.Nodegroup], w)
			vmsDisks[vm.Nodegroup] = append(vmsDisks[vm.Nodegroup], disks)
		}
	}
	return vmsWorkloads, vmsDisks
}

func (d Deployer) MassDeploy(ctx context.Context, vms []workloads.VM, nodes []int, disks [][]workloads.Disk) ([]VMInfo, error) {
	networks := make([]*workloads.ZNet, len(vms))
	vmDeployments := make([]*workloads.Deployment, len(vms))

	var lock sync.Mutex
	var wg sync.WaitGroup

	nodesCounter := 0
	nodesCount := len(nodes)
	deploymentInfo := []vmDeploymentInfo{}

	for i, vm := range vms {
		nodeID := nodes[nodesCounter%nodesCount]
		nodesCounter++

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

	err := d.TFPluginClient.NetworkDeployer.BatchDeploy(ctx, networks)
	if err != nil {
		return []VMInfo{}, err
	}

	err = d.TFPluginClient.DeploymentDeployer.BatchDeploy(ctx, vmDeployments)
	if err != nil {
		return []VMInfo{}, err
	}

	vmsInfo := []VMInfo{}
	for _, vmInfo := range deploymentInfo {
		vm, err := d.TFPluginClient.State.LoadVMFromGrid(vmInfo.nodeID, vmInfo.vmName, vmInfo.deploymentName)
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
