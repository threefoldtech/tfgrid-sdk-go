package workflow

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/generative-ai-go/genai"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/chat"
)

// ResponseHandler defines the interface for handling different response types
type ResponseHandler interface {
	// OnQuestion is called when the agent asks a question (should pause for user input)
	OnQuestion(question string) error
	// OnAnswer is called when the agent provides a final answer
	OnAnswer(answer string) error
	// OnFetchURL is called before fetching a URL
	OnFetchURL(reason, url string)
	// OnProcessing is called while processing fetched content
	OnProcessing()
	// OnCommand is called before executing a command
	OnCommand(explanation, command string)
	// OnAnalyzing is called while analyzing command output
	OnAnalyzing()
	// OnError is called when there's an error
	OnError(message string)
}

// MessageCollector defines the interface for collecting messages during response processing
type MessageCollector interface {
	// CollectCommand is called when a command is executed
	CollectCommand(command []string, output string, err error)
	// CollectAnswer is called when the agent provides a final answer
	CollectAnswer(answer string)
	// CollectQuestion is called when the agent asks a question
	CollectQuestion(question string)
	// CollectURLFetch is called when a URL is fetched
	CollectURLFetch(url string, content string)
}

// CommandExecutor defines the interface for executing commands
type CommandExecutor interface {
	Execute(command []string) (string, error)
}

// ResponseProcessor handles the response processing loop
type ResponseProcessor struct {
	service        *chat.Service
	handler        ResponseHandler
	executor       CommandExecutor
	maxJSONRetries int
}

// NewResponseProcessor creates a new response processor
func NewResponseProcessor(service *chat.Service, handler ResponseHandler, executor CommandExecutor) *ResponseProcessor {
	return &ResponseProcessor{
		service:        service,
		handler:        handler,
		executor:       executor,
		maxJSONRetries: service.GetConfig().MaxJSONRetries,
	}
}

// ProcessResponse processes a Gemini response, handling commands and URL fetches automatically
// Returns true if processing should continue (question encountered), false if done
func (p *ResponseProcessor) ProcessResponse(ctx context.Context, resp *genai.GenerateContentResponse) (bool, error) {
	return p.processResponseWithRetries(ctx, resp, 0)
}

// processResponseWithRetries handles the response processing with JSON retry logic
func (p *ResponseProcessor) processResponseWithRetries(ctx context.Context, resp *genai.GenerateContentResponse, jsonRetries int) (bool, error) {
	// Parse the response
	responses, err := p.service.ParseResponse(resp)
	if err != nil {
		// JSON parsing failed - try to get model to fix it
		if jsonRetries < p.maxJSONRetries {
			// Extract the raw text that failed to parse
			var rawText string
			if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
				if txt, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
					rawText = string(txt)
				}
			}

			// Ask Gemini to fix the JSON
			feedback := fmt.Sprintf("Error parsing JSON response: %v\n\nPlease ensure your response is valid JSON. The response I received was:\n%s", err, rawText)
			newResp, err := p.service.SendMessage(ctx, genai.Text(feedback))
			if err != nil {
				return false, fmt.Errorf("failed to send JSON retry feedback: %w", err)
			}

			// Retry with the new response
			return p.processResponseWithRetries(ctx, newResp, jsonRetries+1)
		}

		// Max retries reached - try to extract raw text as error
		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			if txt, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
				p.handler.OnError(string(txt))
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to parse response after %d retries: %w", jsonRetries, err)
	}

	if len(responses) == 0 {
		return false, fmt.Errorf("empty response")
	}

	// Process responses in a loop
	for _, response := range responses {
		// Question - need user input
		if response.Question != "" {
			if err := p.handler.OnQuestion(response.Question); err != nil {
				return false, err
			}
			return true, nil // Pause for user input
		}

		// Answer - final response
		if response.Answer != "" {
			if err := p.handler.OnAnswer(response.Answer); err != nil {
				return false, err
			}
			return false, nil // Done
		}

		// URL fetch - fetch and continue
		if response.FetchURL != "" {
			p.handler.OnFetchURL(response.Reason, response.FetchURL)

			body, err := fetchURL(response.FetchURL)
			if err != nil {
				return false, fmt.Errorf("failed to fetch URL: %w", err)
			}

			p.handler.OnProcessing()
			feedback := fmt.Sprintf("Fetched content from %s:\n\n%s", response.FetchURL, body)
			newResp, err := p.service.SendMessage(ctx, genai.Text(feedback))
			if err != nil {
				return false, fmt.Errorf("failed to send feedback: %w", err)
			}

			// Recursively process the new response
			return p.ProcessResponse(ctx, newResp)
		}

		// Command execution - execute and continue
		if len(response.Command) > 0 {
			p.handler.OnCommand(response.Explanation, fmt.Sprintf("%v", response.Command))

			output, err := p.executor.Execute(response.Command)
			feedback := fmt.Sprintf("Command executed.\nOutput:\n%s", output)
			if err != nil {
				feedback += fmt.Sprintf("\nError: %v", err)
			}

			p.handler.OnAnalyzing()
			newResp, err := p.service.SendMessage(ctx, genai.Text(feedback))
			if err != nil {
				return false, fmt.Errorf("failed to send command feedback: %w", err)
			}

			// Recursively process the new response
			return p.ProcessResponse(ctx, newResp)
		}

		// Default - treat explanation as answer
		if response.Explanation != "" {
			if err := p.handler.OnAnswer(response.Explanation); err != nil {
				return false, err
			}
			return false, nil
		}
	}

	return false, nil
}

// ProcessResponseWithCollector processes a response and collects all intermediate steps
// This is useful for GUIs that want to display all steps taken
func (p *ResponseProcessor) ProcessResponseWithCollector(ctx context.Context, resp *genai.GenerateContentResponse, collector MessageCollector) (bool, error) {
	return p.processResponseWithCollectorAndRetries(ctx, resp, collector, 0)
}

// processResponseWithCollectorAndRetries handles response processing with collection and JSON retry logic
func (p *ResponseProcessor) processResponseWithCollectorAndRetries(ctx context.Context, resp *genai.GenerateContentResponse, collector MessageCollector, jsonRetries int) (bool, error) {
	// Parse the response
	responses, err := p.service.ParseResponse(resp)
	if err != nil {
		// JSON parsing failed - try to get model to fix it
		if jsonRetries < p.maxJSONRetries {
			// Extract the raw text that failed to parse
			var rawText string
			if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
				if txt, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
					rawText = string(txt)
				}
			}

			// Ask Gemini to fix the JSON
			feedback := fmt.Sprintf("Error parsing JSON response: %v\n\nPlease ensure your response is valid JSON. The response I received was:\n%s", err, rawText)
			newResp, err := p.service.SendMessage(ctx, genai.Text(feedback))
			if err != nil {
				return false, fmt.Errorf("failed to send JSON retry feedback: %w", err)
			}

			// Retry with the new response
			return p.processResponseWithCollectorAndRetries(ctx, newResp, collector, jsonRetries+1)
		}

		// Max retries reached - return error
		return false, fmt.Errorf("failed to parse response after %d retries: %w", jsonRetries, err)
	}

	if len(responses) == 0 {
		return false, fmt.Errorf("empty response")
	}

	// Process responses in a loop
	for _, response := range responses {
		// Question - need user input
		if response.Question != "" {
			collector.CollectQuestion(response.Question)
			return true, nil // Pause for user input
		}

		// Answer - final response
		if response.Answer != "" {
			collector.CollectAnswer(response.Answer)
			return false, nil // Done
		}

		// URL fetch - fetch and continue
		if response.FetchURL != "" {
			body, err := fetchURL(response.FetchURL)
			if err != nil {
				return false, fmt.Errorf("failed to fetch URL: %w", err)
			}

			collector.CollectURLFetch(response.FetchURL, body)

			feedback := fmt.Sprintf("Fetched content from %s:\n\n%s", response.FetchURL, body)
			newResp, err := p.service.SendMessage(ctx, genai.Text(feedback))
			if err != nil {
				return false, fmt.Errorf("failed to send feedback: %w", err)
			}

			// Recursively process the new response
			return p.ProcessResponseWithCollector(ctx, newResp, collector)
		}

		// Command execution - execute and continue
		if len(response.Command) > 0 {
			output, err := p.executor.Execute(response.Command)
			
			collector.CollectCommand(response.Command, output, err)

			feedback := fmt.Sprintf("Command executed.\nOutput:\n%s", output)
			if err != nil {
				feedback += fmt.Sprintf("\nError: %v", err)
			}

			newResp, err := p.service.SendMessage(ctx, genai.Text(feedback))
			if err != nil {
				return false, fmt.Errorf("failed to send command feedback: %w", err)
			}

			// Recursively process the new response
			return p.ProcessResponseWithCollector(ctx, newResp, collector)
		}

		// Default - treat explanation as answer
		if response.Explanation != "" {
			collector.CollectAnswer(response.Explanation)
			return false, nil
		}
	}

	return false, nil
}

// fetchURL fetches content from a URL
func fetchURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
