package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/ui"
)

// ExecuteWithRollingOutput executes a command and returns its output
func ExecuteWithRollingOutput(cmd *exec.Cmd, ui *ui.UI) (string, error) {
	var output strings.Builder

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting command: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	processPipe := func(pipe io.Reader, printFunc func(string)) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			printFunc(line)
			mu.Lock()
			output.WriteString(line + "\n")
			mu.Unlock()
		}
	}

	wg.Add(2)
	go processPipe(stdout, ui.PrintOutput)
	go processPipe(stderr, ui.PrintError)

	// Wait for the command to finish
	cmdErr := cmd.Wait()

	// Wait for the output processing to finish
	wg.Wait()

	if cmdErr != nil {
		return output.String(), fmt.Errorf("command execution failed: %w", cmdErr)
	}

	return output.String(), nil
}

// ExpandTilde expands the tilde in a path and normalizes path separators
func ExpandTilde(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			// Replace ~ with home directory
			expanded := strings.Replace(path, "~", home, 1)
			// Normalize path separators for the OS
			return filepath.FromSlash(expanded)
		}
	}
	return path
}

// ExpandGlob expands globs in arguments
func ExpandGlob(args []string) []string {
	var expandedArgs []string
	for _, arg := range args {
		if strings.ContainsAny(arg, "*?[]") {
			matches, err := filepath.Glob(arg)
			if err == nil && len(matches) > 0 {
				expandedArgs = append(expandedArgs, matches...)
				continue
			}
		}
		expandedArgs = append(expandedArgs, arg)
	}
	return expandedArgs
}
