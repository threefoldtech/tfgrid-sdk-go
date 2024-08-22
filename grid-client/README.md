# Grid3_client_go

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-64.8%25-brightgreen.svg?longCache=true&style=flat)</a>

Grid3_client_go is a go client created to interact with threefold grid. It should manage CRUD operations for deployments on the grid.

## Requirements

[Go](https://golang.org/doc/install) >= 1.19

## Examples

This is a simple example to deploy a VM with a network.

```go
import (
    "github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
    "github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)


// define customized plugin opts
opts := []deloyer.PluginOpt{
  deployer.WithProxyURL("https://gridproxy.bknd1.ninja.tf"),
  deployer.WithNetwork("dev"),
}

// Create Threefold plugin client
tfPlugin, err := deployer.NewTFPluginClient(mnemonics, opts...)

// Get a free node to deploy
nodeID := 14

// Create a new network to deploy
network := workloads.ZNet{
    Name:        "newNetwork",
    Description: "A network to deploy",
    Nodes:       []uint32{nodeID},
    IPRange: gridtypes.NewIPNet(net.IPNet{
      IP:   net.IPv4(10, 1, 0, 0),
      Mask: net.CIDRMask(16, 32),
    }),
    AddWGAccess: true,
}

// Create a new VM to deploy
vm := workloads.VM{
    Name:       "vm",
    Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
    CPU:        2,
    PublicIP:   true,
    Planetary:  true,
    Memory:     1024,
    RootfsSize: 20 * 1024,
    Entrypoint: "/sbin/zinit init",
    EnvVars: map[string]string{
        "SSH_KEY": publicKey,
    },
    IP:          "10.20.2.5",
    NetworkName: network.Name,
}

// Deploy the network first
err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)

// Load the network using the state loader
// this loader should load the deployment as json then convert it to a deployment go object with workloads inside it
networkObj, err := tfPluginClient.State.LoadNetworkFromGrid(network.Name)

// Deploy the VM deployment
dl := workloads.NewDeployment("vm", nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)

// Load the vm using the state loader
vmObj, err := tfPluginClient.State.LoadVMFromGrid(nodeID, vm.Name, dl.Name)

// Cancel the VM deployment
err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)

// Cancel the network
err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
```

Refer to [integration examples](./integration_tests) directory for more examples.

## Run tests

To run the tests, export MNEMONICS and NETWORK

```bash
export MNEMONICS="<mnemonics words>"
export NETWORK="<network>" # dev, qa or test
```

Run the following command

### running unit tests

```bash
make test
```

### running integration tests

```bash
make integration
```
