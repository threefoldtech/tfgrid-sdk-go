package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func executeWithRollingOutput(cmd *exec.Cmd) (string, error) {
	// Create pipes
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	var err error
	if err = cmd.Start(); err != nil {
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
	const maxVisibleLines = 5
	var visibleLines []string
	printedLines := 0

	// ANSI codes
	const (
		ClearLine  = "\033[2K"
		MoveUp     = "\033[1A"
		MoveCol0   = "\r"
		ColorGray  = "\033[90m"
		ColorReset = "\033[0m"
	)

	for line := range outputChan {
		fullOutput.WriteString(line + "\n")

		// Truncate line to avoid wrapping (assuming 80 chars for safety, though terminals vary)
		// This prevents the "MoveUp" calculation from being wrong due to wrapped lines.
		displayLine := line
		if len(displayLine) > 100 {
			displayLine = displayLine[:97] + "..."
		}

		// Add to window
		visibleLines = append(visibleLines, displayLine)
		if len(visibleLines) > maxVisibleLines {
			visibleLines = visibleLines[1:]
		}

		// Clear previously printed lines
		if printedLines > 0 {
			for i := 0; i < printedLines; i++ {
				fmt.Print(MoveUp + ClearLine)
			}
		}

		// Print current window
		fmt.Print(ColorGray)
		for _, l := range visibleLines {
			fmt.Println(l)
		}
		fmt.Print(ColorReset)

		printedLines = len(visibleLines)
	}

	err = cmd.Wait()
	// "Completely disappear"
	if printedLines > 0 {
		for i := 0; i < printedLines; i++ {
			fmt.Print(MoveUp + ClearLine)
		}
	}

	return fullOutput.String(), err
}
