# Node Indexers Manager

Initially the node periodically reports its data to the chain, data like capacity, uptime, location, ...etc and then the chain events is processed by `graphql-processor` to dump these data among with others data for farms/contracts/twins to a postgres database which we use to serve both `graphql-api` and `proxy-api`.
Things looks fine, but when it comes to a bigger data like gpu/dmi it is not the best solution to store these data on the chain.
And that what the `Node-Indexers` solves by periodically calling the nodes based on a configurable interval to get the data and store it on the same postgres database and then it can be served to apis. only `proxy-api` for now.

## The manager

The manager is a service started from the `cmds/main.go` and it has multiple indexer each looking for a kind of data on the nodes and it is configured by command line flags.

## The indexer structure

Each indexer has
two clients:

- `Database`: a client to the postgres db.
- `RmbClient`: an rmb client used to make the node calls.

three channels:

- `NodeTwinIdsChan`: it collects the twin ids for the nodes the indexer will call.
- `ResultChan`: it collects the results returned by the rmb call to the node.
- `BatchChan`: transfer batches of results ready to directly upserted.

four types of workers:

- `Finder`: this worker calls the database to filter nodes and push its data to the `NodeTwinIdsChan`
- `Caller`: this worker pop the twins from `NodeTwinIdsChan` and call the node with the `RmbClient` to get data and then push the result to `ResultChan`
- `Batcher`: this worker collect results from `ResultChan` in batches and send it to the `BatchChan`
- `Upserter`: this worker get data from `BatchChan` then update/insert to the `Database`

Each indexer could have some extra feature based on the use case, but these are essential.

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
