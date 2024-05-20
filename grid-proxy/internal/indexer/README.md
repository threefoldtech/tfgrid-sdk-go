# Node Indexers Manager

Initially the node periodically reports its data to the chain, data like capacity, uptime, location, ...etc and then the chain events is processed by `graphql-processor` to dump these data among with others data for farms/contracts/twins to a postgres database which we use to serve both `graphql-api` and `proxy-api`.
Things looks fine, but when it comes to a bigger data like gpu/dmi it is not the best solution to store these data on the chain.
And that what the `Node-Indexers` solves by periodically calling the nodes based on a configurable interval to get the data and store it on the same postgres database and then it can be served to apis. only `proxy-api` for now.

## The indexer structure

Each indexer has
two clients:

- `Database`: a client to the postgres db.
- `RmbClient`: an rmb client used to make the node calls.

three channels:

- `IdChan`: it collects the twin ids for the nodes the indexer will call.
- `ResultChan`: it collects the results returned by the rmb call to the node.
- `BatchChan`: transfer batches of results ready to directly upserted.

four types of workers:

- `Finder`: this worker calls the database to filter nodes and push its data to the `IdChan`
- `Getter`: this worker pop the twins from `IdChan` and call the node with the `RmbClient` to get data and then push the result to `ResultChan`
- `Batcher`: this worker collect results from `ResultChan` in batches and send it to the `BatchChan`
- `Upserter`: this worker get data from `BatchChan` then update/insert to the `Database`

The indexer struct is generic and each indexer functionality differ from the others based on its Work.
Work a struct that implement the interface `Work` which have three methods:

- `Finders`: this is a map of string and interval to decide which finders this node should use.
- `Get`: a method that prepare the payload from rmb call and parse the response to return a ready db model data.
- `Upsert`: calling the equivalent db upserting method with the ability to remove old expired data.

## Registered Indexers

1. Gpu indexer:
   - Function: query the gpu list on node.
   - Interval: `60 min`
   - Other triggers: new node is added (check every 5m).
   - Default caller worker number: 5
   - Dump table: `node_gpu`
2. Health indexer:
   - Function: decide the node health based on its internal state.
   - Interval: `5 min`
   - Default caller worker number: 100
   - Dump table: `health_report`
3. Dmi indexer:
   - Function: collect some hardware data from the node.
   - Interval: `1 day`
   - Other triggers: new node is added (check every 5m).
   - Default caller worker number: 1
   - Dump table: `dmi`
4. Speed indexer:
   - Function: get the network upload/download speed on the node tested against `iperf` server.
   - Interval: `5 min`
   - Default caller worker number: 100
   - Dump table: `speed`
5. Ipv6 indexer:
   - Function: decide if the node has ipv6 or not.
   - Interval: `1 day`
   - Default caller worker number: 10
   - Dump table: `node_ipv6`
6. Workloads indexer:
   - Function: get the number of workloads on each node.
   - Interval: `1 hour`
   - Default caller worker number: 10
   - Dump table: `node_workloads`
