# Connecting to a relay using an rmb peer

## Prerequirements

You need to run an `rmb-peer` locally. The `peer` instance establishes a connection to the `rmb-relay` to and works
as a gateway for all services and client behind it.\
You can find the latest releases of `rmb-peer` and `rmb-relay` [here](https://github.com/threefoldtech/rmb-rs/releases/latest)

Download the `rmb-peer` binary, add its path to your $PATH, then run:

```bash
rmb-peer -m "<mnemonics>"
```

> Can be added to the system service with systemd so it can be running all the time.\
> run `rmb-peer -h` to customize the peer, including which relay and which tfchain to connect to.

## Example

Please check the example directory for code examples

- [Server](../examples/server/main.go)
- [Client](../examples/client/main.go)
