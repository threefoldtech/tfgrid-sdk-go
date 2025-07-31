<!-- TODO: descibe the three components (wrapper/auth/protocol) -->

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

# Enhanced Signature Verification System

This document describes the enhanced signature verification system for the TFGrid Messenger, which provides cryptographic authentication of messages using twin identities stored on TFChain.

## Overview

The enhanced signature verification system ensures that:
- **Client** sends payload as `(message, twin_id, signature)` where signature is created using the twin's private key stored on TFChain
- **Server** verifies messages by loading the twin's public key from TFChain and validating the cryptographic signature
- **Protocol** maintains the existing JSONRPC logic over Mycelium while adding security

## Key Components

### 1. SignedMessage Structure

```go
type SignedMessage struct {
    TwinID    uint32 `json:"twin_id"`    // Twin ID from TFChain
    Message   string `json:"message"`    // Original message content
    Signature string `json:"signature"`  // Hex-encoded Ed25519 signature
    Timestamp int64  `json:"timestamp"`  // Unix timestamp for replay protection
}
```

### 2. TwinKeyProvider Interface

```go
type TwinKeyProvider interface {
    // GetTwinPublicKey retrieves the Ed25519 public key for a given twin ID
    GetTwinPublicKey(twinID uint32) ([]byte, error)
}
```

### 3. Core Functions

#### Client-Side Functions
- `CreateSignedMessage(twinID, message, identity)` - Creates a cryptographically signed message
- `SendSecureMessage()` - Sends a signed message through the messenger

#### Server-Side Functions
- `VerifyMessageSignature(signedMsg, keyProvider)` - Verifies signature against TFChain
- `ValidateAndExtractMessage(payload, keyProvider)` - Parses and validates signed messages
- `ParseSignedMessage(payload)` - Parses JSON payload into SignedMessage struct

## Usage Examples

### Server Setup

```go
// Connect to TFChain for twin verification
manager := substrate.NewManager("ws://192.168.1.10:9944")
sub, err := manager.Substrate()
if err != nil {
    log.Fatal("Failed to connect to TFChain")
}

// Create messenger with TFChain verification
msgr, err := messenger.NewMessenger(
    messenger.WithSubstrateConnection(sub),
)

// Register JSONRPC server
server := messenger.NewJSONRPCServer(msgr)
server.RegisterHandler("calculator.add", addHandler)
server.Start(ctx)
```

### Client Usage

```go
// Create identity from mnemonic
identity, err := substrate.NewIdentityFromSr25519Phrase(mnemonic)

// Get twin ID from TFChain
twinID, err := sub.GetTwinByPubKey(identity.PublicKey())

// Send secure signed message
response, err := msgr.SendSecureMessage(
    destination,
    jsonPayload,
    messenger.RPCKey,
    twinID,
    identity,
    true, // wait for reply
    30,   // timeout
)
```

### Message Flow

1. **Client Side:**
   ```
   Original Message → Sign with Twin Private Key → Create SignedMessage → Send via Mycelium
   ```

2. **Server Side:**
   ```
   Receive Message → Parse SignedMessage → Fetch Twin Public Key from TFChain → Verify Signature → Process if Valid
   ```

## Security Features

### Cryptographic Verification
- Uses **Ed25519** signatures for strong cryptographic security
- Twin public keys are retrieved from **TFChain blockchain** ensuring authenticity
- Messages are signed with twin's private key, verified against blockchain-stored public key

### Replay Protection
- **Timestamp** field in SignedMessage provides basic replay protection
- Server can implement additional nonce-based protection if needed

### Error Handling
- Comprehensive error messages for debugging
- Graceful fallback to unsigned messaging when TFChain is unavailable
- Clear logging of verification status

## Examples

### 1. Basic Signature Example
```bash
cd examples/signature_example
go run main.go
```

### 2. Secure JSONRPC Server
```bash
cd examples/jsonrpc/server
go run main.go
```

### 3. Enhanced Secure Client
```bash
export MNEMONIC="your twelve word mnemonic phrase here"
cd examples/secure_client
go run main.go
```

### 4. JSONRPC Client with Signature Support
```bash
cd examples/jsonrpc/client
go run main.go
```

## Configuration

### Environment Variables
- `MNEMONIC` - Twin's mnemonic phrase for client authentication

### Constants
- `chainUrl` - TFChain WebSocket endpoint (default: `ws://192.168.1.10:9944`)
- `destination` - Target Mycelium public key or IP address

## Benefits

1. **Strong Authentication** - Cryptographic proof of message origin
2. **Blockchain Integration** - Leverages TFChain for decentralized key management
3. **Backward Compatibility** - Graceful fallback when verification is unavailable
4. **Clean Architecture** - Well-organized, minimal, and easy to understand code
5. **Comprehensive Logging** - Clear visibility into verification process

## Function and Variable Naming

The enhanced system uses clear, descriptive names:
- `SignedMessage` instead of `SecureSignedRequest`
- `TwinKeyProvider` instead of `TwinPublicKeyVerifier`
- `VerifyMessageSignature` instead of `VerifySecureSignature`
- `CreateSignedMessage` instead of `CreateSecureSignedRequest`
- `ValidateAndExtractMessage` instead of `ValidateAndExtractSecureMessage`

This naming convention better describes the actual functionality and makes the code more maintainable.
