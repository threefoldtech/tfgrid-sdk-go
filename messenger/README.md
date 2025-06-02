# Messenger Package

The Messenger package provides a Go SDK for building distributed messaging applications on top of the Mycelium network infrastructure. It offers a topic-based protocol registration system that enables developers to create custom server/client implementations with optional blockchain identity integration.

## Overview

The Messenger package serves as a high-level abstraction over the Mycelium messaging infrastructure, providing:

- **Topic-based message routing**: Register handlers for specific message topics
- **Bidirectional communication**: Send messages and receive replies
- **Optional blockchain identity**: Integrate with ThreeFold Chain for identity management
- **JSON-RPC support**: Built-in JSON-RPC server/client implementation

## Mycelium Infrastructure

The Mycelium daemon provides distinct communication methods:

1. HTTP REST server `:8989`
2. RPC server `:9090`
3. CLI `mycelium message` calling the reset server

For more info check [mycelium docs](https://github.com/threefoldtech/mycelium/tree/master/docs)


### Usage Patterns

- **Sending Messages**: Uses CLI command `mycelium message send <destination> <payload> [--topic <topic>] [--wait] [--timeout <seconds>]`
- **Receiving Messages**: Uses CLI command `mycelium message receive` in a polling loop
- **Sending Replies**: Uses HTTP POST to `/api/v1/messages/reply/{messageId}`
- **Getting Identity**: Uses HTTP GET to `/api/v1/admin`

## Messenger Core Component

The Messenger serves as the main orchestration component with the following key features:

### Topic-Based Protocol Registration

The Messenger implements a topic-based routing system where different message handlers can be registered for specific topics:

```go
// Register a handler for a specific topic
messenger.RegisterHandler("my-topic", func(ctx context.Context, message *Message) ([]byte, error) {
    // Handle the message
    return response, nil
})
```

### Message Structure

```go
type Message struct {
    ID      string `json:"id,omitempty"`      // Unique message identifier
    Topic   string `json:"topic,omitempty"`   // Message topic for routing
    SrcIP   string `json:"srcIp,omitempty"`   // Source IP address
    SrcPK   string `json:"srcPk,omitempty"`   // Source public key
    DstIP   string `json:"dstIp,omitempty"`   // Destination IP address
    DstPK   string `json:"dstPk,omitempty"`   // Destination public key
    Payload string `json:"payload,omitempty"` // Message payload (raw string)
}
```

### Core Operations

- **Send Message**: Send a message to a destination with optional reply waiting
- **Register Handler**: Register topic-specific message handlers
- **Start/Stop Receiver**: Control the message listening loop
- **Send Reply**: Reply to a received message

## Chain Identity Management

The package functions independently without chain identity integration by default. However, it offers optional blockchain identity features:

### Enabling Chain Identity

To enable chain identity management, use the `WithEnableTwinIdentity(true)` configuration option:

```go
messenger, err := messenger.NewMessenger(
    messenger.WithSubstrateManager(manager),
    messenger.WithMnemonic(mnemonic),
    messenger.WithEnableTwinIdentity(true), // Enable chain identity
)
```

### Required Configuration

When chain identity is enabled, the following are required:
- **Substrate Manager**: Connection to ThreeFold Chain
- **Identity or Mnemonic**: Either a substrate identity or mnemonic phrase

### Identity Lifecycle

1. **Startup**: Automatic identity updates on chain during messenger initialization
   - Retrieves Mycelium node information via HTTP API
   - Updates the MyceliumTwin mapping on the blockchain
   
2. **Message Processing**: Identity retrieval and context storage
   - For each incoming message, retrieves the sender's twin ID from the blockchain
   - Stores twin ID in the message context for handler use
   - Accessible via `TwinIdContextKey` context key

### MyceliumTwin Storage Mapping

The chain identity feature leverages the MyceliumTwin storage mapping on the ThreeFold blockchain, which:
- Maps Mycelium public keys to Twin IDs
- Enables identity verification and authorization
- Provides a bridge between Mycelium network and blockchain identity

## JSON-RPC Implementation

The package includes a complete JSON-RPC server/client implementation registered under the 'rpc' topic key:

### JSON-RPC Server

```go
server := messenger.NewJSONRPCServer(msgr)
server.RegisterHandler("calculator.add", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
    // Handle RPC method
    return result, nil
})
server.Start(ctx)
```

### JSON-RPC Client

```go
client := messenger.NewJSONRPCClient(msgr)
var result float64
err := client.Call(ctx, destination, "calculator.add", []float64{10, 20}, &result)
```
## Configuration

### Configuration Options

The Messenger supports various configuration options through functional options:

```go
type MessengerOpt func(*Messenger)

// Available options:
messenger.WithMnemonic(mnemonic)                    // Set mnemonic phrase
messenger.WithIdentity(identity)                    // Set substrate identity directly
messenger.WithEnableTwinIdentity(true)              // Enable chain identity management
messenger.WithSubstrateManager(manager)             // Set substrate manager
messenger.WithBinaryPath("/path/to/mycelium")       // Set custom mycelium binary path
messenger.WithAPIAddress("http://localhost:8989")   // Set custom API address
```

### Default Values

```go
const (
    DefaultMessengerBinary = "mycelium"                    // Mycelium binary path
    DefaultAPIAddress = "http://127.0.0.1:8989"           // Mycelium HTTP API address
    DefaultTimeout = 60                                    // Default timeout in seconds
    DefaultRetryListenerInterval = 100 * time.Millisecond // Retry interval for message listener
)
```