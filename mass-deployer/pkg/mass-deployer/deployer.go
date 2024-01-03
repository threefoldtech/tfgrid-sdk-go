package deployer

import (
	"context"
	"log"
	"os"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
)

type Deployer struct {
	TFPluginClient deployer.TFPluginClient
}

func NewDeployer(conf parser.Config) (Deployer, error) {
	network := os.Getenv("NETWORK")
	log.Printf("network: %s\n", network)

	mnemonic := os.Getenv("MNEMONICS")
	log.Printf("mnemonics: %s\n", mnemonic)

	tf, err := deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 30, false)
	return Deployer{tf}, err
}

func (d Deployer) FilterNodes(group parser.NodesGroup, ctx context.Context) ([]types.Node, error) {
	filter := types.NodeFilter{}
	if group.FreeCPU > 0 {
		filter.TotalCRU = &group.FreeCPU
	}
	if group.FreeMRU > 0 {
		filter.FreeMRU = &group.FreeMRU
	}
	if group.FreeSSD > 0 {
		filter.FreeSRU = &group.FreeSSD
	}
	if group.FreeHDD > 0 {
		filter.FreeHRU = &group.FreeHDD
	}
	if group.Region != "" {
		filter.Region = &group.Region
	}

	statusUp := "up"
	filter.Status = &statusUp

	filter.IPv4 = &group.Pubip4
	filter.IPv6 = &group.Pubip6
	filter.Dedicated = &group.Dedicated
	filter.CertificationType = &group.CertificationType

	return deployer.FilterNodes(ctx, d.TFPluginClient, filter, []uint64{group.FreeSSD}, []uint64{group.FreeHDD}, []uint64{}, group.NodesCount)
}

func (d Deployer) ParseVms(vms []parser.Vm) map[string][]workloads.VM {
	vmsWorkloads := map[string][]workloads.VM{}
	for _, vm := range vms {
		w := workloads.VM{
			Name:       vm.Name,
			Flist:      vm.Flist,
			CPU:        vm.FreeCPU,
			Memory:     vm.FreeMRU,
			PublicIP:   vm.Pubip4,
			PublicIP6:  vm.Pubip6,
			RootfsSize: vm.Rootsize,
			Entrypoint: vm.Entrypoint,
			Mounts:     []workloads.Mount{{DiskName: vm.Name, MountPoint: vm.Disk.Mount}},
		}
		vmsWorkloads[vm.Nodegroup] = append(vmsWorkloads[vm.Nodegroup], w)
	}
	return vmsWorkloads
}

func (d Deployer) CreateNetworkDeployments(vms []parser.Vm, nodesGroups map[string][]int) {
}

func (d Deployer) CreateVMsDeployments(vms []parser.Vm, nodesGroups map[string][]int) {
}
