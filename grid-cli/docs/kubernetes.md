# Kubernetes

This document explains Kubernetes related commands using tf-grid-cli.

## Deploy

```bash
tf-grid-cli deploy kubernetes [flags]
```

### Required Flags

- name: name for the master node deployment also used for canceling the cluster deployment. must be unique.
- ssh: path to public ssh key to set in the master node.

### Optional Flags

- master-node: node id master should be deployed on.
- master-farm: farm id master should be deployed on, if set choose available node from farm that fits master specs (default 1). note: master-node and master-farm flags cannot be set both.
- workers-node: node id workers should be deployed on.
- workers-farm: farm id workers should be deployed on, if set choose available node from farm that fits master specs (default 1). note: workers-node and workers-farm flags cannot be set both.
- ipv4: assign public ipv4 for master node (default false).
- ipv6: assign public ipv6 for master node (default false).
- ygg: assign yggdrasil ip for master node (default true).
- master-cpu: number of cpu units for master node (default 1).
- master-memory: master node memory size in GB (default 1).
- master-disk: master node disk size in GB (default 2).
- workers-number: number of workers nodes (default 0).
- workers-ipv4: assign public ipv4 for each worker node (default false)
- workers-ipv6: assign public ipv6 for each worker node (default false)
- workers-ygg: assign yggdrasil ip for each worker node (default true)
- workers-cpu: number of cpu units for each worker node (default 1).
- workers-memory: memory size for each worker node in GB (default 1).
- workers-disk: disk size in GB for each worker node (default 2).

Example:

```console
$ tf-grid-cli deploy kubernetes -n kube --ssh ~/.ssh/id_rsa.pub --master-node 14 --workers-number 2 --workers-node 14
4:21PM INF deploying network
4:22PM INF deploying cluster
4:22PM INF master yggdrasil ip: 300:e9c4:9048:57cf:504f:c86c:9014:d02d
```

## Get

```bash
tf-grid-cli get kubernetes <kubernetes>
```

kubernetes is the name used when deploying kubernetes cluster using tf-grid-cli.

Example:

```console
$ tf-grid-cli get kubernetes examplevm
3:14PM INF k8s cluster:
{
        "Master": {
                "Name": "kube",
                "Node": 14,
                "DiskSize": 2,
                "PublicIP": false,
                "PublicIP6": false,
                "Planetary": true,
                "Flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
                "FlistChecksum": "c87cf57e1067d21a3e74332a64ef9723",
                "ComputedIP": "",
                "ComputedIP6": "",
                "YggIP": "300:e9c4:9048:57cf:e8a0:662b:4e66:8faa",
                "IP": "10.20.2.2",
                "CPU": 1,
                "Memory": 1024
        },
        "Workers": [
                {
                        "Name": "worker1",
                        "Node": 14,
                        "DiskSize": 2,
                        "PublicIP": false,
                        "PublicIP6": false,
                        "Planetary": true,
                        "Flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
                        "FlistChecksum": "c87cf57e1067d21a3e74332a64ef9723",
                        "ComputedIP": "",
                        "ComputedIP6": "",
                        "YggIP": "300:e9c4:9048:57cf:66d0:3ee4:294e:d134",
                        "IP": "10.20.2.2",
                        "CPU": 1,
                        "Memory": 1024
                },
                {
                        "Name": "worker0",
                        "Node": 14,
                        "DiskSize": 2,
                        "PublicIP": false,
                        "PublicIP6": false,
                        "Planetary": true,
                        "Flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
                        "FlistChecksum": "c87cf57e1067d21a3e74332a64ef9723",
                        "ComputedIP": "",
                        "ComputedIP6": "",
                        "YggIP": "300:e9c4:9048:57cf:1ae5:cc51:3ffc:81e",
                        "IP": "10.20.2.2",
                        "CPU": 1,
                        "Memory": 1024
                }
        ],
        "Token": "",
        "NetworkName": "",
        "SolutionType": "kube",
        "SSHKey": "",
        "NodesIPRange": null,
        "NodeDeploymentID": {
                "14": 22743
        }
}
```

## Cancel

```bash
tf-grid-cli cancel <deployment-name>
```

deployment-name is the name of the deployment specified in while deploying using tf-grid-cli.

Example:

```console
$ tf-grid-cli cancel kube
3:37PM INF canceling contracts for project kube
3:37PM INF kube canceled
```
