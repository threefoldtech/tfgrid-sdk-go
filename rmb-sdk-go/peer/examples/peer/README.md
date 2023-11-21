# Introduction

This is a `Go` example for the `RMB` [direct client](https://github.com/threefoldtech/tfgrid-sdk-go/blob/development/rmb-sdk-go/direct/README.md#direct-client) that can send `RMB` messages through connecting to the relay directly.

## How it works

To use the example, you needs to:

-   Set the mnemonics variable to a valid mnemonics, with an activated account on the TFChain.
-   Set dist to the twinId of a remote calculator process that could add two integers

## Usage

Make sure to have a remote process that the client can call with valid twinId.

Run the client and wait for the response.
