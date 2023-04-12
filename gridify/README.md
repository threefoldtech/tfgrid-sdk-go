# Gridify

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/cd6e18aac6be404ab89ec160b4b36671)](https://www.codacy.com/gh/threefoldtech/gridify/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=threefoldtech/gridify&amp;utm_campaign=Badge_Grade) <a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-92%25-brightgreen.svg?longCache=true&style=flat)</a> [![Testing](https://github.com/threefoldtech/gridify/actions/workflows/test.yml/badge.svg?branch=development)](https://github.com/threefoldtech/gridify/actions/workflows/test.yml) [![Testing](https://github.com/threefoldtech/gridify/actions/workflows/lint.yml/badge.svg?branch=development)](https://github.com/threefoldtech/gridify/actions/workflows/lint.yml) [![Dependabot](https://badgen.net/badge/Dependabot/enabled/green?icon=dependabot)](https://dependabot.com/)


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

## Release

- Check: `goreleaser check`
- Create a tag: `git tag -a v1.0.1 -m "release v1.0.1"`
- Push the tag: `git push origin v1.0.1`
- A goreleaser workflow will release the created tag.
