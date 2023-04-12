# Direct client
Direct client does not use the `rmb-peer` and connects directly to the rmb relay.
This means that the client need to take care of some more things this include:
- Connecting/Authenticating to relay
- Building an RMB envelope
  - The envelope type is generated from the types.proto file which is a copy
    from the one defined by RMB.
- Sign the envelope
- Send the request to the relay.
- Received and verify received envelopes

> The direct client is still a WIP. Although it works perfectly well, it yet need to verify
received envelope signature.

## Types generation
```bash
protoc -I. --go_out=types types.proto
```
