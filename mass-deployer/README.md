# tfrobot

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-88%25-brightgreen.svg?longCache=true&style=flat)</a>

tfrobot is tool designed to automate mass deployment of groups of VMs on ThreeFold Grid, with support of multiple retries for failed deployments

## Features

-   **Mass Deployment:** Deploy groups of vms on ThreeFold Grid simultaneously.
-   **Mass Cancelation:** cancel all vms on ThreeFold Grid defined in configuration file simultaneously.
-   **Customizable Configurations:** Define Node groups, VMs groups and other configurations through YAML or JSON file.

## Download

1.  Download the binaries from [releases](https://github.com/threefoldtech/tfgrid-sdk-go/releases)
2.  Extract the downloaded files
3.  Move the binary to any of `$PATH` directories, for example:
```bash
mv tfrobot /usr/local/bin
```
4.  Create a new configuration file.
For example:
```yaml
node_groups:
  - name: group_a
    nodes_count: 3
    free_cpu: 2
    free_mru: 16384
    free_ssd: 100
    free_hdd: 50
vms:
  - name: examplevm
    vms_count: 5
    node_group: group_a
    cpu: 1
    mem: 256
    flist: example-flist
    entry_point: example-entrypoint
    root_size: 0
    ssh_key: example1
    env_vars:
      user: user1
      pwd: 1234
ssh_keys:
  example1: ssh_key1
mnemonic: example-mnemonic
network: dev
```

You can use this [example](./example/conf.yaml) for further guidance, 
>**Please** make sure to replace placeholders and adapt the groups based on your actual project details.

>**Note:** All storage resources are expected to be in GB, except of memory in MB to be able to deploy vm with a fraction of GB of memory(256 MB for example)

5.  Run the deployer with path to the config file
```bash
tfrobot deploy -c path/to/your/config.yaml
```

## Usage
### Subcommands:

-   **deploy:** used to mass deploy groups of vms with specific configurations
```bash
tfrobot deploy -c path/to/your/config.yaml
```

-   **cancel:** used to cancel all vms deployed using specific configurations
```bash
tfrobot cancel -c path/to/your/config.yaml
```

### Flags:
| Flag | Usage |
| :---:   | :---: |
| -c | used to specify path to configuration file |
| -o | used to specify path to output file to store the output info in |
>Parsing is based on file extension, json format if the file had json extension, yaml format otherwise 

## Using Docker
```bash
docker build -t tfrobot -f Dockerfile ../
docker run -v $(pwd)/config.yaml:/config.yaml -it tfrobot:latest deploy -c /config.yaml
```

## Build

To build the deployer locally clone the repo and run the following command inside the repo directory:

```bash
make build
```

## Test

To run the deployer tests run the following command inside the repo directory:

```bash
make test
```
