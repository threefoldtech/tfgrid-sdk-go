# ThreeFold Grid Agent Framework

A standalone, reusable AI agent framework for building intelligent conversational assistants. This framework provides the core infrastructure for the ThreeFold Grid Agent, enabling natural language interactions with CLI tools and external services through Large Language Models (LLMs).

## Overview

The Agent Framework is designed to be modular, extensible, and provider-agnostic. It abstracts away the complexity of LLM interactions, tool execution, and workflow management, allowing developers to focus on building powerful AI-driven applications.

## Features

- 🤖 **LLM Provider Abstraction** - Support for multiple LLM providers (Google Gemini, OpenAI, etc.)
- 🔧 **Extensible Tool System** - Generic tool interface for integrating any CLI tool or external service
- 📡 **Real-time Streaming** - Stream command output and responses in real-time
- 💬 **Conversation Management** - Built-in session handling and conversation history
- 🔄 **Automatic Retry Logic** - Robust error handling with configurable retry mechanisms
- 📝 **Structured Responses** - JSON-based response parsing for reliable tool execution
- 🎯 **Workflow Processing** - Orchestrate complex multi-step workflows with tool calls

## Architecture

```
agent/
├── pkg/
│   ├── core/          # Agent orchestration and lifecycle management
│   ├── llm/           # LLM provider abstraction and implementations
│   ├── tools/         # Tool system and built-in tools
│   ├── workflow/      # Response processing and workflow orchestration
│   └── config/        # Configuration management
└── go.mod
```

### Package Overview

#### `pkg/core`
Core agent orchestration, managing the lifecycle of the agent and coordinating between LLM providers and tools.

**Key Components:**
- `Agent` - Main agent struct that orchestrates LLM and tools
- `Config` - Agent configuration
- Tool registration and management

#### `pkg/llm`
LLM provider abstraction layer with implementations for various providers.

**Key Components:**
- `Provider` interface - Generic LLM provider interface
- `GeminiProvider` - Google Gemini implementation
- `Config` - LLM configuration (model, prompts, retries)
- `Response` - Structured response format with tool calls

#### `pkg/tools`
Tool system for executing commands and interacting with external services.

**Key Components:**
- `Tool` interface - Generic tool interface
- `Registry` - Tool registration and lookup
- Built-in tools:
  - `CommandTool` - Execute shell commands with streaming support
  - `URLTool` - Fetch content from URLs

#### `pkg/workflow`
Workflow processing and response handling.

**Key Components:**
- `Processor` - Orchestrates tool execution based on LLM responses
- `ResponseHandler` - Interface for handling workflow events (commands, answers, errors)

## Installation

```bash
go get github.com/threefoldtech/tfgrid-sdk-go/agent
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/core"
    "github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/llm"
    "github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/tools/builtin"
)

func main() {
    // Create LLM provider
    provider, err := llm.NewGeminiProvider("your-api-key", "gemini-2.5-flash")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create agent
    agent := core.NewAgent(core.Config{
        LLMProvider: provider,
    })
    defer agent.Close()
    
    // Register built-in tools
    agent.RegisterTool(builtin.NewCommandTool())
    agent.RegisterTool(builtin.NewURLTool())
    
    // Send message
    ctx := context.Background()
    response, err := agent.SendMessage(ctx, "List files in the current directory")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Response: %s", response.Text)
}
```

### Advanced Configuration

```go
// Create provider with custom configuration
config := llm.Config{
    ModelName:        "gemini-2.5-flash",
    ResponseMIMEType: "application/json",
    SystemPrompt:     "You are a helpful assistant for managing infrastructure.",
    MaxRetries:       3,
    MaxJSONRetries:   2,
}

provider, err := llm.NewGeminiProviderWithConfig("your-api-key", config)
if err != nil {
    log.Fatal(err)
}

// Create agent with custom tool registry
registry := tools.NewRegistry()
registry.Register(builtin.NewCommandTool())
registry.Register(builtin.NewURLTool())

agent := core.NewAgent(core.Config{
    LLMProvider: provider,
    Tools:       registry,
})
```

### Streaming Command Output

```go
// Create a streaming callback for real-time output
streamCallback := func(requestID, commandID, line string) {
    fmt.Printf("[%s] %s\n", commandID, line)
}

// Register command tool with streaming
agent.RegisterTool(builtin.NewCommandToolWithStreaming(streamCallback))
```

### Using the Workflow Processor

```go
import (
    "github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/workflow"
)

// Create a response handler
type MyHandler struct{}

func (h *MyHandler) OnCommand(commandID, explanation, command string, isStreaming bool) {
    fmt.Printf("Executing: %s\n", command)
}

func (h *MyHandler) UpdateCommandOutput(commandID, command, output string, err error) {
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Output: %s\n", output)
    }
}

func (h *MyHandler) OnAnswer(answer string) error {
    fmt.Printf("Answer: %s\n", answer)
    return nil
}

func (h *MyHandler) OnQuestion(question string) error {
    fmt.Printf("Question: %s\n", question)
    return nil
}

func (h *MyHandler) OnError(message string) {
    fmt.Printf("Error: %s\n", message)
}

func (h *MyHandler) OnExplanation(text string) {
    fmt.Printf("Explanation: %s\n", text)
}

func (h *MyHandler) OnFetchURL(reason, url string) {
    fmt.Printf("Fetching URL: %s\n", url)
}

func (h *MyHandler) OnAnalyzing() {
    fmt.Println("Analyzing...")
}

// Use the processor
handler := &MyHandler{}
processor := workflow.NewProcessor(agent, handler, "request-123")

ctx := context.Background()
err := processor.ProcessMessage(ctx, "Deploy a virtual machine on the grid")
if err != nil {
    log.Fatal(err)
}
```

## Creating Custom Tools

Implement the `Tool` interface to create custom tools:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/tools"
)

type MyCustomTool struct{}

func (t *MyCustomTool) Name() string {
    return "my_custom_tool"
}

func (t *MyCustomTool) Description() string {
    return "A custom tool that does something useful"
}

func (t *MyCustomTool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
    // Extract arguments
    input, ok := args["input"].(string)
    if !ok {
        return nil, fmt.Errorf("missing 'input' argument")
    }
    
    // Do something with the input
    result := processInput(input)
    
    // Return results
    return map[string]any{
        "output": result,
    }, nil
}

// Register the tool
agent.RegisterTool(&MyCustomTool{})
```

## LLM Response Format

The framework expects LLM responses in structured JSON format. The LLM should return **one of the following response types** based on the action it needs to take:

### Execute a Command

```json
{
  "command": ["ls", "-la"],
  "explanation": "Listing all files in the current directory"
}
```

### Ask a Question

```json
{
  "question": "Which directory would you like to explore?",
  "explanation": "I need this information to proceed"
}
```

### Provide an Answer

```json
{
  "answer": "Here are the files in your directory: file1.txt, file2.go, README.md",
  "explanation": "Showing directory contents"
}
```

### Fetch External Data

```json
{
  "fetch_url": "https://example.com/api/data",
  "reason": "I will fetch the latest deployment information",
  "explanation": "Retrieving data from the API"
}
```

### Multiple Actions (Array)

For multi-step workflows, the LLM can return an array of responses:

```json
[
  {
    "command": ["mkdir", "test"],
    "explanation": "Creating a test directory"
  },
  {
    "command": ["cd", "test"],
    "explanation": "Navigating to the test directory"
  },
  {
    "answer": "Directory created and ready to use"
  }
]
```

## Configuration

### LLM Config

```go
type Config struct {
    ModelName        string // LLM model name (e.g., "gemini-2.5-flash")
    ResponseMIMEType string // Response format (e.g., "application/json")
    SystemPrompt     string // System prompt for the LLM
    MaxRetries       int    // Maximum retry attempts for API calls
    MaxJSONRetries   int    // Maximum retry attempts for JSON parsing
}
```

### Agent Config

```go
type Config struct {
    LLMProvider Provider       // LLM provider instance
    Tools       *tools.Registry // Tool registry (optional, auto-created if nil)
}
```

## Error Handling

The framework includes robust error handling:

- **Automatic Retries** - Configurable retry logic for LLM API calls
- **JSON Parsing Retries** - Retry mechanism for malformed JSON responses
- **Session Recovery** - Automatic session restart on panic with history preservation
- **Graceful Degradation** - Falls back to raw text when JSON parsing fails

## Best Practices

1. **Always Close Resources** - Call `agent.Close()` when done to release resources
2. **Use Context** - Pass context for cancellation and timeout support
3. **Handle Errors** - Check errors from all agent operations
4. **Configure Retries** - Set appropriate retry limits based on your use case
5. **Stream Long-Running Commands** - Use streaming callbacks for better UX
6. **Validate Tool Arguments** - Always validate arguments in custom tools

## Examples

See the following implementations for real-world usage:

- **CLI Application**: `../grid-cli/` - Command-line interface using the agent
- **GUI Application**: `../grid-agent-gui/` - Desktop GUI built with Wails
- **Custom Tools**: `../grid-agent-gui/internal/tfcmd/` - ThreeFold Grid command integration

## Testing

```bash
# Run tests
cd agent
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

## Contributing

Contributions are welcome! When contributing:

1. Follow Go best practices and conventions
2. Add tests for new features
3. Update documentation for API changes
4. Ensure backward compatibility when possible
5. Use meaningful commit messages

## Roadmap

- [ ] Support for additional LLM providers (OpenAI, Anthropic, etc.)
- [ ] Enhanced tool validation and schema generation
- [ ] Built-in caching for LLM responses
- [ ] Metrics and observability
- [ ] Multi-turn conversation optimization
- [ ] Tool composition and chaining

## License

This project is part of the [tfgrid-sdk-go](https://github.com/threefoldtech/tfgrid-sdk-go) repository.

## Support

For issues and questions:
- GitHub Issues: https://github.com/threefoldtech/tfgrid-sdk-go/issues
- ThreeFold Forum: https://forum.threefold.io/

## Related Projects

- [tfgrid-sdk-go](https://github.com/threefoldtech/tfgrid-sdk-go) - ThreeFold Grid SDK for Go
- [grid-cli](../grid-cli/) - ThreeFold Grid CLI tool
- [grid-agent-gui](../grid-agent-gui/) - Desktop GUI for the Grid Agent
