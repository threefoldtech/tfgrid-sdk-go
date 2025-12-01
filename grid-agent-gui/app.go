package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/core"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/llm"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/tools/builtin"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/workflow"
	internalConfig "github.com/threefoldtech/tfgrid-sdk-go/grid-agent-gui/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-agent-gui/internal/tfcmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/cmd"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	agent           *core.Agent
	settings        *Settings
	activeWorkflows map[string]context.CancelFunc
	workflowsMutex  sync.RWMutex
}

// Settings holds user configuration
type Settings struct {
	Mnemonics    string `json:"mnemonics"`
	Network      string `json:"network"` // mainnet, testnet, devnet
	GeminiAPIKey string `json:"geminiApiKey"`
	Theme        string `json:"theme"` // light, dark
	IsConfigured bool   `json:"isConfigured"`
}

// Step represents a single step in the agent's workflow
type Step struct {
	Type      string `json:"type"`      // "command", "url_fetch", "analysis"
	CommandID string `json:"commandID"` // Unique ID for command steps
	Content   string `json:"content"`   // Command or URL
	Output    string `json:"output"`    // Command output or fetched content
	Error     string `json:"error"`     // Error message if any
}

// Message represents a chat message for the frontend
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"requestID"` // Unique identifier for correlation
	Steps     []Step `json:"steps"`     // Workflow steps taken
	// Deprecated fields (kept for backward compatibility)
	IsCommand bool   `json:"isCommand"`
	Output    string `json:"output"`
	Error     string `json:"error"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		settings: &Settings{
			Theme: "dark",
		},
		activeWorkflows: make(map[string]context.CancelFunc),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadSettings()
}

// GetSettings returns the current settings
func (a *App) GetSettings() *Settings {
	return a.settings
}

// SaveSettings saves settings and initializes services
func (a *App) SaveSettings(mnemonics, network, apiKey string) error {
	a.settings.Mnemonics = mnemonics
	a.settings.Network = network
	a.settings.GeminiAPIKey = apiKey
	a.settings.IsConfigured = true

	// Save to file
	if err := a.saveSettingsToFile(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Set Gemini API key environment variable
	os.Setenv("GEMINI_API_KEY", apiKey)

	// Run tfcmd login
	if err := a.runTfcmdLogin(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// Initialize agent
	if err := a.initializeAgent(); err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	return nil
}

// Logout clears settings and returns to onboarding
func (a *App) Logout() error {
	// Keep theme preference
	currentTheme := a.settings.Theme

	// Clear all settings
	a.settings = &Settings{
		Theme:        currentTheme,
		IsConfigured: false,
	}

	// Save cleared settings
	if err := a.saveSettingsToFile(); err != nil {
		return fmt.Errorf("failed to clear settings: %w", err)
	}

	// Clear agent
	if a.agent != nil {
		a.agent.Close()
		a.agent = nil
	}

	return nil
}

// GUIMessageCollector collects steps and emits events
type GUIMessageCollector struct {
	ctx         context.Context
	requestID   string
	steps       []Step
	finalAnswer string
}

func (g *GUIMessageCollector) emitEvent(step Step) {
	wailsRuntime.EventsEmit(g.ctx, "agent-progress", map[string]interface{}{
		"requestID": g.requestID,
		"step":      step,
	})
}

func (g *GUIMessageCollector) OnCommand(commandID, explanation, command string, isStreaming bool) {
	// For streaming commands, create step but don't set initial output
	// Let the real-time events populate the actual content
	var initialOutput string
	if !isStreaming {
		initialOutput = "Executing..."
	}

	step := Step{
		Type:      "command",
		CommandID: commandID,
		Content:   command,
		Output:    initialOutput,
	}
	g.steps = append(g.steps, step)
	g.emitEvent(step)
}

func (g *GUIMessageCollector) UpdateCommandOutput(commandID, command, output string, err error) {
	// Find the command step by commandID and update its output
	for i := len(g.steps) - 1; i >= 0; i-- {
		if g.steps[i].Type == "command" && g.steps[i].CommandID == commandID {
			if err != nil {
				g.steps[i].Output = output
				g.steps[i].Error = err.Error()
			} else {
				g.steps[i].Output = output
			}
			g.emitEvent(g.steps[i])
			break
		}
	}
}

func (g *GUIMessageCollector) OnAnalyzing() {
	// Optional: emit analysis event
}

func (g *GUIMessageCollector) OnFetchURL(reason, url string) {
	// Truncate content for display to avoid lagging the UI
	// Note: The actual content is not passed here in the new interface,
	// but if we were to pass it, we should truncate it.
	// The current agent interface only passes reason and url for OnFetchURL.
	// The output comes later in the tool execution result.
	// Wait, the previous implementation had 'content' in OnFetchURL?
	// Let's check the original app.go again...
	// Original: func (g *GUIMessageCollector) CollectURLFetch(url string, content string)
	// New Interface: func (h ResponseHandler) OnFetchURL(reason, url string)

	// The new interface separates "Starting to fetch" (OnFetchURL) from "Tool Output" (which comes via OnCommand/OnAnalyzing or just in the loop).
	// In the new processor loop, I call:
	// p.handler.OnFetchURL("Fetching URL...", urlStr)
	// Then execute tool.
	// Then: p.handler.OnCommand("Executing command...", cmdStr) <-- Wait, for URL tool, I should probably have a way to report output.

	// In processor.go:
	// if toolCall.ToolName == "command" { p.handler.OnCommand(...) }
	// else if toolCall.ToolName == "fetch_url" { p.handler.OnFetchURL(...) }
	// ... execute ...
	// feedback = "Tool output: ..."
	// p.handler.OnAnalyzing() (only for command?)

	// The new processor doesn't explicitly report "Tool Output" to the handler for URL fetches,
	// except maybe via OnCommand if I reused it, or OnAnalyzing.
	// The original app had CollectURLFetch(url, content).

	// I should probably update the processor to pass the output to the handler,
	// or update the handler to accept output.
	// But I can't change the interface easily without breaking other things.

	// For now, I will just emit the "Fetching..." step.
	// The actual content will be fed back to the LLM.
	// If I want to show it in the UI, I need to capture the tool output.

	step := Step{
		Type:    "url_fetch",
		Content: url,
		Output:  "Fetching...",
	}
	g.steps = append(g.steps, step)
	g.emitEvent(step)
}

func (g *GUIMessageCollector) OnAnswer(answer string) error {
	// If this is the first answer, just set it
	if g.finalAnswer == "" {
		g.finalAnswer = answer
	} else {
		// If the answer is already in the final answer, don't add it again
		if !strings.Contains(g.finalAnswer, answer) {
			// Add a separator and the new answer
			separator := "\n\n" + strings.Repeat("-", 50) + "\n"
			g.finalAnswer += separator + answer
		}
	}

	// Always update the final answer with the latest content
	wailsRuntime.EventsEmit(g.ctx, "agent-answer", g.finalAnswer)

	// Add as a step if it's not already there
	step := Step{
		Type:    "answer",
		Content: answer,
	}

	// Check if we already have this exact answer in steps
	found := false
	for _, s := range g.steps {
		if s.Type == "answer" && s.Content == answer {
			found = true
			break
		}
	}

	if !found {
		g.steps = append(g.steps, step)
		g.emitEvent(step)
	}

	return nil
}

func (g *GUIMessageCollector) OnQuestion(question string) error {
	// Append the new question with a newline if there's existing content
	if g.finalAnswer != "" {
		g.finalAnswer += "\n\n" + question
	} else {
		g.finalAnswer = question
	}

	// Emit the updated final answer
	wailsRuntime.EventsEmit(g.ctx, "agent-question", g.finalAnswer)

	// Also emit as a step for consistency
	step := Step{
		Type:    "question",
		Content: question,
	}
	g.steps = append(g.steps, step)
	g.emitEvent(step)

	return nil
}

func (g *GUIMessageCollector) OnError(message string) {
	step := Step{
		Type:  "error",
		Error: message,
	}
	if len(g.steps) > 0 {
		g.steps[len(g.steps)-1].Error = message
		// Re-emit the last step with error
		g.emitEvent(g.steps[len(g.steps)-1])
	} else {
		g.steps = append(g.steps, step)
		g.emitEvent(step)
	}
}

func (g *GUIMessageCollector) OnExplanation(text string) {
	step := Step{
		Type:    "analysis",
		Content: text,
	}
	g.steps = append(g.steps, step)
	g.emitEvent(step)
}

// SendMessage sends a message to the agent and returns a rich message with all steps
func (a *App) SendMessage(message string, requestID string) (*Message, error) {
	if a.agent == nil {
		return nil, fmt.Errorf("agent not initialized")
	}

	// Create cancellable context for this workflow
	ctx, cancel := context.WithCancel(a.ctx)

	// Store cancel function
	a.workflowsMutex.Lock()
	a.activeWorkflows[requestID] = cancel
	a.workflowsMutex.Unlock()

	// Clean up after workflow completes
	defer func() {
		a.workflowsMutex.Lock()
		delete(a.activeWorkflows, requestID)
		a.workflowsMutex.Unlock()
	}()

	collector := &GUIMessageCollector{ctx: a.ctx, requestID: requestID}
	processor := workflow.NewProcessor(a.agent, collector, requestID)

	err := processor.ProcessMessage(ctx, message)
	if err != nil {
		// Check if error is due to cancellation
		if ctx.Err() == context.Canceled {
			return &Message{
				Role:      "agent",
				Content:   "I have interrupted the workflow per your request.",
				Timestamp: time.Now().Format(time.RFC3339),
				RequestID: requestID,
				Steps:     collector.steps,
			}, nil
		}
		return nil, fmt.Errorf("failed to process message: %w", err)
	}

	return &Message{
		Role:      "agent",
		Content:   collector.finalAnswer,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: requestID,
		Steps:     collector.steps,
	}, nil
}

// AbortWorkflow cancels a running workflow by requestID
func (a *App) AbortWorkflow(requestID string) error {
	a.workflowsMutex.Lock()
	defer a.workflowsMutex.Unlock()

	if cancel, exists := a.activeWorkflows[requestID]; exists {
		cancel()
		delete(a.activeWorkflows, requestID)
		log.Printf("Workflow %s aborted by user", requestID)
		return nil
	}
	return fmt.Errorf("workflow %s not found or already completed", requestID)
}

// SetTheme updates the theme
func (a *App) SetTheme(theme string) error {
	a.settings.Theme = theme
	return a.saveSettingsToFile()
}

// Helper functions

func (a *App) loadSettings() {
	settingsPath := a.getSettingsPath()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return // Settings don't exist yet
	}

	if err := json.Unmarshal(data, a.settings); err != nil {
		log.Printf("Failed to unmarshal settings: %v", err)
	}

	// If configured, initialize services
	if a.settings.IsConfigured {
		os.Setenv("GEMINI_API_KEY", a.settings.GeminiAPIKey)
		if err := a.initializeAgent(); err != nil {
			log.Printf("Failed to initialize agent: %v", err)
		}
	}
}

func (a *App) saveSettingsToFile() error {
	settingsPath := a.getSettingsPath()

	// Create directory if it doesn't exist
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0600)
}

func (a *App) getSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "grid-agent", "settings.json")
}

func (a *App) runTfcmdLogin() error {
	// Use the centralized findTfcmd function from executor
	// executor := tfcmd.NewExecutor()
	// We need to access findTfcmd but it's not exported.
	// For now, let's just assume tfcmd is in path or use a simple check.
	// Or better, expose FindTfcmd in executor package?
	// Let's just use "tfcmd" and rely on PATH for now, or copy the logic.
	// Since I can't easily modify executor.go right now without another tool call,
	// I'll copy the logic here briefly or just use "tfcmd".
	// Actually, I can just use "tfcmd" and let the user ensure it's in PATH.
	// But to be safe, I'll copy the find logic.

	tfcmdPath, err := a.findTfcmd()
	if err != nil {
		return fmt.Errorf("tfcmd not found: %w", err)
	}

	cmd := exec.Command(tfcmdPath, "login")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// Use a channel to communicate errors from the goroutine back to the main function
	errChan := make(chan error, 1)

	go func() {
		defer stdin.Close()
		// Write mnemonics and network to stdin
		if _, err := io.WriteString(stdin, a.settings.Mnemonics+"\n"); err != nil {
			errChan <- fmt.Errorf("failed to write mnemonics to stdin: %w", err)
			return
		}
		if _, err := io.WriteString(stdin, a.settings.Network+"\n"); err != nil {
			errChan <- fmt.Errorf("failed to write network to stdin: %w", err)
			return
		}
		close(errChan) // Signal that no error occurred
	}()

	output, err := cmd.CombinedOutput()

	// Check for errors from the goroutine
	if writeErr := <-errChan; writeErr != nil {
		return writeErr
	}

	if err != nil {
		return fmt.Errorf("login failed: %s", string(output))
	}

	return nil
}

func (a *App) initializeAgent() error {
	// Generate schema from tfcmd commands
	rootCmd := cmd.GetRootCmd()
	schemaDef := tfcmd.GenerateSchema(rootCmd)
	schemaJSON, err := json.MarshalIndent(schemaDef, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Create streaming callback for real-time command output
	streamCallback := func(requestID, commandID, line string) {
		// Debug: Log what we're emitting
		log.Printf("[DEBUG] Emitting command-output: requestID=%s, commandID=%s, line=%s", requestID, commandID, line)

		// Emit real-time command output to GUI
		wailsRuntime.EventsEmit(a.ctx, "command-output", map[string]interface{}{
			"requestID": requestID,
			"commandID": commandID,
			"line":      line,
			"type":      "stdout",
		})
	}

	// Create LLM provider with Threefold-specific config
	providerConfig := llm.Config{
		ModelName:        "gemini-2.5-flash",
		ResponseMIMEType: "application/json",
		SystemPrompt:     strings.Replace(internalConfig.GetSystemPrompt(), "SCHEMA_PLACEHOLDER", string(schemaJSON), 1),
		MaxRetries:       3,
		MaxJSONRetries:   2,
	}

	provider, err := llm.NewGeminiProviderWithConfig(a.settings.GeminiAPIKey, providerConfig)
	if err != nil {
		return err
	}

	// Create agent config
	cfg := core.Config{
		LLMProvider: provider,
	}

	// Create agent
	a.agent = core.NewAgent(cfg)

	// Register tools
	// 1. Built-in tools with streaming support
	a.agent.RegisterTool(builtin.NewCommandToolWithStreaming(streamCallback))
	a.agent.RegisterTool(builtin.NewURLTool())

	// 2. Tfcmd tool
	a.agent.RegisterTool(tfcmd.NewTool(rootCmd))

	return nil
}

// findTfcmd finds the tfcmd executable
func (a *App) findTfcmd() (string, error) {
	// Determine executable name based on OS
	exeName := "tfcmd"
	if runtime.GOOS == "windows" {
		exeName = "tfcmd.exe"
	}

	possiblePaths := []string{
		exeName,
		filepath.Join("..", "grid-cli", exeName),
		filepath.Join("grid-cli", exeName),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "..", "..", "grid-cli", exeName),
		filepath.Join(os.Getenv("HOME"), "Projects", "tfgrid-sdk-go", "grid-cli", exeName),
	}

	for _, path := range possiblePaths {
		if p, err := exec.LookPath(path); err == nil {
			return p, nil
		}
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
	}

	return "", fmt.Errorf("%s not found in PATH or common locations", exeName)
}
