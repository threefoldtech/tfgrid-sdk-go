# Introduction

This is a `GO` example for the `RMB` [rpc client](https://github.com/threefoldtech/tfgrid-sdk-go/blob/development/rmb-sdk-go/direct/rpc.go) that can send `RMB` messages through working with rmb-peer and a redis server.

## How it works

To use the example, you needs to:

- Set the mnemonics variable to a valid mnemonics, with an activated account on the TFChain.
- A twinId of a remote calculator process that could add two integers

## Usage

Make sure to have a remote process that the client can call with valid twinId.

Run the client and wait for the response.
