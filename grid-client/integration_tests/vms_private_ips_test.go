package integration

import (
	"context"
	"fmt"
	"net"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

const (
	vm1Name = "vm1"
	vm2Name = "vm2"
	vm3Name = "vm3"
)

func TestDeploymentsDeploy(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeMRU(*convertGBToBytes(4 * minMemory))),
		nil,
		nil,
		nil,
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network := generateBasicNetwork([]uint32{nodeID})

	vm1 := workloads.VM{
		Name:        vm1Name,
		NetworkName: network.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		Flist:       "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		Entrypoint:  "/sbin/zinit init",
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		if err != nil {
			t.Log(err)
		}
	})

	d1 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &d1)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &d1)
		if err != nil {
			t.Log(err)
		}
	})

	d2 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &d2)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &d2)
		if err != nil {
			t.Log(err)
		}
	})

	dl1, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, d1.Name)
	if err != nil {
		t.Fatal(err)
	}
	dl2, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, d2.Name)
	if err != nil {
		t.Fatal(err)
	}

	if dl1.Vms[0].IP == dl2.Vms[0].IP {
		t.Fatal("expected vms in the same network to have different ips but got the same ip")
	}

	nodeClient, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID)
	assert.NoError(t, err)

	privateIPs, err := nodeClient.NetworkListPrivateIPs(context.Background(), network.Name)
	assert.NoError(t, err)

	if len(slices.Compact[[]string, string](privateIPs)) != 2 {
		t.Fatalf("expected 2 used private IPs but got %d", len(privateIPs))
	}
	assert.Equal(t, privateIPs, []string{dl1.Vms[0].IP, dl2.Vms[0].IP})

	// replace first vm and add another one (vm2 and vm3)
	d1.Vms = append(d1.Vms, vm1)
	d1.Vms[1].Name = vm2Name
	d1.Vms = append(d1.Vms, vm1)
	d1.Vms[2].Name = vm3Name

	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &d1)
	if err != nil {
		t.Fatal(err)
	}

	dl1, err = tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, d1.Name)
	if err != nil {
		t.Fatal(err)
	}

	privateIPs, err = nodeClient.NetworkListPrivateIPs(context.Background(), network.Name)
	assert.NoError(t, err)

	if len(slices.Compact[[]string, string](privateIPs)) != 4 {
		t.Fatalf("expected 4 unique used private IPs but got %d", len(privateIPs))
	}
}

func TestDeploymentsBatchDeploy(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeMRU(*convertGBToBytes(6 * minMemory))),
		nil,
		nil,
		nil,
		2,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[0].NodeID)
	network := generateBasicNetwork([]uint32{nodeID1, nodeID2})

	vm1 := workloads.VM{
		Name:        vm1Name,
		NetworkName: network.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		Flist:       "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		Entrypoint:  "/sbin/zinit init",
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		if err != nil {
			t.Log(err)
		}
	})

	d1 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID1, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d1.Vms[1].Name = vm2Name
	d1.Vms[2].Name = vm3Name

	d2 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID1, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d2.Vms[1].Name = vm2Name
	d2.Vms[2].Name = vm3Name

	d3 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID2, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d3.Vms[1].Name = vm2Name
	d3.Vms[2].Name = vm3Name

	err = tfPluginClient.DeploymentDeployer.BatchDeploy(context.Background(), []*workloads.Deployment{&d1, &d2, &d3})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tfPluginClient.BatchCancelContract([]uint64{d1.ContractID, d2.ContractID, d3.ContractID})
		if err != nil {
			t.Log(err)
		}
	})

	nodeClient, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID1)
	assert.NoError(t, err)

	privateIPs, err := nodeClient.NetworkListPrivateIPs(context.Background(), network.Name)
	assert.NoError(t, err)

	if len(slices.Compact[[]string, string](privateIPs)) != 6 {
		t.Fatalf("expected 6 used private IPs but got %d", len(privateIPs))
	}

	nodeClient2, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID2)
	assert.NoError(t, err)

	privateIPs2, err := nodeClient2.NetworkListPrivateIPs(context.Background(), network.Name)
	assert.NoError(t, err)

	if len(slices.Compact[[]string, string](privateIPs2)) != 3 {
		t.Fatalf("expected 3 used private IPs but got %d", len(privateIPs))
	}

	dl1, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID1, d1.Name)
	if err != nil {
		t.Fatal(err)
	}

	dl2, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID1, d2.Name)
	if err != nil {
		t.Fatal(err)
	}

	dl3, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID2, d3.Name)
	if err != nil {
		t.Fatal(err)
	}

	ips, err := calcDeploymentHostIDs(dl1)

	ips2, err := calcDeploymentHostIDs(dl2)
	ips = append(ips, ips2...)

	ips3, err := calcDeploymentHostIDs(dl3)
	ips = append(ips, ips3...)

	// make sure we have different 9 ips
	assert.Equal(t, len(slices.Compact[[]byte, byte](ips)), 9)
}

func calcDeploymentHostIDs(dl workloads.Deployment) ([]byte, error) {
	ips := make([]byte, 0)

	for _, vm := range dl.Vms {
		ip := net.ParseIP(vm.IP).To4()
		if ip == nil {
			return nil, fmt.Errorf("vm private ip should never be empty")
		}

		if workloads.Contains(ips, ip[3]) {
			return nil, fmt.Errorf("ip already used before %s", ip)
		}

		ips = append(ips, ip[3])
	}

	return ips, nil
}
