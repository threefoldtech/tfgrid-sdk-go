package tfcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Executor handles command execution
type Executor struct{}

// NewExecutor creates a new executor
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute executes a command and returns its output
func (e *Executor) Execute(command []string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("empty command")
	}

	var cmd *exec.Cmd
	if command[0] == "tfcmd" {
		// Find tfcmd executable
		tfcmdPath, err := e.findTfcmd()
		if err != nil {
			return "", err
		}
		cmd = exec.Command(tfcmdPath, command[1:]...)
	} else {
		cmd = exec.Command(command[0], command[1:]...)
	}

	// Expand tilde and globs in arguments
	for i, arg := range cmd.Args {
		cmd.Args[i] = e.expandTilde(arg)
	}
	cmd.Args = e.expandGlob(cmd.Args)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// expandTilde expands the tilde in a path
func (e *Executor) expandTilde(path string) string {
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
func (e *Executor) expandGlob(args []string) []string {
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

// findTfcmd finds the tfcmd executable
func (e *Executor) findTfcmd() (string, error) {
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
