# mass-deployer

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-90%25-brightgreen.svg?longCache=true&style=flat)</a>

Mass Deplyer tool is designed to automate mass deployment of groups of VMs on ThreeFold Grid.

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

You can use this [example](./example/conf.yaml) for guidance, and make sure to replace placeholders and adapt the groups based on your actual project details.

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

## Test

To run the deployer tests run the following command inside the repo directory:

```bash
make test
```
