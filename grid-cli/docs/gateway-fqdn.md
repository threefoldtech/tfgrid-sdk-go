# Gateway FQDN

This document explains Gateway FQDN related commands using tf-grid-cli.

## Deploy

```bash
tf-grid-cli deploy gateway fqdn [flags]
```

### Required Flags

- name: name for the gateway deployment also used for canceling the deployment. must be unique.
- node: node id to deploy gateway on.
- backends: list of backends the gateway will forward requests to.
- fqdn: FQDN pointing to the specified node.

### Optional Flags

-tls: add TLS passthrough option (default false).

Example:

```console
$ tf-grid-cli deploy gateway fqdn -n gatewaytest --node 14 --backends http://93.184.216.34:80 --fqdn example.com
3:34PM INF deploying gateway fqdn
3:34PM INF gateway fqdn deployed
```

## Get

```bash
tf-grid-cli get gateway fqdn <gateway>
```

gateway is the name used when deploying gateway-fqdn using tf-grid-cli.

Example:

```console
$ tf-grid-cli get gateway fqdn gatewaytest
2:05PM INF gateway fqdn:
{
        "NodeID": 14,
        "Backends": [
                "http://93.184.216.34:80"
        ],
        "FQDN": "awady.gridtesting.xyz",
        "Name": "gatewaytest",
        "TLSPassthrough": false,
        "Description": "",
        "NodeDeploymentID": {
                "14": 19653
        },
        "SolutionType": "gatewaytest",
        "ContractID": 19653
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
