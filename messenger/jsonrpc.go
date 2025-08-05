package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32000

	RPCKey = "rpc"
)

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      string      `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      string      `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type RPCHandlerFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)

// TODO: should be abstracted in an interface demo main functions
type JSONRPCServer struct {
	messenger *Messenger
	handlers  map[string]RPCHandlerFunc
}

// NewJSONRPCServer creates a new JSON-RPC server with the given client
func NewJSONRPCServer(messenger *Messenger) *JSONRPCServer {
	return &JSONRPCServer{
		messenger: messenger,
		handlers:  make(map[string]RPCHandlerFunc),
	}
}

func (s *JSONRPCServer) RegisterHandler(method string, handler RPCHandlerFunc) {
	s.handlers[method] = handler
}

func (s *JSONRPCServer) Start(ctx context.Context) error {
	s.messenger.RegisterHandler(RPCKey, s.handleRPCMessage)
	return s.messenger.StartReceiver(ctx)
}

func (s *JSONRPCServer) Stop() {
	s.messenger.StopReceiver()
}

// TODO: can this function be cleaner?
func (s *JSONRPCServer) handleRPCMessage(ctx context.Context, message *Message) ([]byte, error) {
	var request JSONRPCRequest
	if err := json.Unmarshal([]byte(message.Payload), &request); err != nil {
		response := JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    ErrCodeParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
			ID: message.ID,
		}

		responseBytes, _ := json.Marshal(response)
		return responseBytes, nil
	}

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	if request.JSONRPC != "2.0" {
		response.Error = &RPCError{
			Code:    ErrCodeInvalidRequest,
			Message: "Invalid Request",
			Data:    "jsonrpc version must be 2.0",
		}
		responseBytes, _ := json.Marshal(response)
		return responseBytes, nil
	}

	handler, ok := s.handlers[request.Method]
	if !ok {
		response.Error = &RPCError{
			Code:    ErrCodeMethodNotFound,
			Message: "Method not found",
			Data:    request.Method,
		}
		responseBytes, _ := json.Marshal(response)
		return responseBytes, nil
	}

	params := json.RawMessage("{}")
	if request.Params != nil {
		var err error
		if params, err = json.Marshal(request.Params); err != nil {
			response.Error = &RPCError{
				Code:    ErrCodeInvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			}
			responseBytes, _ := json.Marshal(response)
			return responseBytes, nil
		}
	}

	result, err := handler(ctx, params)
	if err != nil {
		response.Error = &RPCError{
			Code:    ErrCodeInternalError,
			Message: err.Error(),
		}
	} else {
		response.Result = result
	}

	responseBytes, _ := json.Marshal(response)
	return responseBytes, nil
}

// TODO: do we actually need to have client/server or we should only expose on thing
// TODO: do we need to expose messenger? or should we just expose the jsonrpc client/server
type JSONRPCClient struct {
	messenger *Messenger
}

// NewJSONRPCClient creates a new JSON-RPC client with the given client
func NewJSONRPCClient(messenger *Messenger) *JSONRPCClient {
	return &JSONRPCClient{
		messenger: messenger,
	}
}

func (c *JSONRPCClient) Call(ctx context.Context, destination string, twinID uint32, identity substrate.Identity, method string, params interface{}, result interface{}) error {
	// TODO: this encode/decode should be separated
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      fmt.Sprintf("request-%d", time.Now().UnixNano()),
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// TODO: all should be signed
	msg, err := c.messenger.SendSignedMessage(destination, string(requestBytes), RPCKey, twinID, identity, true, 0)
	if err != nil {
		return fmt.Errorf("failed to send RPC request: %w", err)
	}

	if msg == nil {
		return fmt.Errorf("no response received")
	}

	var response JSONRPCResponse
	if err := json.Unmarshal([]byte(msg.Payload), &response); err != nil {
		return fmt.Errorf("failed to parse RPC response: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("RPC error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	if result != nil && response.Result != nil {
		resultBytes, err := json.Marshal(response.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}

		if err := json.Unmarshal(resultBytes, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}
