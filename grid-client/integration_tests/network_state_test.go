package integration

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

const (
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
		generateNodeFilter(),
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
		Name:        "vm1",
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

	d := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &d)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &d)
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
	dl, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, "deployment1")
	if err != nil {
		t.Fatal(err)
	}
	dl2, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, "deployment2")
	if err != nil {
		t.Fatal(err)
	}
	if dl.Vms[0].IP == dl2.Vms[0].IP {
		t.Fatal("expected vms in the same network to have different ips but got the same ip")
	}
	networkState := tfPluginClient.State.Networks.GetNetwork(network.Name)
	usedIPs := networkState.GetUsedNetworkHostIDs(nodeID)
	if len(usedIPs) != 2 {
		t.Fatalf("expected 2 used IPs but got %d", len(usedIPs))
	}

	// replace first vm and add another one
	d.Vms[0] = vm1
	d.Vms[0].Name = vm2Name
	d.Vms = append(d.Vms, vm1)
	d.Vms[1].Name = vm3Name

	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &d)
	if err != nil {
		t.Fatal(err)
	}

	dl, err = tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, "deployment1")
	if err != nil {
		t.Fatal(err)
	}
	if workloads.Contains(usedIPs, net.ParseIP(dl.Vms[0].IP).To4()[3]) || workloads.Contains(usedIPs, net.ParseIP(dl.Vms[1].IP).To4()[3]) {
		t.Fatal("expected new vms to not use previously assinged ips")
	}

	networkState = tfPluginClient.State.Networks.GetNetwork(network.Name)
	usedIPs = networkState.GetUsedNetworkHostIDs(nodeID)
	if len(usedIPs) != 4 {
		t.Fatalf("expected 4 used IPs but got %d", len(usedIPs))
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
		generateNodeFilter(),
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
		Name:        "vm1",
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
	d := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d.Vms[1].Name = vm2Name
	d.Vms[2].Name = vm3Name

	d2 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm1, vm1}, nil)
	d2.Vms[1].Name = vm2Name
	d2.Vms[2].Name = vm3Name

	err = tfPluginClient.DeploymentDeployer.BatchDeploy(context.Background(), []*workloads.Deployment{&d, &d2})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = tfPluginClient.BatchCancelContract([]uint64{d.ContractID, d2.ContractID})
		if err != nil {
			t.Log(err)
		}
	})

	networkState := tfPluginClient.State.Networks.GetNetwork(network.Name)
	usedIPs := networkState.GetUsedNetworkHostIDs(nodeID)
	if len(usedIPs) != 6 {
		t.Fatalf("expected 6 used IPs but got %d", len(usedIPs))
	}

	dl, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, "deployment1")
	if err != nil {
		t.Fatal(err)
	}
	dl2, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, "deployment2")
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
