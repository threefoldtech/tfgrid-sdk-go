package workflow

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/core"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/llm"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/tools/builtin"
)

var commandCounter uint64

const (
	requestIDKey    = "requestID"
	commandIDKey    = "commandID"
	commandToolName = "command"
)

// ResponseHandler defines the interface for handling streaming responses
type ResponseHandler interface {
	OnCommand(commandID, explanation, command string, isStreaming bool)
	OnAnalyzing()
	OnFetchURL(reason, url string)
	OnAnswer(answer string) error
	OnQuestion(question string) error
	OnExplanation(text string)
	OnError(message string)
	UpdateCommandOutput(commandID, command, output string, err error)
}

// Processor handles the response processing loop
type Processor struct {
	agent     *core.Agent
	handler   ResponseHandler
	requestID string
}

// NewProcessor creates a new response processor
func NewProcessor(agent *core.Agent, handler ResponseHandler, requestID string) *Processor {
	return &Processor{
		agent:     agent,
		handler:   handler,
		requestID: requestID,
	}
}

// ProcessMessage sends a message and processes the response stream
func (p *Processor) ProcessMessage(ctx context.Context, message string) error {
	resp, err := p.agent.SendMessage(ctx, message)
	if err != nil {
		return err
	}

	return p.processResponseLoop(ctx, resp)
}

func (p *Processor) processResponseLoop(ctx context.Context, resp *llm.Response) error {
	// Loop limit to prevent infinite loops
	const maxSteps = 20 // Increased from 10 to allow for more complex workflows
	steps := 0

	for {
		if steps >= maxSteps {
			return fmt.Errorf("maximum number of steps (%d) reached. This might indicate a loop in the agent's responses", maxSteps)
		}
		steps++

		// 1. Handle Question
		if resp.Question != "" {
			if err := p.handler.OnQuestion(resp.Question); err != nil {
				return err
			}
			// If we have a question, we pause for user input.
			// Even if there are tools, we usually want to ask first.
			// But if the model asks AND calls tools, we might want to run tools?
			// For safety, let's assume Question means "Stop and Ask".
			return nil
		}

		// 2. Handle Text/Answer/Explanation
		if resp.Text != "" {
			if len(resp.ToolCalls) > 0 {
				// Intermediate explanation - treat as analysis step
				p.handler.OnExplanation(resp.Text)
			} else {
				// Final answer - accumulate
				if err := p.handler.OnAnswer(resp.Text); err != nil {
					return err
				}
			}
		}

		// 2. Handle Tool Calls
		if len(resp.ToolCalls) > 0 {
			for _, toolCall := range resp.ToolCalls {
				var commandID string

				// Notify handler
				switch toolCall.ToolName {
				case commandToolName:
					// Generate unique command ID
					commandID = fmt.Sprintf("cmd_%d", atomic.AddUint64(&commandCounter, 1))
					cmdStr := fmt.Sprintf("%v", toolCall.Arguments["command"])

					// Check if this is a streaming command tool
					tool, _ := p.agent.GetTool(toolCall.ToolName)
					isStreaming := false
					if commandTool, ok := tool.(*builtin.CommandTool); ok && commandTool.HasStreamingCallback() {
						isStreaming = true
					}

					p.handler.OnCommand(commandID, cmdStr, cmdStr, isStreaming)
				case "fetch_url":
					urlStr := fmt.Sprintf("%v", toolCall.Arguments["url"])
					p.handler.OnFetchURL("Fetching URL...", urlStr)
				}

				// Execute tool
				tool, ok := p.agent.GetTool(toolCall.ToolName)
				if !ok {
					// Tool not found - report error to LLM
					feedback := fmt.Sprintf("Error: Tool '%s' not found.", toolCall.ToolName)
					var err error
					resp, err = p.agent.SendMessage(ctx, feedback)
					if err != nil {
						return err
					}
					continue
				}

				ctxWithID := context.WithValue(ctx, requestIDKey, p.requestID)
				// Also add commandID to context for streaming callback
				if commandID != "" {
					ctxWithID = context.WithValue(ctxWithID, commandIDKey, commandID)
				}
				output, err := tool.Execute(ctxWithID, toolCall.Arguments)

				// Update GUI with actual command output
				if toolCall.ToolName == commandToolName {
					cmdStr := fmt.Sprintf("%v", toolCall.Arguments["command"])

					// Always update the handler with the final, complete output.
					// This ensures the message returned to the frontend has the full content,
					// preventing the real-time output from being overwritten by an empty step.
					var outputStr string
					if out, ok := output["output"]; ok {
						outputStr = fmt.Sprintf("%v", out)
					} else if content, ok := output["content"]; ok {
						outputStr = fmt.Sprintf("%v", content)
					} else {
						outputStr = fmt.Sprintf("Command executed. Result: %v", output)
					}
					p.handler.UpdateCommandOutput(commandID, cmdStr, outputStr, err)
				}

				// Format feedback for LLM
				var feedback string
				if err != nil {
					feedback = fmt.Sprintf("Tool '%s' failed: %v", toolCall.ToolName, err)
					p.handler.OnError(feedback)
				} else {
					// Assume output has "output" or "content" key
					if out, ok := output["output"]; ok {
						feedback = fmt.Sprintf("Tool output:\n%v", out)
					} else if content, ok := output["content"]; ok {
						feedback = fmt.Sprintf("Tool output:\n%v", content)
					} else {
						feedback = fmt.Sprintf("Tool executed successfully. Result: %v", output)
					}

					if toolCall.ToolName == "command" {
						p.handler.OnAnalyzing()
					}
				}

				// Send feedback to LLM
				resp, err = p.agent.SendMessage(ctx, feedback)
				if err != nil {
					return err
				}
			}
			// Continue loop with new response
			continue
		}

		// If no tools (and text was already handled), we are done
		return nil
	}
}
