# Gridify

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-92%25-brightgreen.svg?longCache=true&style=flat)</a> [![Testing](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/gridify-test.yml/badge.svg?branch=development_mono)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/gridify-test.yml) [![Testing](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/gridify-lint.yml/badge.svg?branch=development_mono)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/gridify-lint.yml)

A tool used to deploy projects on [Threefold grid](https://threefold.io/).

## Usage

First [download](#download) gridify binaries.

Login using your [mnemonics](https://threefoldtech.github.io/info_grid/dashboard/portal/dashboard_portal_polkadot_create_account.html) and specify which grid network (mainnet/testnet) to deploy on by running:

```bash
gridify login
```

Use `gridify` to deploy your project and specify the ports you want gridify to assign domains to:

```bash
gridify deploy --ports <ports>
```

ports are your services' ports defined in Procfile

for example:

```bash
gridify deploy --ports 80,8080
```

gridify generates a unique domain for each service.

To get your domains for the deployed project, use the following:

```bash
gridify get
```

To destroy deployed project run the following command inside the project directory:

```bash
gridify destroy
```

## Download

- Download the binaries from [releases](https://github.com/threefoldtech/gridify/releases)
- Extract the downloaded files
- Move the binary to any of `$PATH` directories, for example:

```bash
mv gridify /usr/local/bin
```

## Configuration

Gridify saves user configuration in `.gridifyconfig` under default configuration directory for your system see: [UserConfigDir()](https://pkg.go.dev/os#UserConfigDir)

## Requirements

- gridify uses [ginit](https://github.com/rawdaGastan/ginit) so Procfile and env must exist in root directory of your project see: [Demo](#gridify-demo-project)
- the project github repository must be public

## Gridify Demo Project

See [gridify-demo](https://github.com/AbdelrahmanElawady/gridify-demo)

In this demo gridify deploys a VM with [flist](https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-gridify-test-latest.flist.md) that clones the demo project and run each service defined in Procfile. Then, gridify assign a domain for each service.

## Supported Projects Languages and Tools

- go 1.18
- python 3.10.10
- node 16.17.1
- npm 8.10.0
- caddy

## Testing

For unittests run:

```bash
make test
```

## Build

Clone the repo and run the following command inside the repo directory:

```bash
make build
```
