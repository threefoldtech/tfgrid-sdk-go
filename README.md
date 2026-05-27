# ZOS SDK Go

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/cd6e18aac6be404ab89ec160b4b36671)](https://www.codacy.com/gh/threefoldtech/tfgrid-sdk-go/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=threefoldtech/tfgrid-sdk-go&amp;utm_campaign=Badge_Grade) [![Dependabot](https://badgen.net/badge/Dependabot/enabled/green?icon=dependabot)](https://dependabot.com/) [![Lint](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/lint.yml/badge.svg?branch=development)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/lint.yml)
[![Test](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/test.yml/badge.svg?branch=development)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/test.yml) [![Build](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/build.yml/badge.svg?branch=development)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/build.yml)

Go client libraries and tools for deploying and managing workloads on the ThreeFold Grid. This SDK provides typed interfaces for grid operations including VM provisioning, network configuration, contract management, and node communication.

## What this is

This repository contains the official Go SDK for interacting with the ThreeFold Grid. It wraps the grid's APIs and on-chain interfaces into reusable Go packages that can be imported into applications, automation tools, and infrastructure providers. The SDK handles the complexity of grid discovery, contract creation, and remote procedure calls over the grid's message bus.

## What this repository contains

- [grid-client](./grid-client/README.md) — Core client library for deploying and managing grid workloads.
- [grid-proxy](./grid-proxy/README.md) — Proxy service and client for querying grid state and node information.
- [grid-cli](./grid-cli/README.md) — Command-line interface for grid operations.
- [rmb-sdk-go](./rmb-sdk-go/README.md) — Client for the Reliable Message Bus (RMB), the peer-to-peer communication layer used to send commands to grid nodes.
- [gridify](./gridify/README.md) — Tool for converting applications into grid-deployable formats.
- [monitoring-bot](./monitoring-bot/README.md) — Bot for monitoring grid services and alerts.
- [tfrobot](./tfrobot/README.md) — Automation tool for bulk grid deployments.
- [user-contracts-mon](./user-contracts-mon/README.md) — User contracts monitoring utility.
- [activation-service](./activation-service/README.md) — Service for activating new grid accounts.
- [farmerbot](./farmerbot/README.md) — Automation tool for farm and node management.

## Role in the stack

The SDK connects user-facing and backend Go applications to the ThreeFold Grid. It uses Ledger Chain for on-chain contracts and billing, the grid proxy for node discovery and statistics, and RMB for direct communication with nodes. The packages in this repository are used by the Terraform provider, CLI tools, monitoring systems, and other backend services within the broader infrastructure stack.

## Relation to ThreeFold

This technology is used within the ThreeFold ecosystem and was first deployed on the ThreeFold Grid. The component itself is designed as reusable infrastructure technology and should be understood by its technical function first, independent of any specific deployment.

## Ownership

This repository is owned and maintained by TF-Tech NV, a Belgian company responsible for the development and maintenance of this technology.

## Release

- [release document](./docs/release.md)
- [release script](./release.sh)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
Copyright (c) TF-Tech NV.
