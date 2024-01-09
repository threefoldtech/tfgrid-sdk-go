# mass-deployer

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-90%25-brightgreen.svg?longCache=true&style=flat)</a>

Mass Deplyer tool designed to automate mass deployment of groups of VMs on ThreeFold Grid.

## Features

-   **Mass Deployment:** Deploy groups of vms on ThreeFold Grid simultaneously.
-   **Customizable Configurations:** Define Node groups, VMs groups and other configurations through YAML file.

## Download

1.  Download the binaries from [releases](https://github.com/threefoldtech/tfgrid-sdk-go/releases)
2.  Extract the downloaded files
3.  Move the binary to any of `$PATH` directories, for example:
```bash
mv mass-deployer /usr/local/bin
```
4.  Create a new configuration file.

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

5.  Run the deployer with path to the config file
```bash
$ mass-deployer -c path/to/your/config.yaml
```

## Using Docker
```bash
docker build -t mass-deployer -f Dockerfile ../
docker run -v $(pwd)/config.yaml:/config.yaml -it mass-deployer:latest -c /config.yaml
```

## Build

To build the deployer locally clone the repo and run the following command inside the repo directory:

```bash
make build
```
