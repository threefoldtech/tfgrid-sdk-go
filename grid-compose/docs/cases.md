These are most if not all the cases supported by the grid compose tool when deploying one or more services on the grid.

## Single Service

### Case 1 - Node ID Not Given + No Assigned Network

This is probably the simplest case there is.

- Filter the nodes based on the resources given to the service and choose a random one.
- Generate a default network and assign it to the deployment.

Refer to example [single_service_1.yml](/examples/single-service/single_service_1.yml)

### Case 2 - Node ID Given + No Assigned Network

- Simply use the the node id given to deploy the service.
  - Return an error if node is not available
- Generate a default network and assign it to the deployment.

Refer to example [single_service_2.yml](/examples/single-service/single_service_2.yml)

### Case 3 - Assigned Network

- Either use the assigned node id or filter the nodes for an available node if no node id given.
- Use the network assigned to the service when deploying.

Refer to example [single_service_3.yml](/examples/single-service/single_service_3.yml)

## Multiple Services

Dealing with multiple services will depend on the networks assigned to each service. In a nutshell, it is assumed that **if two services are assigned the same network they are going to be in the same deployment, which means in the same node,** so failing to stick to this assumption will yield errors and no service will be deployed.

### Same Network/No Network

This is a common general case: Laying down services a user needs to deploy in the same node using a defined network or not defining any network at all.

#### Case 1 - Node ID Given

Essentially what is required is that at least one service is assigned a node id and automatically all the other services assigned to the same network will be deployed using this node.

It is also possible to assign a node id to some of the services or even all of them, but keep in mind that **if two services running on the same network and each one is assigned a different node id, this will cause an error and nothing will be deployed.**

#### Case 2 - No Node ID Given

This is a more common case, the user mostly probably will not care to provide any node ids. In that case:

- The node id will be filtered based on the total resources for the services provided.

<br />
If all the services are assigned a network, then all of them will be deployed using that network.

If no networks are defined, then all the services will use the **default generated network**.

Refer to examples

- [two_services_same_network_1.yml](/examples/multiple-services/two_services_same_network_1.yml)
- [two_services_same_network_2.yml](/examples/multiple-services/two_services_same_network_2.yml)
- [two_services_same_network_3.yml](/examples/multiple-services/two_services_same_network_3.yml)

### Different Networks

Simple divide the services into groups having the same network(given or generated) and deal with each group using the approached described in the previous [section](#same-networkno-network).

Refer to examples

- [multiple_services_diff_network_1.yml](/examples/multiple-services/multiple_services_diff_network_1.yml)
- [multiple_services_diff_network_2.yml](/examples/multiple-services/multiple_services_diff_network_2.yml)
- [multiple_services_diff_network_3.yml](/examples/multiple-services/multiple_services_diff_network_3.yml)

## Dependencies

The tool supports deploying services that depend on each other. You can define dependencies in the yaml file by using the `depends_on` key, just like in docker-compose.

Refer to examples:

- deploying services that depend on each other on different networks:
  - [diff_networks.yml](/examples/dependency/diff_networks.yml)
- deploying services that depend on each other on the same network:
  - [same_network.yml](/examples/dependency/same_network.yml)
- a service that would depend on multiple services:
  - [multiple_dependencies.yml](/examples/dependency/multiple_dependencies.yml)
