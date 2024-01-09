package deployer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type Deployer struct {
	TFPluginClient deployer.TFPluginClient
}

func NewDeployer(conf parser.Config) (Deployer, error) {
	network := conf.Network
	log.Printf("network: %s\n", network)

	mnemonic := conf.Mnemonic
	log.Printf("mnemonics: %s\n", mnemonic)

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
		mru := uint64(convertGBToBytes(int(group.FreeMRU)))
		filter.FreeMRU = &mru
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
	if group.CertificationType != "" {
		filter.CertificationType = &group.CertificationType
	}

	filter.IPv4 = &group.Pubip4
	filter.IPv6 = &group.Pubip6
	filter.Dedicated = &group.Dedicated

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
	if len(nodes) < int(group.NodesCount) {
		return []types.Node{}, errors.New("could not find enough nodes with the requested filter")
	}
	return nodes, err
}

func (d Deployer) ParseVms(vms []parser.Vm, groups map[string][]int, sshKeys map[string]string) (map[string][]workloads.VM, map[string][]*workloads.Disk) {
	vmsWorkloads := map[string][]workloads.VM{}
	vmsDisks := map[string][]*workloads.Disk{}
	for _, vm := range vms {
		// make sure the group has vaild nodes
		if _, ok := groups[vm.Nodegroup]; !ok {
			continue
		}
		sshKey := sshKeys[vm.SSHKey]

		w := workloads.VM{
			Name:       vm.Name,
			Flist:      vm.Flist,
			CPU:        vm.FreeCPU,
			Memory:     convertGBToBytes(vm.FreeMRU),
			PublicIP:   vm.Pubip4,
			PublicIP6:  vm.Pubip6,
			RootfsSize: convertGBToBytes(vm.Rootsize),
			Entrypoint: vm.Entrypoint,
			EnvVars:    map[string]string{"SSH_KEY": sshKey},
		}

		var disk *workloads.Disk
		if vm.Disk.Mount != "" {
			disk = &workloads.Disk{
				Name:   fmt.Sprintf("%sdisk", vm.Name),
				SizeGB: convertGBToBytes(vm.Disk.Capacity),
			}
			w.Mounts = []workloads.Mount{{DiskName: disk.Name, MountPoint: vm.Disk.Mount}}
		}

		if vm.Count == 0 { // if vms count is not specified so it's one vm
			vm.Count++
		}

		for i := 0; i < vm.Count; i++ {
			vmsWorkloads[vm.Nodegroup] = append(vmsWorkloads[vm.Nodegroup], w)
			vmsDisks[vm.Nodegroup] = append(vmsDisks[vm.Nodegroup], disk)
		}
	}
	return vmsWorkloads, vmsDisks
}

func (d Deployer) MassDeploy(ctx context.Context, vms []workloads.VM, nodes []int, disks []*workloads.Disk) error {
	networks := make([]*workloads.ZNet, len(vms))
	vmDeployments := make([]*workloads.Deployment, len(vms))

	var lock sync.Mutex
	var wg sync.WaitGroup

	nodesCounter := 0
	nodesCount := len(nodes)

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

			var workloadDisks []workloads.Disk
			if disks[i] != nil {
				workloadDisks = []workloads.Disk{*disks[i]}
			}
			deployment := workloads.NewDeployment(generateRandomString(10), nodeID, "", nil, network.Name, workloadDisks, nil, []workloads.VM{vm}, nil)

			lock.Lock()
			networks[i] = &network
			vmDeployments[i] = &deployment
			lock.Unlock()
		}(vm, i, uint32(nodeID))
	}
	wg.Wait()

	err := d.TFPluginClient.NetworkDeployer.BatchDeploy(ctx, networks)
	if err != nil {
		return err
	}

	err = d.TFPluginClient.DeploymentDeployer.BatchDeploy(ctx, vmDeployments)
	if err != nil {
		return err
	}

	return nil
}
