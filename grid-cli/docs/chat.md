# Chat Command Documentation

## Overview
`tfgrid-cli` now includes an AI‑powered **chat** command that lets you interact with the Grid CLI using natural language. The command talks to the Gemini 2.5‑flash model, validates input, fetches external data when needed, and executes the appropriate CLI sub‑commands.

## Usage
```bash
# Make sure the Gemini API key is set
export GEMINI_API_KEY=your_key_here

# Start an interactive chat session
./tfcmd chat
```
The prompt will appear and you can type natural language requests such as:
- *"Deploy a Ubuntu VM named `myvm` with 2 CPU and 4 GB RAM"*
- *"What is my current contract ID for `myvm`?"*
- *"Show me the available official flists for Ubuntu"*
- *"Delete the deployment `myvm`"*

## Features
1. **Natural‑language parsing** – Gemini converts your request into a structured JSON response.
2. **Pre‑execution validation** – Required flags, flag groups, and boolean syntax are checked before any command runs.
3. **Consultative flow** – The agent asks for missing optional parameters and explains defaults.
4. **External look‑ups** – Can fetch data from:
   - `https://hub.grid.tf/api/flist/tf‑official‑vms`
   - `https://hub.grid.tf/api/flist/tf‑official‑apps`
   - GitHub `tf‑images` READMEs for env‑vars and entrypoints
   - Grid documentation (`https://manual.grid.tf/labs/documentation/`)
5. **Auto‑retry & feedback loop** – Command output is fed back to the model, which can suggest fixes or ask follow‑up questions.
6. **Safety checks** – Deleting all contracts (`tfcmd cancel contracts -a`) requires explicit confirmation with a clear affirmation.

## Example Session
```
$ ./tfcmd chat
> what can you do for me?
Agent: I can help you interact with the Threefold Grid CLI (`tfcmd`). I can assist you with:
* Deploying resources (VMs, Kubernetes, Gateways, ZDBs)
* Getting information about contracts, VMs, clusters, etc.
* Updating resources (add/delete workers)
* Cancelling resources (single or all)
* Authentication guidance
* General CLI info

> i want to deploy Kubernetes cluster
Agent: To deploy a Kubernetes cluster, I need a name for the cluster and the path to your public SSH key. Could you please provide these?
> kb02, /home/sameh/.ssh/id_rsa.pub
Agent: Great! You want to deploy a Kubernetes cluster named 'kb02' with the SSH key from '/home/sameh/.ssh/id_rsa.pub'.
By default, the master node will be deployed with:
* 1 CPU
* 1 GB memory
* 2 GB disk storage
* On farm ID 1
* Mycelium and Yggdrasil IPs (no public IPv4/IPv6)
No worker nodes are added initially.
Would you like to:
1. Customize the master node's resources?
2. Add worker nodes now?
3. Proceed with defaults?
> 4 cpu and 4 gb, and ipv4. add also one worker with same specs but no ipv4
Agent: I will deploy a Kubernetes cluster named 'kb02'. The master node will have 4 CPUs, 4GB memory, and a public IPv4. It will use the default 2GB disk, be deployed on farm ID 1, and have Mycelium and Yggdrasil IPs enabled. I will also add one worker node with 4 CPUs and 4GB memory, using the default 2GB disk, on farm ID 1, and with Mycelium and Yggdrasil IPs enabled, but no public IPv4.
Running: tfcmd deploy kubernetes --name kb02 --ssh /home/sameh/.ssh/id_rsa.pub --master-cpu 4 --master-memory 4 --ipv4 --workers-number 1 --workers-cpu 4 --workers-memory 4 --workers-ipv4=false
Agent: Your Kubernetes cluster 'kb02' has been successfully deployed! Master Node IPs: Wireguard 10.20.2.2, IPv4 185.206.122.51/24, Planetary ..., Mycelium ...
Worker0 Node IPs: Wireguard 10.20.3.2, Planetary ..., Mycelium ...
> delete it
Agent: I will delete the Kubernetes cluster named 'kb02'. This command will remove all associated contracts and resources for this specific cluster.
Running: tfcmd cancel kb02
Agent: Your Kubernetes cluster 'kb02' and all its associated resources have been successfully deleted.

> lets deploy ubuntu 24 based vm
Agent: I will lookup the official flist for Ubuntu 24.
Fetching: https://hub.grid.tf/api/flist/tf-official-vms
Agent: I see `ubuntu-24.04-full.flist` and `ubuntu-24.04-latest.flist` available for Ubuntu 24. Which one would you like to use? Also, I will need a name for your VM and the path to your public SSH key.
> latest, vm02, /home/sameh/.ssh/id_rsa.pub
Agent: Great! Details:
- Name: vm02
- Flist: https://hub.grid.tf/api/flist/tf-official-vms/ubuntu-24.04-latest.flist
- SSH Key: /home/sameh/.ssh/id_rsa.pub
- Entrypoint: /sbin/zinit init (default)
Default VM resources: 1 CPU, 1 GB memory, 2 GB rootfs, 0 GB data disk, Mycelium enabled.
Would you like to customize resources, add a volume, configure public IPv4/IPv6, or specify a farm/node?
> 4 cpu, 8 gb memory, and a disk with 10gb for data. only ipv4, no Mycelium and ygg
Agent: Deploying VM 'vm02' with 4 CPUs, 8 GB memory, 10 GB data disk, public IPv4, Mycelium disabled, Yggdrasil disabled.
Running: tfcmd deploy vm --name vm02 --flist https://hub.grid.tf/api/flist/tf-official-vms/ubuntu-24.04-latest.flist --ssh /home/sameh/.ssh/id_rsa.pub --entrypoint /sbin/zinit init --cpu 4 --memory 8 --disk 10 --ipv4=true --mycelium=false --ygg=false
Agent: Your VM 'vm02' has been successfully deployed! Public IPv4: 185.206.122.51.
> what is the contract id? and node id?
Agent: To get the contract ID and node ID, I will run:
Running: tfcmd get vm vm02
Agent: The contract ID is `250933`, Node ID is `11`.
> lets delete it please
Agent: Deleting VM 'vm02' by name.
Running: tfcmd cancel vm02
Agent: Your VM 'vm02' and its associated contracts have been successfully canceled and deleted.

> now lets deploy a presearch node
Agent: I will lookup the official flist for Presearch.
Fetching: https://hub.grid.tf/api/flist/tf-official-apps
Agent: I will lookup the required environment variables and entrypoint for the Presearch node.
Fetching: https://github.com/threefoldtech/tf-images/tree/development/tfgrid3/presearch/README.md
Agent: To deploy a Presearch node, I need your PRESEARCH_REGISTRATION_CODE, a name, and your SSH key.
> PRESEARCH_REGISTRATION_CODE is askjdhs23hedj, my ssh path is /home/sameh/.ssh/id_rsa.pub and deployment name could be pre02. can you double the presearch minimum requirement spec?
Agent: Using double minimum specs: 2 CPU, 2 GB memory, 20 GB data disk, 2 GB rootfs, Mycelium enabled, no public IPv4.
Would you like a public IPv4?
> use a public IPv4
Agent: Deploying Presearch node 'pre02' with public IPv4, 2 CPU, 2 GB memory, 20 GB data disk.
Running: tfcmd deploy vm --name pre02 --flist https://hub.grid.tf/api/flist/tf-official-apps/presearch.flist --ssh /home/sameh/.ssh/id_rsa.pub --entrypoint /sbin/zinit init --env PRESEARCH_REGISTRATION_CODE=askjdhs23hedj --cpu 2 --memory 2 --disk 20 --ipv4=true
Agent: Your Presearch node 'pre02' has been successfully deployed! Public IPv4: 185.206.122.51, Mycelium IP: 4a0:dd61:94df:12ab:ff0f:dbae:e333:d2b.
> delete it please
Agent: Deleting Presearch node 'pre02'.
Running: tfcmd cancel pre02
Agent: Your Presearch node 'pre02' and its associated contracts have been successfully canceled and deleted.
> exit
Session ended.
```
