# Farmerbot

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-73%25-brightgreen.svg?longCache=true&style=flat)</a>

Farmerbot is a service that a farmer can run allowing him to automatically manage power of the nodes of his farm.

## How to use

- Create a new yml/yaml file `config.yml` and add your configurations:

```yml
farm_id: "<your farm ID, required>"
included_nodes:
  - "<your node ID to be included, required at least 2>"
excluded_nodes:
  - "<your node ID to be excluded, optional>"
never_shutdown_nodes:
  - "<your node ID to be never shutdown, optional>"
power:
  periodic_wake_up_start: "<daily time to wake up nodes for your farm, default is the time your run the command, format is 00:00AM or 00:00PM, optional>"
  wake_up_threshold: "<the threshold number for resources usage that will need another node to be on, default is 80, optional>"
  periodic_wake_up_limit: "<the number (limit) of nodes to be waken up everyday, default is 1, optional>"
  overprovision_cpu: "<how much node allows over provisioning the CPU , default is 1, range: [1;4], optional>"
```

- [Download](#download) farmerbot binaries.

- Run the bot

```bash
farmerbot run -c config.json -m <mnemonic> -n dev -d
```

Where:

```bash
Flags:
-c, --config string     your config file that includes your farm, node and power configs. Available format is yml/yaml

Global Flags:
-d, --debug             by setting this flag the farmerbot will print debug logs too
-m, --mnemonic string   the mnemonic of the account of the farmer
-n, --network string    the grid network to use (default "main")
-s, --seed string       the hex seed of the account of the farmer
```

> Note: you should only provide **`mnemonic`** or **`seed`**

## Download

- Download the binaries from [releases](https://github.com/threefoldtech/tfgrid-sdk-go/releases)
- Extract the downloaded files
- Move the binary to any of `$PATH` directories, for example:

```bash
mv farmerbot /usr/local/bin
```

## Build

Run the following command inside the directory:

```bash
make build
```

## Supported commands

- `start`: to start (power on) a node

```bash
farmerbot start --node <node ID> -m <mnemonic> -n dev -d
```

Where:

```bash
Flags:
    --node uint32       the node ID you want to use

Global Flags:
-d, --debug             by setting this flag the farmerbot will print debug logs too
-m, --mnemonic string   the mnemonic of the account of the farmer
-n, --network string    the grid network to use (default "main")
-s, --seed string       the hex seed of the account of the farmer
```

- `start all`:  to start (power on) all nodes in a farm

```bash
farmerbot start all --farm <farm ID> -m <mnemonic> -n dev -d
```

Where:

```bash
Flags:
    --farm uint32       the farm ID you want to start your nodes ins

Global Flags:
-d, --debug             by setting this flag the farmerbot will print debug logs too
-m, --mnemonic string   the mnemonic of the account of the farmer
-n, --network string    the grid network to use (default "main")
-s, --seed string       the hex seed of the account of the farmer
```

- `version`: to get the current version of farmerbot

```bash
farmerbot version
```

## Calls

Calls can be send to the farmerbot via RMB. This section describes the arguments that they accept.

### farmerbot.nodemanager.findnode

This call allows you to look for a node with specific requirements (minimum amount of resources, public config, etc). You will get the node id as a result. The farmerbot will power on the node if the node is off. It will also claim the required resources for 30 minutes. After that, if the user has not deployed anything on the node the resources will be freed and the node might go down again if it was put on by that call.

Arguments (all arguments are optional):

- _has_gpus_ => if you require one or more gpus you can filter on that with this parameter (should be a positive value)
- _gpu_vendors_ => a list of strings that will be used to filter the nodes on gpu vendor (for example AMD)
- _gpu_devices_ => a list of strings that will be used to filter the nodes on gpu device (for example GTX 1080)
- _certified_ => whether or not you want a certified node (not adding this argument means you don't care whether you get a certified or non certified node)
- _public_config_ => whether or not you want a node with a public config (not adding this argument means you don't care whether or not the node has a public config)
- _public_ips_ => how much public ips you need
- _dedicated_ => whether you want a dedicated node (rent the full node)
- _node_exclude_ => the list of node ids you want to exclude in your search
- _hru_ => the amount of hru required in kilobytes
- _sru_ => the amount of sru required in kilobytes
- _mru_ => the amount of mru required in kilobytes
- _cru_ => the amount of cru required

Result:

- `node_id` => the node id that meets your requirements

Example:

- [findnode](./examples/findnode/main.go)

### farmerbot.powermanager.poweron

This call is only allowed to be executed if it comes from the farmer (the twin ID should equal the farmer's twin ID). It will power on the node specified in the arguments. After powering on a node it will be excluded from farmerbot management

Arguments:

- _node_id_ => the node id of the node that needs to powered on

Example:

- [power on](./examples/poweron/main.go)

### farmerbot.powermanager.poweroff

This call is only allowed to be executed if it comes from the farmer (the twin ID should equal the farmer's twin ID). It will power off the node specified in the arguments. After powering off a node it will be excluded from farmerbot management

Arguments:

- _node_id_ => the node id of the node that needs to powered off

Example:

- [power off](./examples/poweroff/main.go)

### farmerbot.farmmanager.version

This call returns the current version of the farmerbot

Example:

- [version](./examples/version/main.go)

### farmerbot.powermanager.includenode

This call is only allowed to be executed if it comes from the farmer (the twin ID should equal the farmer's twin ID). It will include an excluded node from power on and off calls (it should be included in the farmerbot configurations)

Arguments:

- _node_id_ => the node id of the node that needs to be included

Example:

- [include node](./examples/includenode/main.go)

## Examples

Check the [examples](/examples)

To run examples:

- Don't forget to write your mnemonic in the example

```bash
cd <example>
go run main.go
```

## Test

Run the following command inside the directory:

```bash
make test
```
