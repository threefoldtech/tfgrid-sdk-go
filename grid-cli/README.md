# tf-grid-cli

Threefold CLI to manage deployments on Threefold Grid.

## Usage

First [download](#download) tf-grid-cli binaries.

Login using your [mnemonics](https://threefoldtech.github.io/info_grid/dashboard/portal/dashboard_portal_polkadot_create_account.html) and specify which grid network (mainnet/testnet) to deploy on by running:

```bash
tf-grid-cli login
```

For examples and description of tf-grid-cli commands check out:

- [vm](docs/vm.md)
- [gateway-fqdn](docs/gateway-fqdn.md)
- [gateway-name](docs/gateway-name.md)
- [kubernetes](docs/kubernetes.md)

## Download

- Download the binaries from [releases](https://github.com/threefoldtech/tfgrid-sdk-go/grid-cli/releases)
- Extract the downloaded files
- Move the binary to any of `$PATH` directories, for example:

```bash
mv tf-grid-cli /usr/local/bin
```

## Configuration

tf-grid saves user configuration in `.tfgridconfig` under default configuration directory for your system see: [UserConfigDir()](https://pkg.go.dev/os#UserConfigDir)

## Build

Clone the repo and run the following command inside the repo directory:

```bash
make build
```
