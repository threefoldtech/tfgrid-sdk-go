# Peer

This package implements the full peer logic in go. It means you don't need the intermediate rmb-relay

The peer once created establishes and retain the connection to your relay. Any message is received by
this peer is forwarded (after passing all validation and authorization) to a custom handler

The RpcClient on the other hand is a thin wrapper around the Peer, that allows you to make Rpc calls
directly. It does this by building a special handler that routes the received responses directly to the caller
but this is completely abstract to the caller.

## Functionality

The peer implements the full rmb protocol include:

- Connecting/Authenticating to relay
- Building an RMB envelope
  - The envelope type is generated from the types.proto file which is a copy
    from the one defined by RMB.
- Sign the envelope
- Send the messages to the relay.
- Received and verify received envelopes

## Types generation

```bash
protoc -I. --go_out=types types.proto
```

## Examples

Please check the [examples](examples/) directory

## How it Works

### Peer initialization

```go
ctx, cancel := context.WithCancel(context.Background())

p, err := peer.NewPeer(
    ctx,
    mnemonics,
    subManager,
    relayCallback,
    peer.WithRelay("wss://relay.dev.grid.tf"),
    peer.WithSession("test-client"),
)
if err != nil {
    // handle error
}

// ... use p ...

// When done, initiate shutdown then wait for clean exit:
cancel()
p.Wait()
```

1- After creating a peer like this at first it will try to get the identity from the provided `mnemonics`
2- It will create a twinDB cache to keep track of twins instead of issuing a request each time to get it
3- It will update pubkey/relayurl if it doesn't match the one on substrate
4- Then it will create a Peer out of all the data provided and start it e.g calling `process()` function of that peer

### Handling incoming requests

- As mentioned above the `process()` method will be called which is a long running method which listen for incoming messages and handle them
- it listen for incoming messages and check if it is a valid envelope and then handle it
- handling incoming messages is done in `handleIncoming()` method of the peer which basically do:
  1- signature verification
  2- decipher message
  3- set the envelope payload to the decrypted message
  4- execute the callback

### Sending requests

- This done in `sendRequest()`
- This method used to send requests to a remote entity
- Do data encoding for the data
- Make an envelope out of it and send it to the relay

### Sending Responses

- This is done using `sendResponse()` method
- To reply for requests you will need the following
  1- Your peer needs to create `Router`

```go
 router := peer.NewRouter()
```

2- Then you need to create a Route for example if you are providing a calculator service

```go
app := router.SubRoute("calculator")
```

3- Then you need to register your handlers for this `subRoute` like the following

```go
app.WithHandler("sub", func(ctx context.Context, payload []byte) (interface{}, error) {
  var numbers []float64

  if err := json.Unmarshal(payload, &numbers); err != nil {
   return nil, fmt.Errorf("failed to load request payload was expecting list of float: %w", err)
  }

  var result float64
  for _, v := range numbers {
   result -= v
  }

  return result, nil
 })
```

### Handler expectations and recommended patterns

- __Handler callback semantics__
  - The `Peer` calls the configured handler synchronously from its processing loop: `handler(ctx, peer, env, err)`.
  - If you do heavy or blocking work inside your handler, you should offload it to a goroutine or a bounded worker pool to avoid stalling the peer loop.

- __Server-side: use `Router.Serve`__
  - `Router.Serve` is designed for servers. It immediately spawns a goroutine per incoming request and runs middlewares/handlers there, then replies via `peer.SendResponse(...)`.
  - This decouples heavy handler work from the peer loop. Register handlers with `router.SubRoute(...).WithHandler(...)`.

- __Client-side: use `RpcClient`__
  - `RpcClient` wraps a `Peer` and correlates responses to callers via `uid`. Its internal handler is fast and non-blocking.
  - Prefer `RpcClient.Call(ctx, twin, fn, data, &result)` over wiring your own response handler. Always pass a context with timeout/deadline.

- __Concurrency and ordering__
  - With `Router.Serve`, each request is processed concurrently in its own goroutine. Requests to the same or different commands run concurrently; there are no ordering guarantees across requests.
  - `RpcClient` handles responses without blocking the peer loop; each call blocks only the caller goroutine until its response arrives or `ctx` cancels.

- __Backpressure__
  - Sending a response uses `peer.SendResponse(...)` which ultimately writes to a relay connection. Under sustained load, the write path applies backpressure; only the handler goroutine sending that response will block. Other requests continue.
  - The peer employs bounded channel buffering on ingress and per-connection IO to improve burst tolerance while preserving backpressure.

### Shutdown

- Cancel the parent context you passed to `NewPeer(...)` (or `NewRpcClient(...)`) to request shutdown.
- Then call `p.Wait()` (or `rpc.Wait()`) to block until all goroutines have exited (including connection workers).

### Quick reference

- __Server__
  - Build routes with `router := peer.NewRouter()` and `router.SubRoute(...).WithHandler(...)`.

- __Client__
  - Use `rpc, _ := peer.NewRpcClient(ctx, mnemonics, subManager, opts...)` and call `rpc.Call(...)` with a timeout.
