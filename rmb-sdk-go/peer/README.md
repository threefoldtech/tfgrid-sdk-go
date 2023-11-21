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
