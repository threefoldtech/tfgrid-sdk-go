package builtin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StreamCallback is called for each line of output during command execution
type StreamCallback func(requestID, commandID, line string)

type contextKey string

const (
	// RequestIDKey is the context key for the request ID
	RequestIDKey contextKey = "requestID"
	// CommandIDKey is the context key for the command ID
	CommandIDKey contextKey = "commandID"
)

// CommandTool executes shell commands with optional real-time streaming
type CommandTool struct {
	streamCallback StreamCallback
}

func NewCommandTool() *CommandTool {
	return &CommandTool{}
}

func NewCommandToolWithStreaming(callback StreamCallback) *CommandTool {
	return &CommandTool{
		streamCallback: callback,
	}
}

func (t *CommandTool) HasStreamingCallback() bool {
	return t.streamCallback != nil
}

func (t *CommandTool) Name() string {
	return "command"
}

func (t *CommandTool) Description() string {
	return "Execute a shell command"
}

func (t *CommandTool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
	requestID, _ := ctx.Value(RequestIDKey).(string)
	commandID, _ := ctx.Value(CommandIDKey).(string)
	cmdStr, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'command' argument")
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// Expand tilde and globs in arguments
	for i, arg := range parts {
		parts[i] = t.expandTilde(arg)
	}
	parts = t.expandGlob(parts)

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	var output string
	var err error

	if t.streamCallback != nil {
		output, err = t.executeWithStreaming(requestID, commandID, cmd)
	} else {
		var outputBytes []byte
		outputBytes, err = cmd.CombinedOutput()
		output = string(outputBytes)
	}

	result := map[string]any{
		"output": output,
	}
	if err != nil {
		result["error"] = err.Error()
	}

	return result, nil
}

// executeWithStreaming executes a command and streams output line by line
func (t *CommandTool) executeWithStreaming(requestID, commandID string, cmd *exec.Cmd) (string, error) {
	// Create pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Use a channel to receive output lines
	outputChan := make(chan string)
	doneChan := make(chan bool)

	// Helper to read from pipe to channel
	readPipe := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
		doneChan <- true
	}

	go readPipe(stdout)
	go readPipe(stderr)

	// Close channel when both readers are done
	go func() {
		<-doneChan
		<-doneChan
		close(outputChan)
	}()

	var fullOutput strings.Builder

	// Stream each line as it comes
	for line := range outputChan {
		fullOutput.WriteString(line + "\n")

		// Send line to callback for real-time streaming
		if t.streamCallback != nil {
			t.streamCallback(requestID, commandID, line)
		}
	}

	// Wait for command to finish
	err = cmd.Wait()
	return fullOutput.String(), err
}

// expandTilde expands the tilde in a path
func (t *CommandTool) expandTilde(path string) string {
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

// expandGlob expands glob patterns in arguments
func (t *CommandTool) expandGlob(args []string) []string {
	var expandedArgs []string
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err == nil && len(matches) > 0 {
			expandedArgs = append(expandedArgs, matches...)
		} else {
			expandedArgs = append(expandedArgs, arg)
		}
	}
	return expandedArgs
}
