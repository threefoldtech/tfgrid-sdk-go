[![Go Documentation](https://godocs.io/github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go?status.svg)](https://godocs.io/github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go)

# Introduction

This is a `GO` sdk that can be used to build both **services**, and **clients**
that can talk over the `rmb`.

[RMB](https://github.com/threefoldtech/rmb-rs) is a message bus that enable secure
and reliable `RPC` calls across the globe.

`RMB` itself does not implement an RPC protocol, but just the secure and reliable messaging
hence it's up to server and client to implement their own data format.

## How it works

If two processes needed to communicate over `RMB`, they both need to have some sort of a connection to an `rmb-relay`.\
This connection could be established using a `direct-client`, or an `rmb-peer`.

### Direct client

A process could connect to an `rmb-relay` using a direct client.\
To create a new direct client instance, a process needs to have:

- A valid mnemonics, with an activated account on the TFChain.
- The key type of these mnemonics.
- A relay URL that the direct client will connect to.
- A session id. This could be anything, but a twin must only have a unique session id per connection.
- A substrate connection.

#### **Example**

Creating a new direct client instance:

```Go
subManager := substrate.NewManager("wss://tfchain.dev.grid.tf/ws")
sub, err := subManager.Substrate()
if err != nil {
    return fmt.Errorf("failed to connect to substrate: %w", err)
}

defer sub.Close()
client, err := direct.NewRpcClient(direct.KeyTypeSr25519, mnemonics, "wss://relay.dev.grid.tf", "test-client", sub, false)
if err != nil {
    return fmt.Errorf("failed to create direct client: %w", err)
}
```

Assuming there is a remote calculator process that could add two integers, an rmb call using the direct client would look like this:

```Go
x := 1
y := 2
var sum int
err := client.Call(ctx, destinationTwinID, "calculator.add", []int{x, y}, &sum)
```
