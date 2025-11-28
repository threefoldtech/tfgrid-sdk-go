package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/chat"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/runner"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/workflow"
)

// App struct
type App struct {
	ctx         context.Context
	chatService *chat.Service
	settings    *Settings
	lastMessage *Message // Store the last message for the response handler
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
	Type    string `json:"type"`    // "command", "url_fetch", "analysis"
	Content string `json:"content"` // Command or URL
	Output  string `json:"output"`  // Command output or fetched content
	Error   string `json:"error"`   // Error message if any
}

// Message represents a chat message for the frontend
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
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

	// Initialize chat service
	if err := a.initializeChatService(); err != nil {
		return fmt.Errorf("failed to initialize chat: %w", err)
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

	// Clear chat service
	a.chatService = nil

	return nil
}

// GUIMessageCollector collects all steps during response processing
type GUIMessageCollector struct {
	steps       []Step
	finalAnswer string
}

func (g *GUIMessageCollector) CollectCommand(command []string, output string, err error) {
	step := Step{
		Type:    "command",
		Content: fmt.Sprintf("%v", command),
		Output:  output,
	}
	if err != nil {
		step.Error = err.Error()
	}
	g.steps = append(g.steps, step)
}

func (g *GUIMessageCollector) CollectAnswer(answer string) {
	g.finalAnswer = answer
}

func (g *GUIMessageCollector) CollectQuestion(question string) {
	g.finalAnswer = question
}

func (g *GUIMessageCollector) CollectURLFetch(url string, content string) {
	// Truncate content to first 500 chars for display
	displayContent := content
	if len(content) > 500 {
		displayContent = content[:500] + "\n\n... (content truncated for brevity)"
	}
	g.steps = append(g.steps, Step{
		Type:    "url_fetch",
		Content: url,
		Output:  displayContent,
	})
}

// SendMessage sends a message to the chat service and returns a rich message with all steps
func (a *App) SendMessage(message string) (*Message, error) {
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}

	// Send initial message to Gemini
	resp, err := a.chatService.SendMessage(a.ctx, genai.Text(message))
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Create collector and processor
	collector := &GUIMessageCollector{}
	processor := workflow.NewResponseProcessor(a.chatService, a, a)

	// Process response with collector
	_, err = processor.ProcessResponseWithCollector(a.ctx, resp, collector)
	if err != nil {
		return nil, fmt.Errorf("failed to process response: %w", err)
	}

	// Build rich message with all steps
	return &Message{
		Role:      "agent",
		Content:   collector.finalAnswer,
		Timestamp: time.Now().Format(time.RFC3339),
		Steps:     collector.steps,
	}, nil
}

// ResponseHandler implementation (workflow.ResponseHandler - required by ResponseProcessor, but not used with collector)
func (a *App) OnQuestion(question string) error { return nil }
func (a *App) OnAnswer(answer string) error     { return nil }
func (a *App) OnFetchURL(reason, url string)    {}
func (a *App) OnProcessing()                     {}
func (a *App) OnCommand(explanation, command string) {}
func (a *App) OnAnalyzing()                      {}
func (a *App) OnError(message string)            {}

// CommandExecutor implementation (workflow.CommandExecutor - required by ResponseProcessor)
func (a *App) Execute(command []string) (string, error) {
	return a.executeCommand(command)
}

// executeCommand executes a command and returns its output
func (a *App) executeCommand(command []string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("empty command")
	}

	var cmd *exec.Cmd
	if command[0] == "tfcmd" {
		// Find tfcmd executable
		tfcmdPath, err := a.findTfcmd()
		if err != nil {
			return "", err
		}
		cmd = exec.Command(tfcmdPath, command[1:]...)
	} else {
		cmd = exec.Command(command[0], command[1:]...)
	}

	// Expand tilde and globs in arguments (reusing CLI logic)
	for i, arg := range cmd.Args {
		cmd.Args[i] = runner.ExpandTilde(arg)
	}
	cmd.Args = runner.ExpandGlob(cmd.Args)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// expandTilde expands the tilde in a path
func (a *App) expandTilde(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "~", home, 1)
		}
	}
	return path
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

	json.Unmarshal(data, a.settings)

	// If configured, initialize services
	if a.settings.IsConfigured {
		os.Setenv("GEMINI_API_KEY", a.settings.GeminiAPIKey)
		a.initializeChatService()
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
	// Use the centralized findTfcmd function
	tfcmdPath, err := a.findTfcmd()
	if err != nil {
		return fmt.Errorf("tfcmd not found. Please ensure it is in your PATH or in the grid-cli directory: %w", err)
	}

	cmd := exec.Command(tfcmdPath, "login")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, a.settings.Mnemonics+"\n")
		io.WriteString(stdin, a.settings.Network+"\n")
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("login failed: %s", string(output))
	}

	return nil
}

func (a *App) initializeChatService() error {
	// Use the shared schema generation logic (same as CLI)
	rootCmd := cmd.GetRootCmd()
	service, err := chat.NewServiceWithSchema(a.settings.GeminiAPIKey, rootCmd)
	if err != nil {
		return err
	}

	a.chatService = service
	return nil
}
