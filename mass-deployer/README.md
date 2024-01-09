# mass-deployer

Mass Deplyer tool designed to automate mass deployment of groups of VMs on ThreeFold Grid.

## Features

-   **Mass Deployment:** Deploy groups of vms on ThreeFold Grid simultaneously.
-   **Customizable Configurations:** Define Node groups, VMs groups and other configurations through YAML file.

## Usage
1.  First [download](#download) mass-deployer binaries.
2.  Create a new configuration file.

```yaml
node_groups:
  - name: example-group
    nodes_count: 3
    free_cpu: 8
    free_mru: 16384
    # ... other fields

vms:
  - name: example-vm
    vms_count: 2
    node_group: example-group
    cpu: 2
    mem: 4096
    flist: example-flist,
    entry_point: /sbin/zinit init
    # ... other fields

sshkey: example-ssh-key
mnemonic: example-mnemonic
network: example-network
```
> Make sure to replace placeholders and adapt the groups based on your actual project details.

3.  Run the deployer with path to the config file
```bash
$ mass-deployer -c path/to/your/config.yaml
```

## Download

-   Download the binaries from [releases](https://github.com/threefoldtech/tfgrid-sdk-go/releases)
-   Extract the downloaded files
-   Move the binary to any of `$PATH` directories, for example:

```bash
mv mass-deployer /usr/local/bin
```

## Build

To build the deployer locally clone the repo and run the following command inside the repo directory:

```bash
make build
```
