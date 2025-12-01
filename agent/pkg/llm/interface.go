package llm

import "context"

// Provider defines the interface for LLM providers
type Provider interface {
	// SendMessage sends a message and returns the response
	SendMessage(ctx context.Context, message string) (*Response, error)

	// GetHistory returns the conversation history
	GetHistory() []Message

	// Close closes the provider and releases resources
	Close() error
}

// Response represents an LLM response
type Response struct {
	Text         string
	Question     string
	ToolCalls    []ToolCall
	FinishReason string
}

// ToolCall represents a request to call a tool
type ToolCall struct {
	ToolName  string
	Arguments map[string]any
}

// Message represents a conversation message
type Message struct {
	Role    string
	Content string
}

// Config holds LLM provider configuration
type Config struct {
	Provider         string // "gemini", "openai", etc.
	APIKey           string
	ModelName        string
	ResponseMIMEType string
	SystemPrompt     string
	MaxRetries       int
	MaxJSONRetries   int
}
