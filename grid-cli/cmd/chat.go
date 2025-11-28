package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/chat"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/runner"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/ui"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/workflow"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with the grid agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY environment variable is not set")
		}

		// Create chat service with schema (reusing shared logic)
		chatService, err := chat.NewServiceWithSchema(apiKey, rootCmd)
		if err != nil {
			return err
		}
		defer chatService.Close()

		// Create UI
		chatUI := ui.NewUI()
		chatUI.Welcome()

		// Create CLI-specific handler and executor
		handler := &cliResponseHandler{ui: chatUI}
		executor := &cliCommandExecutor{}

		// Create processor with CLI implementations
		processor := workflow.NewResponseProcessor(chatService, handler, executor)

		ctx := context.Background()

		for {
			input, err := chatUI.GetInput()
			if err != nil {
				return err
			}

			if strings.TrimSpace(input) == "" {
				continue
			}

			// Check for exit command
			if strings.TrimSpace(strings.ToLower(input)) == "exit" {
				chatUI.Answer("Goodbye!")
				break
			}

			resp, err := chatService.SendMessage(ctx, genai.Text(input))
			if err != nil {
				log.Error().Err(err).Msg("Error sending message to Gemini")
				continue
			}

			// Use processor (replaces the entire ResponseLoop!)
			_, err = processor.ProcessResponse(ctx, resp)
			if err != nil {
				log.Error().Err(err).Msg("Error processing response")
				chatUI.Error(err.Error())
			}
		}

		return nil
	},
}

// cliResponseHandler implements workflow.ResponseHandler for CLI-specific UI and logging
type cliResponseHandler struct {
	ui *ui.UI
}

func (h *cliResponseHandler) OnQuestion(question string) error {
	h.ui.Question(question)
	return nil
}

func (h *cliResponseHandler) OnAnswer(answer string) error {
	h.ui.Answer(answer)
	return nil
}

func (h *cliResponseHandler) OnFetchURL(reason, url string) {
	h.ui.Fetching(reason, url)
}

func (h *cliResponseHandler) OnProcessing() {
	h.ui.Processing()
}

func (h *cliResponseHandler) OnCommand(explanation, command string) {
	h.ui.Running(explanation, command)
}

func (h *cliResponseHandler) OnAnalyzing() {
	h.ui.Analyzing()
}

func (h *cliResponseHandler) OnError(message string) {
	log.Error().Msg("Error parsing response")
	h.ui.Error(message)
}

// cliCommandExecutor implements workflow.CommandExecutor for CLI-specific command execution
type cliCommandExecutor struct{}

func (e *cliCommandExecutor) Execute(command []string) (string, error) {
	var cmd *exec.Cmd

	// Handle tfcmd specially (self-execution)
	if command[0] == "tfcmd" {
		cmdArgs := command[1:]
		exe, err := os.Executable()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get executable path")
			return "", err
		}
		cmd = exec.Command(exe, cmdArgs...)
	} else {
		cmd = exec.Command(command[0], command[1:]...)
	}

	// Expand tilde and globs in arguments
	for i, arg := range cmd.Args {
		cmd.Args[i] = runner.ExpandTilde(arg)
	}
	cmd.Args = runner.ExpandGlob(cmd.Args)

	// Use rolling output helper (preserves the rolling effect!)
	output, err := executeWithRollingOutput(cmd)

	return output, err
}
