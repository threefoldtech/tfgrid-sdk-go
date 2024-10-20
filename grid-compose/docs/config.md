## Configuration File

This document describes the configuration file for the grid-compose project.

```yaml
version: '1.0.0'

networks:
  net1:
    name: 'miaminet'
    range:
      ip:
        type: ipv4
        ip: 10.20.0.0
      mask:
        type: cidr
        mask: 16/32
    wg: true

services:
  server:
    flist: 'https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist'
    resources:
      cpu: 2
      memory: 2048
      rootfs: 2048
    entrypoint: '/sbin/zinit init'
    ip_types:
      - ipv4
    environment:
      - SSH_KEY=<SSH_KEY>
    node_id: 11
    healthcheck:
      test:
      interval: '10s'
      timeout: '1m30s'
      retries: 3
    volumes:
      - webdata
      - dbdata
    network: net1
    depends_on:
      - database
  database:
    flist: 'https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist'
    resources:
      cpu: 2
      memory: 2048
      rootfs: 2048
    entrypoint: '/sbin/zinit init'
    ip_types:
      - ipv4
    environment:
      - SSH_KEY=<SSH_KEY>
    network: net1

volumes:
  webdata:
    mountpoint: '/data'
    size: 10GB
  dbdata:
    mountpoint: '/var/lib/postgresql/data'
    size: 10GB
```

The configuration file is a YAML file that contains the following sections:

- `version`: The version of the configuration file.
- `networks`: A list of networks that the services can use `optional`.
  - By default, the tool will create a network that will use to deploy each service.
- `services`: A list of services to deploy.
- `volumes`: A list of volumes that the services can use `optional`.

### Networks

The `networks` section defines the networks that the services can use. Each network has the following properties:

- `name`: The name of the network.
- `range`: The IP range of the network.
  - `ip`: The IP address of the network.
    - `type`: The type of the IP address.
    - `ip`: The IP address.
  - `mask`: The subnet mask of the network.
    - `type`: The type of the subnet mask.
    - `mask`: The subnet mask.
- `wg`: A boolean value that indicates whether to add WireGuard access to the network.

### Services

The `services` section defines the services to deploy. Each service has the following properties:

- `flist`: The URL of the flist to deploy.
- `resources`: The resources required by the service (CPU, memory, and rootfs) `optional`.
  - By default, the tool will use the minimum resources required to deploy the service.
    - `cpu`: 1
    - `memory`: 256MB
    - `rootfs`: 2GB
- `entrypoint`: The entrypoint command to run when the service starts.
- `ip_types`: The types of IP addresses to assign to the service `optional`.
  - ip type can be ipv4, ipv6, mycelium, yggdrasil.
- `environment`: The environment variables to set in the virtual machine.
- `node_id`: The ID of the node to deploy the service on `optional`.
  - By default, the tool will filter the nodes based on the resources required by the service.
- `healthcheck`: The healthcheck configuration for the service `optional`.
  - `test`: The command/script to run to test if the service is deployed as expected.
  - `interval`: The interval between health checks.
  - `timeout`: The timeout for the health check(includes the time the vm takes until it is up and ready to be connected to).
  - `retries`: The number of retries for the health check.
- `volumes`: The volumes to mount in the service `optional`.
- `network`: The network to deploy the service on `optional`.
  - By default, the tool will use the general network created automatically.
- `depends_on`: The services that this service depends on `optional`.

### Volumes

The `volumes` section defines the volumes that the services can use. Each volume has the following properties:

- `mountpoint`: The mountpoint of the volume.
- `size`: The size of the volume.
