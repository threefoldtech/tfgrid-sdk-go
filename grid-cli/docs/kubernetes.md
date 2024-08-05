# Kubernetes

This document explains Kubernetes related commands using tfcmd.

## Deploy

```bash
tfcmd deploy kubernetes [flags]
```

### Required Flags

- name: name for the master node deployment also used for canceling the cluster deployment. must be unique.
- ssh: path to public ssh key to set in the master node.

### Optional Flags

- master-node: node id master should be deployed on.
- master-farm: farm id master should be deployed on, if set choose available node from farm that fits master specs (default 1). note: master-node and master-farm flags cannot be set both.
- workers-nodes: array of nodes ids workers should be deployed on and the remaining unassigned workers will be randomly assigned to nodes that meet the specifications.
- workers-farm: farm id workers should be deployed on, if set choose available node from farm that fits master specs (default 1). note: workers-node and workers-farm flags cannot be set both.
- ipv4: assign public ipv4 for master node (default false).
- ipv6: assign public ipv6 for master node (default false).
- ygg: assign yggdrasil ip for master node (default true).
- mycelium: assign mycelium ip for master node (default true).
- master-cpu: number of cpu units for master node (default 1).
- master-memory: master node memory size in GB (default 1).
- master-disk: master node disk size in GB (default 2).
- workers-number: number of workers nodes (default 0).
- workers-ipv4: assign public ipv4 for each worker node (default false)
- workers-ipv6: assign public ipv6 for each worker node (default false)
- workers-ygg: assign yggdrasil ip for each worker node (default true)
- workers-mycelium: assign mycelium ip for each worker node (default true)
- workers-cpu: number of cpu units for each worker node (default 1).
- workers-memory: memory size for each worker node in GB (default 1).
- workers-disk: disk size in GB for each worker node (default 2).

Example:

```console
$ tfcmd deploy kubernetes -n kube --ssh ~/.ssh/id_rsa.pub --master-node 14 --workers-number 2 --workers-nodes 14,1
11:43AM INF starting peer session=tf-1510734 twin=192
11:43AM INF deploying network
11:43AM INF deploying cluster
11:43AM INF master wireguard ip: 10.20.2.2
11:43AM INF master planetary ip: 300:e9c4:9048:57cf:d73d:eb4c:7a6d:503
11:43AM INF master mycelium ip: 423:16f5:ca74:b600:ff0f:9ee:d57e:827a
11:43AM INF worker1 wireguard ip: 10.20.2.3
11:43AM INF worker0 wireguard ip: 10.20.2.3
11:43AM INF worker1 planetary ip: 300:e9c4:9048:57cf:3c4f:d477:b4a5:890b
11:43AM INF worker0 planetary ip: 300:e9c4:9048:57cf:77ca:5424:21da:4fff
11:43AM INF worker1 mycelium ip: 423:16f5:ca74:b600:ff0f:e02f:1ad7:d74e
11:43AM INF worker0 mycelium ip: 423:16f5:ca74:b600:ff0f:bb5a:6036:56f6
```

## Update

### Add workers

```bash
tfcmd apdate kubernetes add [flags]
```

### Required Flags

- name: name for the master node deployment also used for canceling the cluster deployment. must be unique.
- ssh: path to public ssh key to set in the master node.

### Optional Flags

- workers-number: number of workers to be added to the cluster (default = 1)
- workers-nodes: array of nodes ids workers should be deployed on and the remaining unassigned workers will be randomly assigned to nodes that meet the specifications.
- workers-farm: farm id workers should be deployed on, if set choose available node from farm that fits master specs (default 1). note: workers-node and workers-farm flags cannot be set both.
- workers-ipv4: assign public ipv4 for each worker node (default false)
- workers-ipv6: assign public ipv6 for each worker node (default false)
- workers-ygg: assign yggdrasil ip for each worker node (default true)
- workers-mycelium: assign mycelium ip for each worker node (default true)
- workers-cpu: number of cpu units for each worker node (default 1).
- workers-memory: memory size for each worker node in GB (default 1).
- workers-disk: disk size in GB for each worker node (default 2).

Example:

```console
➜  grid-cli git:(development-support-updating-k8s-cluster-grid-cli) ✗ go run main.go deploy kubernetes --name test --ssh ~/.ssh/id_rsa.pub --workers-number 1 --master-memory 2 --mycelium
11:57AM INF starting peer session=tf-21418 twin=4653
11:57AM INF deploying network
11:57AM INF deploying cluster
11:58AM INF master wireguard ip: 10.20.2.2
11:58AM INF master planetary ip: 302:9e63:7d43:b742:8e80:99b8:a08f:98da
11:58AM INF master mycelium ip: 58b:d053:77f9:37cd:ff0f:b0bf:828d:5c60
11:58AM INF worker0 wireguard ip: 10.20.3.2
11:58AM INF worker0 planetary ip: 301:1037:16ee:23fb:83f6:d82e:aad:cf06
11:58AM INF worker0 mycelium ip: 59f:354e:418c:5027:ff0f:357a:f019:73d6

➜  grid-cli git:(development-support-updating-k8s-cluster-grid-cli) ✗ go run main.go update kubernetes add --name test --ssh ~/.ssh/id_rsa.pub --workers-number 2
12:00PM INF starting peer session=tf-21803 twin=4653
12:00PM INF updating network
12:00PM INF updating cluster
12:00PM INF master wireguard ip: 10.20.2.2
12:00PM INF master planetary ip: 302:9e63:7d43:b742:8e80:99b8:a08f:98da
12:00PM INF master mycelium ip: 58b:d053:77f9:37cd:ff0f:b0bf:828d:5c60
12:00PM INF worker2 wireguard ip: 10.20.2.3
12:00PM INF worker1 wireguard ip: 10.20.3.2
12:00PM INF worker0 wireguard ip: 10.20.3.2
12:00PM INF worker2 planetary ip: 302:9e63:7d43:b742:871b:b2d2:cc74:ab12
12:00PM INF worker1 planetary ip: 301:1037:16ee:23fb:b72d:497a:95d3:443f
12:00PM INF worker0 planetary ip: 301:1037:16ee:23fb:83f6:d82e:aad:cf06
12:00PM INF worker2 mycelium ip: 418:fdd0:8715:c1c2:ff0f:b43a:e4:a323
12:00PM INF worker1 mycelium ip: 523:7612:fa7c:d8bf:ff0f:6c50:2200:7569
12:00PM INF worker0 mycelium ip: 59f:354e:418c:5027:ff0f:357a:f019:73d6
```

### Delete worker

```bash
tfcmd apdate kubernetes delete [flags]
```

### Required Flags

- name: name for the master node deployment also used for canceling the cluster deployment. must be unique.

- worker-name: name of the worker to be deleted

Example:

```console
➜  grid-cli git:(development-support-updating-k8s-cluster-grid-cli) ✗ go run main.go update kubernetes delete --name test --worker-name worker0
12:34PM INF starting peer session=tf-25294 twin=4653
12:34PM INF updating network
12:35PM INF updating cluster
12:35PM INF master wireguard ip: 10.20.2.2
12:35PM INF master planetary ip: 302:9e63:7d43:b742:8e80:99b8:a08f:98da
12:35PM INF master mycelium ip: 58b:d053:77f9:37cd:ff0f:b0bf:828d:5c60
12:35PM INF worker2 wireguard ip: 10.20.2.3
12:35PM INF worker1 wireguard ip: 10.20.3.2
12:35PM INF worker2 planetary ip: 302:9e63:7d43:b742:871b:b2d2:cc74:ab12
12:35PM INF worker1 planetary ip: 301:1037:16ee:23fb:b72d:497a:95d3:443f
12:35PM INF worker2 mycelium ip: 418:fdd0:8715:c1c2:ff0f:b43a:e4:a323
12:35PM INF worker1 mycelium ip: 523:7612:fa7c:d8bf:ff0f:6c50:2200:7569
```

## Get

```bash
tfcmd get kubernetes <kubernetes>
```

kubernetes is the name used when deploying kubernetes cluster using tfcmd.

Example:

```console
$ tfcmd get kubernetes kube
11:44AM INF starting peer session=tf-1511628 twin=192
11:44AM INF k8s cluster:
{
        "Master": {
                "name": "kube",
                "node": 14,
                "disk_size": 10,
                "publicip": false,
                "publicip6": false,
                "planetary": true,
                "flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
                "flist_checksum": "c87cf57e1067d21a3e74332a64ef9723",
                "computedip": "",
                "computedip6": "",
                "planetary_ip": "300:e9c4:9048:57cf:d73d:eb4c:7a6d:503",
                "mycelium_ip": "423:16f5:ca74:b600:ff0f:9ee:d57e:827a",
                "mycelium_ip_seed": "Ce7VfoJ6",
                "ip": "10.20.2.2",
                "cpu": 2,
                "memory": 4096,
                "network_name": "kubenetwork",
                "token": "securetoken",
                "ssh_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDcGrS1RT36rHAGLK3/4FMazGXjIYgWVnZ4bCvxxg8KosEEbs/DeUKT2T2LYV91jUq3yibTWwK0nc6O+K5kdShV4qsQlPmIbdur6x2zWHPeaGXqejbbACEJcQMCj8szSbG8aKwH8Nbi8BNytgzJ20Ysaaj2QpjObCZ4Ncp+89pFahzDEIJx2HjXe6njbp6eCduoA+IE2H9vgwbIDVMQz6y/TzjdQjgbMOJRTlP+CzfbDBb6Ux+ed8F184bMPwkFrpHs9MSfQVbqfIz8wuq/wjewcnb3wK9dmIot6CxV2f2xuOZHgNQmVGratK8TyBnOd5x4oZKLIh3qM9Bi7r81xCkXyxAZbWYu3gGdvo3h85zeCPGK8OEPdYWMmIAIiANE42xPmY9HslPz8PAYq6v0WwdkBlDWrG3DD3GX6qTt9lbSHEgpUP2UOnqGL4O1+g5Rm9x16HWefZWMjJsP6OV70PnMjo9MPnH+yrBkXISw4CGEEXryTvupfaO5sL01mn+UOyE= abdulrahman@AElawady-PC\n",
                "console_url": "10.20.2.0:20002"
        },
        "Workers": [
                {
                        "name": "worker1",
                        "node": 14,
                        "disk_size": 10,
                        "publicip": false,
                        "publicip6": false,
                        "planetary": true,
                        "flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
                        "flist_checksum": "c87cf57e1067d21a3e74332a64ef9723",
                        "computedip": "",
                        "computedip6": "",
                        "planetary_ip": "300:e9c4:9048:57cf:3c4f:d477:b4a5:890b",
                        "mycelium_ip": "423:16f5:ca74:b600:ff0f:e02f:1ad7:d74e",
                        "mycelium_ip_seed": "4C8a19dO",
                        "ip": "10.20.2.3",
                        "cpu": 2,
                        "memory": 4096,
                        "network_name": "kubenetwork",
                        "token": "securetoken",
                        "ssh_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDcGrS1RT36rHAGLK3/4FMazGXjIYgWVnZ4bCvxxg8KosEEbs/DeUKT2T2LYV91jUq3yibTWwK0nc6O+K5kdShV4qsQlPmIbdur6x2zWHPeaGXqejbbACEJcQMCj8szSbG8aKwH8Nbi8BNytgzJ20Ysaaj2QpjObCZ4Ncp+89pFahzDEIJx2HjXe6njbp6eCduoA+IE2H9vgwbIDVMQz6y/TzjdQjgbMOJRTlP+CzfbDBb6Ux+ed8F184bMPwkFrpHs9MSfQVbqfIz8wuq/wjewcnb3wK9dmIot6CxV2f2xuOZHgNQmVGratK8TyBnOd5x4oZKLIh3qM9Bi7r81xCkXyxAZbWYu3gGdvo3h85zeCPGK8OEPdYWMmIAIiANE42xPmY9HslPz8PAYq6v0WwdkBlDWrG3DD3GX6qTt9lbSHEgpUP2UOnqGL4O1+g5Rm9x16HWefZWMjJsP6OV70PnMjo9MPnH+yrBkXISw4CGEEXryTvupfaO5sL01mn+UOyE= abdulrahman@AElawady-PC\n",
                        "console_url": "10.20.2.0:20003"
                },
                {
                        "name": "worker0",
                        "node": 14,
                        "disk_size": 10,
                        "publicip": false,
                        "publicip6": false,
                        "planetary": true,
                        "flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
                        "flist_checksum": "c87cf57e1067d21a3e74332a64ef9723",
                        "computedip": "",
                        "computedip6": "",
                        "planetary_ip": "300:e9c4:9048:57cf:77ca:5424:21da:4fff",
                        "mycelium_ip": "423:16f5:ca74:b600:ff0f:bb5a:6036:56f6",
                        "mycelium_ip_seed": "u1pgNlb2",
                        "ip": "10.20.2.3",
                        "cpu": 2,
                        "memory": 4096,
                        "network_name": "kubenetwork",
                        "token": "securetoken",
                        "ssh_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDcGrS1RT36rHAGLK3/4FMazGXjIYgWVnZ4bCvxxg8KosEEbs/DeUKT2T2LYV91jUq3yibTWwK0nc6O+K5kdShV4qsQlPmIbdur6x2zWHPeaGXqejbbACEJcQMCj8szSbG8aKwH8Nbi8BNytgzJ20Ysaaj2QpjObCZ4Ncp+89pFahzDEIJx2HjXe6njbp6eCduoA+IE2H9vgwbIDVMQz6y/TzjdQjgbMOJRTlP+CzfbDBb6Ux+ed8F184bMPwkFrpHs9MSfQVbqfIz8wuq/wjewcnb3wK9dmIot6CxV2f2xuOZHgNQmVGratK8TyBnOd5x4oZKLIh3qM9Bi7r81xCkXyxAZbWYu3gGdvo3h85zeCPGK8OEPdYWMmIAIiANE42xPmY9HslPz8PAYq6v0WwdkBlDWrG3DD3GX6qTt9lbSHEgpUP2UOnqGL4O1+g5Rm9x16HWefZWMjJsP6OV70PnMjo9MPnH+yrBkXISw4CGEEXryTvupfaO5sL01mn+UOyE= abdulrahman@AElawady-PC\n",
                        "console_url": "10.20.2.0:20003"
                }
        ],
        "Token": "securetoken",
        "NetworkName": "kubenetwork",
        "SolutionType": "kubernetes/kube",
        "SSHKey": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDcGrS1RT36rHAGLK3/4FMazGXjIYgWVnZ4bCvxxg8KosEEbs/DeUKT2T2LYV91jUq3yibTWwK0nc6O+K5kdShV4qsQlPmIbdur6x2zWHPeaGXqejbbACEJcQMCj8szSbG8aKwH8Nbi8BNytgzJ20Ysaaj2QpjObCZ4Ncp+89pFahzDEIJx2HjXe6njbp6eCduoA+IE2H9vgwbIDVMQz6y/TzjdQjgbMOJRTlP+CzfbDBb6Ux+ed8F184bMPwkFrpHs9MSfQVbqfIz8wuq/wjewcnb3wK9dmIot6CxV2f2xuOZHgNQmVGratK8TyBnOd5x4oZKLIh3qM9Bi7r81xCkXyxAZbWYu3gGdvo3h85zeCPGK8OEPdYWMmIAIiANE42xPmY9HslPz8PAYq6v0WwdkBlDWrG3DD3GX6qTt9lbSHEgpUP2UOnqGL4O1+g5Rm9x16HWefZWMjJsP6OV70PnMjo9MPnH+yrBkXISw4CGEEXryTvupfaO5sL01mn+UOyE= abdulrahman@AElawady-PC\n",
        "NodesIPRange": {
                "14": "10.20.2.0/24"
        },
        "NodeDeploymentID": {
                "14": 100050
        }
}
```

## Cancel

```bash
tfcmd cancel <deployment-name>
```

deployment-name is the name of the deployment specified in while deploying using tfcmd.

Example:

```console
$ tfcmd cancel kube
3:37PM INF canceling contracts for project kube
3:37PM INF kube canceled
```
