package integration

import (
	"context"
	"net"
	"testing"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const vm2Name = "vm2"
const vm3Name = "vm3"

func TestDeploymentsDeploy(t *testing.T) {

	tf, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := deployer.FilterNodes(context.Background(), tf, nodeFilter, []uint64{*convertGBToBytes(1)}, nil, []uint64{minRootfs})
	if err != nil {
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
		Name:        "vm1",
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
	d := workloads.NewDeployment("deployment1", node, "", nil, network.Name, nil, nil, []workloads.VM{vm1}, nil)
	err = tf.DeploymentDeployer.Deploy(context.Background(), &d)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err = tf.DeploymentDeployer.Cancel(context.Background(), &d)
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
	dl, err := tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment1")
	if err != nil {
		t.Fatal(err)
	}
	dl2, err := tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment2")
	if err != nil {
		t.Fatal(err)
	}
	if dl.Vms[0].IP == dl2.Vms[0].IP {
		t.Fatal("expected vms in the same network to have different ips but got the same ip")
	}
	networkState := tf.State.Networks.GetNetwork(network.Name)
	usedIPs := networkState.GetUsedNetworkHostIDs(node)
	if len(usedIPs) != 2 {
		t.Fatalf("expected 2 used IPs but got %d", len(usedIPs))
	}

	// replace first vm and add another one
	d.Vms[0] = vm1
	d.Vms[0].Name = vm2Name
	d.Vms = append(d.Vms, vm1)
	d.Vms[1].Name = vm3Name

	err = tf.DeploymentDeployer.Deploy(context.Background(), &d)
	if err != nil {
		t.Fatal(err)
	}

	dl, err = tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment1")
	if err != nil {
		t.Fatal(err)
	}
	if workloads.Contains(usedIPs, net.ParseIP(dl.Vms[0].IP).To4()[3]) || workloads.Contains(usedIPs, net.ParseIP(dl.Vms[1].IP).To4()[3]) {
		t.Fatal("expected new vms to not use previously assinged ips")
	}

	networkState = tf.State.Networks.GetNetwork(network.Name)
	usedIPs = networkState.GetUsedNetworkHostIDs(node)
	if len(usedIPs) != 4 {
		t.Fatalf("expected 4 used IPs but got %d", len(usedIPs))
	}
}

func TestDeploymentsBatchDeploy(t *testing.T) {
	tf, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := deployer.FilterNodes(context.Background(), tf, nodeFilter, []uint64{*convertGBToBytes(1)}, nil, []uint64{minRootfs})
	if err != nil {
		t.Skip("no available nodes found")
	}
	node := uint32(nodes[0].NodeID)
	network := workloads.ZNet{
		Name:  "network_two_deployments_batch",
		Nodes: []uint32{node},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
	}
	vm1 := workloads.VM{
		Name:        "vm1",
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
	d := workloads.NewDeployment("deployment1", node, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d.Vms[1].Name = vm2Name
	d.Vms[2].Name = vm3Name

	d2 := workloads.NewDeployment("deployment2", node, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d2.Vms[1].Name = vm2Name
	d2.Vms[2].Name = vm3Name

	err = tf.DeploymentDeployer.BatchDeploy(context.Background(), []*workloads.Deployment{&d, &d2})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tf.BatchCancelContract([]uint64{d.ContractID, d2.ContractID})
		if err != nil {
			t.Log(err)
		}
	})

	networkState := tf.State.Networks.GetNetwork(network.Name)
	usedIPs := networkState.GetUsedNetworkHostIDs(node)
	if len(usedIPs) != 6 {
		t.Fatalf("expected 6 used IPs but got %d", len(usedIPs))
	}

	dl, err := tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment1")
	if err != nil {
		t.Fatal(err)
	}
	dl2, err := tf.State.LoadDeploymentFromGrid(context.Background(), node, "deployment2")
	if err != nil {
		t.Fatal(err)
	}
	ips := make([]byte, 0)
	for _, vm := range dl.Vms {
		ip := net.ParseIP(vm.IP).To4()
		if ip == nil {
			t.Fatal("vm private ip should never be empty")
		}
		if workloads.Contains(ips, ip[3]) {
			t.Errorf("ip already used before %s", ip)
			continue
		}
		ips = append(ips, ip[3])
	}
	for _, vm := range dl2.Vms {
		ip := net.ParseIP(vm.IP).To4()
		if ip == nil {
			t.Fatal("vm private ip should never be empty")
		}
		if workloads.Contains(ips, ip[3]) {
			t.Errorf("ip already used before %s", ip)
			continue
		}
		ips = append(ips, ip[3])
	}
}
