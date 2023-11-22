# Introduction

This is a `Go` example for the `RMB` [peer router using direct client](https://github.com/threefoldtech/tfgrid-sdk-go/blob/development/rmb-sdk-go/peer/README.md#direct-client) that starts a server as peer using the peer router that. The peer can send `RMB` messages through connecting to the relay directly.

## How it works

To use the example, you needs to:

-   Set the mnemonics variable to a valid mnemonics for client peer and server, with an activated account on the TFChain.
-   set the client peer destination twin and session with the ones of the created peer router.
-   make sure you are running the server before the client peer.
   
## Usage

Run the client and wait for the response.
This example doesn't depend on rmb-peer.
