# ZDBs

This document explains ZDBs related commands using tf-grid-cli.

## Deploy

```bash
tf-grid-cli deploy zdb [flags]
```

### Required Flags

- project_name: project name for the ZDBs deployment also used for canceling the deployment. must be unique.
- size: size of disk data in GB.

### Optional Flags

- node: node id zdbs should be deployed on.
- farm: farm id zdbs should be deployed on, if set choose available node from farm that fits zdbs deployment specs (default 1). note: node and farm flags cannot be set both.
- n: number of zdbs to be deployed (default 1).
- names: a slice of names for the number of ZDBs.
- password: passowrd for ZDBs deployed
- description: description for your ZDBs, it's optional.
- mode: the enumeration of the modes 0-db can operate in (default user).
- public: if the namespace of the ZDB is public (default false).

Example:

- Deploying ZDBs

```console
$ tf-grid-cli deploy zdb --project_name examplezdb --size=10 --n=2 --password=password
12:06PM INF deploying zdbs
12:06PM INF zdb 'examplezdb0' is deployed
12:06PM INF zdb 'examplezdb1' is deployed
```

## Get

```bash
tf-grid-cli get zdb <zdb-project-name>
```

zdb-project-name is the name of the deployment specified in while deploying using tf-grid-cli.

Example:

```console
$ tf-grid-cli get zdb examplezdb
3:20PM INF zdb:
{
        "Name": "examplezdb",
        "NodeID": 11,
        "SolutionType": "examplezdb",
        "SolutionProvider": null,
        "NetworkName": "",
        "Disks": [],
        "Zdbs": [
                {
                        "name": "examplezdb1",
                        "password": "password",
                        "public": false,
                        "size": 10,
                        "description": "",
                        "mode": "user",
                        "ips": [
                                "2a10:b600:1:0:c4be:94ff:feb1:8b3f",
                                "302:9e63:7d43:b742:469d:3ec2:ab15:f75e"
                        ],
                        "port": 9900,
                        "namespace": "81-36155-examplezdb1"
                },
                {
                        "name": "examplezdb0",
                        "password": "password",
                        "public": false,
                        "size": 10,
                        "description": "",
                        "mode": "user",
                        "ips": [
                                "2a10:b600:1:0:c4be:94ff:feb1:8b3f",
                                "302:9e63:7d43:b742:469d:3ec2:ab15:f75e"
                        ],
                        "port": 9900,
                        "namespace": "81-36155-examplezdb0"
                }
        ],
        "Vms": [],
        "QSFS": [],
        "NodeDeploymentID": {
                "11": 36155
        },
        "ContractID": 36155,
        "IPrange": ""
}
```

## Cancel

```bash
tf-grid-cli cancel <zdb-project-name>
```

zdb-project-name is the name of the deployment specified in while deploying using tf-grid-cli.

Example:

```console
$ tf-grid-cli cancel examplezdb
3:37PM INF canceling contracts for project examplezdb
3:37PM INF examplezdb canceled
```
