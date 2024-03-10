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
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	vm1Name = "vm1"
	vm2Name = "vm2"
	vm3Name = "vm3"
)

func TestDeploymentsDeploy(t *testing.T) {
	tf, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := deployer.FilterNodes(context.Background(), tf, nodeFilter, []uint64{*convertGBToBytes(1)}, nil, []uint64{minRootfs})
	if err != nil || len(nodes) == 0 {
		t.Skip("no available nodes found")
	}

	node := uint32(nodes[0].NodeID)
	network := workloads.ZNet{
		Name:  "network_two_deployments",
		Nodes: []uint32{node},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
	}

	vm1 := workloads.VM{
		Name:        vm1Name,
		Flist:       "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:         2,
		Memory:      1024,
		Entrypoint:  "/sbin/zinit init",
		NetworkName: network.Name,
	}

	err = tf.NetworkDeployer.Deploy(context.Background(), &network)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tf.NetworkDeployer.Cancel(context.Background(), &network)
		if err != nil {
			t.Log(err)
		}
	})

	d1 := workloads.NewDeployment("deployment1", node, "", nil, network.Name, nil, nil, []workloads.VM{vm1}, nil)
	err = tf.DeploymentDeployer.Deploy(context.Background(), &d1)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tf.DeploymentDeployer.Cancel(context.Background(), &d1)
		if err != nil {
			t.Log(err)
		}
	})

	d2 := workloads.NewDeployment("deployment2", node, "", nil, network.Name, nil, nil, []workloads.VM{vm1}, nil)
	err = tf.DeploymentDeployer.Deploy(context.Background(), &d2)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tf.DeploymentDeployer.Cancel(context.Background(), &d2)
		if err != nil {
			t.Log(err)
		}
	})

	dl1, err := tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment1")
	if err != nil {
		t.Fatal(err)
	}

	dl2, err := tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment2")
	if err != nil {
		t.Fatal(err)
	}

	if dl1.Vms[0].IP == dl2.Vms[0].IP {
		t.Fatal("expected vms in the same network to have different ips but got the same ip")
	}

	nodeClient, err := tf.NcPool.GetNodeClient(tf.SubstrateConn, node)
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

	err = tf.DeploymentDeployer.Deploy(context.Background(), &d1)
	if err != nil {
		t.Fatal(err)
	}

	dl1, err = tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment1")
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
	tf, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := deployer.FilterNodes(context.Background(), tf, nodeFilter, []uint64{*convertGBToBytes(1)}, nil, []uint64{minRootfs}, 2)
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network := workloads.ZNet{
		Name:  "network_two_deployments_batch",
		Nodes: []uint32{nodeID1, nodeID2},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
	}

	vm1 := workloads.VM{
		Name:        vm1Name,
		Flist:       "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:         2,
		Memory:      1024,
		Entrypoint:  "/sbin/zinit init",
		NetworkName: network.Name,
	}

	err = tf.NetworkDeployer.Deploy(context.Background(), &network)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tf.NetworkDeployer.Cancel(context.Background(), &network)
		if err != nil {
			t.Log(err)
		}
	})

	d1 := workloads.NewDeployment("deployment1", nodeID1, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d1.Vms[1].Name = vm2Name
	d1.Vms[2].Name = vm3Name

	d2 := workloads.NewDeployment("deployment2", nodeID1, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d2.Vms[1].Name = vm2Name
	d2.Vms[2].Name = vm3Name

	d3 := workloads.NewDeployment("deployment3", nodeID2, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d3.Vms[1].Name = vm2Name
	d3.Vms[2].Name = vm3Name

	err = tf.DeploymentDeployer.BatchDeploy(context.Background(), []*workloads.Deployment{&d1, &d2, &d3})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tf.BatchCancelContract([]uint64{d1.ContractID, d2.ContractID, d3.ContractID})
		if err != nil {
			t.Log(err)
		}
	})

	nodeClient, err := tf.NcPool.GetNodeClient(tf.SubstrateConn, nodeID1)
	assert.NoError(t, err)

	privateIPs, err := nodeClient.NetworkListPrivateIPs(context.Background(), network.Name)
	assert.NoError(t, err)

	if len(slices.Compact[[]string, string](privateIPs)) != 6 {
		t.Fatalf("expected 6 used private IPs but got %d", len(privateIPs))
	}

	nodeClient2, err := tf.NcPool.GetNodeClient(tf.SubstrateConn, nodeID2)
	assert.NoError(t, err)

	privateIPs2, err := nodeClient2.NetworkListPrivateIPs(context.Background(), network.Name)
	assert.NoError(t, err)

	if len(slices.Compact[[]string, string](privateIPs2)) != 3 {
		t.Fatalf("expected 3 used private IPs but got %d", len(privateIPs))
	}

	dl1, err := tf.State.LoadDeploymentFromGrid(context.Background(), nodeID1, "deployment1")
	if err != nil {
		t.Fatal(err)
	}

	dl2, err := tf.State.LoadDeploymentFromGrid(context.Background(), nodeID1, "deployment2")
	if err != nil {
		t.Fatal(err)
	}

	dl3, err := tf.State.LoadDeploymentFromGrid(context.Background(), nodeID2, "deployment3")
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
