package core

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/llm"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/tools"
)

// Agent orchestrates LLM and tools
type Agent struct {
	provider llm.Provider
	tools    *tools.Registry
}

// Config holds agent configuration
type Config struct {
	LLMProvider llm.Provider
	Tools       *tools.Registry
}

// NewAgent creates a new agent
func NewAgent(cfg Config) *Agent {
	if cfg.Tools == nil {
		cfg.Tools = tools.NewRegistry()
	}

	return &Agent{
		provider: cfg.LLMProvider,
		tools:    cfg.Tools,
	}
}

// SendMessage sends a message to the agent
func (a *Agent) SendMessage(ctx context.Context, message string) (*llm.Response, error) {
	return a.provider.SendMessage(ctx, message)
}

// Close closes the agent and releases resources
func (a *Agent) Close() error {
	return a.provider.Close()
}

// GetTool returns a tool by name
func (a *Agent) GetTool(name string) (tools.Tool, bool) {
	return a.tools.Get(name)
}

// RegisterTool registers a new tool
func (a *Agent) RegisterTool(tool tools.Tool) {
	a.tools.Register(tool)
}
