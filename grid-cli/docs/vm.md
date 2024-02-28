# Virtual Machine

This document explains Virtual Machine related commands using tfcmd.

## Deploy

```bash
tfcmd deploy vm [flags]
```

### Required Flags

- name: name for the VM deployment also used for canceling the deployment. must be unique.
- ssh: path to public ssh key to set in the VM.

### Optional Flags

- node: node id vm should be deployed on.
- farm: farm id vm should be deployed on, if set choose available node from farm that fits vm specs (default 1). note: node and farm flags cannot be set both.
- cpu: number of cpu units (default 1).
- disk: size of disk in GB mounted on /data. if not set no disk workload is made.
- entrypoint: entrypoint for VM flist (default "/sbin/zinit init"). note: setting this without the flist option will fail.
- flist: flist used in VM (default "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist"). note: setting this without the entrypoint option will fail.
- ipv4: assign public ipv4 for VM (default false).
- ipv6: assign public ipv6 for VM (default false).
- memory: memory size in GB (default 1).
- rootfs: root filesystem size in GB (default 2).
- ygg: assign yggdrasil ip for VM (default true).
- gpus: assign a list of gpus' ids to the VM. note: setting this without the node option will fail.

Example:

- Deploying VM without GPU

```console
$ tfcmd deploy vm --name examplevm --ssh ~/.ssh/id_rsa.pub --cpu 2 --memory 4 --disk 10
12:06PM INF deploying network
12:06PM INF deploying vm
12:07PM INF vm planetary ip: 300:e9c4:9048:57cf:7da2:ac99:99db:8821
```
- Deploying VM with GPU

```console
$ tfcmd deploy vm --name examplevm --ssh ~/.ssh/id_rsa.pub --cpu 2 --memory 4 --disk 10 --gpus '0000:0e:00.0/1882/543f' --gpus '0000:0e:00.0/1887/593f' --node 12
12:06PM INF deploying network
12:06PM INF deploying vm
12:07PM INF vm planetary ip: 300:e9c4:9048:57cf:7da2:ac99:99db:8821
```

## Get

```bash
tfcmd get vm <vm>
```

vm is the name used when deploying vm using tfcmd.

Example:

```console
$ tfcmd get vm examplevm
3:20PM INF vm:
{
        "Name": "examplevm",
        "NodeID": 15,
        "SolutionType": "vm/examplevm",
        "SolutionProvider": null,
        "NetworkName": "examplevmnetwork",
        "Disks": [
                {
                        "Name": "examplevmdisk",
                        "SizeGB": 10,
                        "Description": ""
                }
        ],
        "Zdbs": [],
        "Vms": [
                {
                        "Name": "examplevm",
                        "Flist": "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
                        "FlistChecksum": "",
                        "PublicIP": false,
                        "PublicIP6": false,
                        "Planetary": true,
                        "Corex": false,
                        "ComputedIP": "",
                        "ComputedIP6": "",
                        "PlanetaryIP": "301:ad3a:9c52:98d1:cd05:1595:9abb:e2f1",
                        "IP": "10.20.2.2",
                        "Description": "",
                        "CPU": 2,
                        "Memory": 4096,
                        "RootfsSize": 2048,
                        "Entrypoint": "/sbin/zinit init",
                        "Mounts": [
                                {
                                        "DiskName": "examplevmdisk",
                                        "MountPoint": "/data"
                                }
                        ],
                        "Zlogs": null,
                        "EnvVars": {
                                "SSH_KEY": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDcGrS1RT36rHAGLK3/4FMazGXjIYgWVnZ4bCvxxg8KosEEbs/DeUKT2T2LYV91jUq3yibTWwK0nc6O+K5kdShV4qsQlPmIbdur6x2zWHPeaGXqejbbACEJcQMCj8szSbG8aKwH8Nbi8BNytgzJ20Ysaaj2QpjObCZ4Ncp+89pFahzDEIJx2HjXe6njbp6eCduoA+IE2H9vgwbIDVMQz6y/TzjdQjgbMOJRTlP+CzfbDBb6Ux+ed8F184bMPwkFrpHs9MSfQVbqfIz8wuq/wjewcnb3wK9dmIot6CxV2f2xuOZHgNQmVGratK8TyBnOd5x4oZKLIh3qM9Bi7r81xCkXyxAZbWYu3gGdvo3h85zeCPGK8OEPdYWMmIAIiANE42xPmY9HslPz8PAYq6v0WwdkBlDWrG3DD3GX6qTt9lbSHEgpUP2UOnqGL4O1+g5Rm9x16HWefZWMjJsP6OV70PnMjo9MPnH+yrBkXISw4CGEEXryTvupfaO5sL01mn+UOyE= abdulrahman@AElawady-PC\n"
                        },
                        "NetworkName": "examplevmnetwork"
                }
        ],
        "QSFS": [],
        "NodeDeploymentID": {
                "15": 22748
        },
        "ContractID": 22748
}
```

## Cancel

```bash
tfcmd cancel <deployment-name>
```

deployment-name is the name of the deployment specified in while deploying using tfcmd.

Example:

```console
$ tfcmd cancel examplevm
3:37PM INF canceling contracts for project examplevm
3:37PM INF examplevm canceled
```
