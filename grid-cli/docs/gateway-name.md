# Gateway Name

This document explains Gateway Name related commands using tf-grid-cli.

## Deploy

```bash
tf-grid-cli deploy gateway name [flags]
```

### Required Flags

- name: name for the gateway deployment also used for canceling the deployment. must be unique.
- backends: list of backends the gateway will forward requests to.

### Optional Flags

- node: node id gateway should be deployed on.
- farm: farm id gateway should be deployed on, if set choose available node from farm that fits vm specs (default 1). note: node and farm flags cannot be set both.
-tls: add TLS passthrough option (default false).

Example:

```console
$ tf-grid-cli deploy gateway name -n gatewaytest --node 14 --backends http://93.184.216.34:80
3:34PM INF deploying gateway name
3:34PM INF fqdn: gatewaytest.gent01.dev.grid.tf
```

## Get

```bash
tf-grid-cli get gateway name <gateway>
```

gateway is the name used when deploying gateway-name using tf-grid-cli.

Example:

```console
$ tf-grid-cli get gateway gatewaytest
1:56PM INF gateway name:
{
        "NodeID": 14,
        "Name": "gatewaytest",
        "Backends": [
                "http://93.184.216.34:80"
        ],
        "TLSPassthrough": false,
        "Description": "",
        "SolutionType": "gatewaytest",
        "NodeDeploymentID": {
                "14": 19644
        },
        "FQDN": "gatewaytest.gent01.dev.grid.tf",
        "NameContractID": 19643,
        "ContractID": 19644
}
```

## Cancel

```bash
tf-grid-cli cancel <deployment-name>
```

deployment-name is the name of the deployment specified in while deploying using tf-grid-cli.

Example:

```console
$ tf-grid-cli cancel gatewaytest
3:37PM INF canceling contracts for project gatewaytest
3:37PM INF gatewaytest canceled
```
