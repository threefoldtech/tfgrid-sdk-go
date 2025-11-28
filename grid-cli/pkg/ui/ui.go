package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// UI handles user interface interactions
type UI struct {
	reader *bufio.Reader
}

// NewUI creates a new UI
func NewUI() *UI {
	return &UI{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Welcome prints the welcome message
func (u *UI) Welcome() {
	fmt.Println(ColorGreen + "Welcome to ThreefoldGrid Agent! (type 'exit' to quit)" + ColorReset)
}

// GetInput reads a line of input from the user
func (u *UI) GetInput() (string, error) {
	fmt.Print(ColorBold + "> " + ColorReset)
	input, err := u.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// Question prints a question from the agent
func (u *UI) Question(text string) {
	fmt.Println(ColorCyan + "Agent: " + text + ColorReset)
}

// Answer prints an answer from the agent
func (u *UI) Answer(text string) {
	fmt.Println(ColorCyan + "Agent: " + text + ColorReset)
}

// Fetching prints a message indicating a URL is being fetched
func (u *UI) Fetching(reason, url string) {
	fmt.Println(ColorCyan + "Agent: " + reason + ColorReset)
	fmt.Printf(ColorYellow+"Fetching: %s"+ColorReset+"\n", url)
}

// Processing prints a message indicating content is being processed
func (u *UI) Processing() {
	fmt.Println(ColorGray + "Agent is processing the fetched content..." + ColorReset)
}

// Running prints the command being executed
func (u *UI) Running(explanation, command string) {
	fmt.Println(ColorCyan + "Agent: " + explanation + ColorReset)
	fmt.Printf(ColorYellow+"Running: %s"+ColorReset+"\n", command)
}

// Analyzing prints a message indicating the output is being analyzed
func (u *UI) Analyzing() {
	fmt.Println(ColorGray + "Agent is analyzing the output..." + ColorReset)
}

// Error prints an error message from the agent
func (u *UI) Error(text string) {
	fmt.Println(ColorCyan + "Agent: " + text + ColorReset)
}

// PrintOutput prints the output of a command
func (u *UI) PrintOutput(text string) {
	fmt.Println(ColorGray + text + ColorReset)
}

// PrintError prints an error from a command
func (u *UI) PrintError(text string) {
	fmt.Println(ColorRed + text + ColorReset)
}
