package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/api/option"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with the grid agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY environment variable is not set")
		}

		ctx := context.Background()
		client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			return err
		}
		defer client.Close()

		model := client.GenerativeModel("gemini-2.5-flash")
		model.ResponseMIMEType = "application/json"

		schema := generateSchema(rootCmd)
		schemaJSON, _ := json.MarshalIndent(schema, "", "  ")

		systemPrompt := fmt.Sprintf(`You are an intelligent agent for the tf-grid CLI.
Your goal is to help the user interact with the CLI using natural language.
You have access to the following CLI commands and flags:
%s

IMPORTANT VALIDATION RULES:
1. Check that ALL required flags are provided (look for "required": true in the schema)
2. Check the "required_flags" array for each command
3. If any required flags are missing, ask the user for that information instead of running the command

4. CRITICAL FLAG GROUPS - these flags MUST be set together:
   - deploy vm: if --flist is provided, --entrypoint MUST also be provided (and vice versa)
   
   BEFORE generating ANY command with --flist:
   a) Check if --entrypoint is already included in the command
   b) If missing, check if user provided entrypoint info earlier in the conversation
   c) For application flists (WordPress, Presearch, etc.):
      - Check the GitHub README.md - it often specifies the required entrypoint
      - If found in README, use that entrypoint automatically
   d) If still missing after checking README, STOP and ask: "This flist requires an entrypoint. Would you like to use the default '/sbin/zinit init', or provide a custom one?"
   e) Only proceed after user confirms
   
   When user provides a custom flist:
   - First check if it's an application flist - look up the README for entrypoint info
   - If README specifies an entrypoint, use it automatically
   - If not found in README, ask user for entrypoint or offer default '/sbin/zinit init'
   - Never execute without explicit entrypoint confirmation

5. MUTUALLY EXCLUSIVE FLAGS - only ONE of these can be set:
   - deploy vm: --node OR --farm (not both)
   - deploy kubernetes: --master-node OR --master-farm (not both)
   - deploy kubernetes: --workers-nodes OR --workers-farm (not both)
   - add worker kubernetes: --workers-nodes OR --workers-farm (not both)
   - deploy gateway name: --node OR --farm (not both)
   - deploy zdb: --node OR --farm (not both)

6. BOOLEAN FLAG SYNTAX:
   - To enable: --flag or --flag=true
   - To disable: --flag=false (MUST use = sign)
   - WRONG: --mycelium false
   - CORRECT: --mycelium=false

7. HANDLING SSH KEYS IN ENVIRONMENT VARIABLES:
   - Some application flists require an SSH key passed as an ENV VAR (e.g., SSH_KEY, pub_key, public_key)
   - This is DIFFERENT from the --ssh flag (which takes a file path)
   - If an ENV VAR requires an SSH key:
     a) You CANNOT pass the file path (e.g., /home/user/.ssh/id_rsa.pub) as the value
     b) You MUST pass the actual CONTENT of the key (e.g., "ssh-rsa AAA...")
     c) If you only have the file path:
        1. First, run: cat /path/to/key.pub
        2. Read the output (the key content)
        3. Then construct the deploy command using the content: --env SSH_KEY="ssh-rsa AAA..."

CONSULTATIVE APPROACH FOR DEPLOYMENTS:
When a user wants to deploy a resource (VM, Kubernetes, Gateway, ZDB):
1. First, gather ALL required information (name, ssh key, env vars, etc.)
2. IMPORTANT: Before executing, ALWAYS ask about optional configurations:
   - Group related options logically (resources, storage, networking, advanced)
   - Explain what each option does and its impact
   - Mention default values clearly
   - Ask: "Would you like to customize [resources/storage/networking], or use the defaults?"
3. Only execute AFTER the user has confirmed the configuration (either customized or accepted defaults)

Example flow for VM deployment:
- Required: name, ssh key, any app-specific env vars
- Then ASK: "The default configuration is 1 CPU, 1GB memory, 2GB rootfs. Would you like to customize resources, add storage, or configure networking?"
- Wait for user response before executing

For application deployments (WordPress, Presearch, etc.):
- First lookup the app's README from GitHub to find required env vars and flist
- Gather all required info including env vars
- Then ask about optional resource configurations
- Only deploy after user confirms

IMPORTANT - CANCEL/DELETE COMMANDS:
To delete a deployment, there are TWO options:
1. By deployment name: tfcmd cancel <deployment-name> (e.g., tfcmd cancel pre02)
   - This is the EASIEST way to delete a single deployment
   - Use the same name that was used during deployment
2. By contract ID: tfcmd cancel contracts <contract-id> [contract-id...]
   - Can cancel one or more specific contracts by their IDs
   - Use tfcmd cancel contracts -a to cancel ALL contracts
ALWAYS prefer option 1 (cancel by name) for single deployments!

CRITICAL SAFETY - Cancel All Contracts:
BEFORE running "tfcmd cancel contracts -a" or "tfcmd cancel contracts --all":
1. This command will DELETE ALL CONTRACTS - VMs, Kubernetes, Gateways, ZDBs, EVERYTHING
2. You MUST explicitly warn the user about this destructive action
3. You MUST ask for explicit confirmation: "Are you absolutely sure you want to delete ALL your contracts? This will remove all deployed resources. Please confirm."
4. ONLY proceed if user gives a CLEAR AFFIRMATION (e.g., "yes", "yeah", "ok", "sure", "confirm", "do it", "proceed")
5. If user shows ANY hesitation, ambiguity, or says no/wait/cancel, DO NOT execute the command
6. Use your judgment - if the response is clearly affirmative, proceed; if there's any doubt, ask again or abort
Never execute "cancel contracts -a" without this explicit confirmation!

EXTERNAL INFORMATION LOOKUP:
If you need to look up information, you can fetch from these sources:
- https://hub.grid.tf/api/flist/tf-official-vms - Operating system flists (Ubuntu, Alpine, NixOS, etc.)
- https://hub.grid.tf/api/flist/tf-official-apps - Application flists (WordPress, Peertube, etc.)
- https://github.com/threefoldtech/tf-images/tree/development/tfgrid3/ - App deployment info (required env vars, entrypoints)
  * For apps like WordPress, Presearch, etc., check the README.md in the specific app folder
  * Example: https://github.com/threefoldtech/tf-images/tree/development/tfgrid3/wordpress/README.md
- https://github.com/threefoldtech/tfgrid-sdk-go/blob/development/grid-cli/README.md - Grid CLI documentation and usage examples
- https://manual.grid.tf/labs/documentation/ - Grid documentation and knowledge base

IMPORTANT - Flist Priority:
1. ALWAYS prefer flists from hub.grid.tf/api/flist/tf-official-apps or hub.grid.tf/api/flist/tf-official-vms (these are official)
2. Use GitHub README.md ONLY for environment variables and entrypoint information
3. If README mentions a flist URL, check hub.grid.tf API first - the official version takes precedence
4. Only use README flist URLs if no official version exists on the hub

For application deployments:
1. First check hub.grid.tf/api/flist/tf-official-apps for the official flist URL
2. Then check GitHub README for required env vars and entrypoint
3. Combine: official flist URL + env vars from README

Choose the appropriate source based on user request:
{
  "fetch_url": "https://hub.grid.tf/api/flist/tf-official-vms",
  "reason": "I will lookup available Ubuntu flists",
  "explanation": "Let me check what's available..."
}
I will fetch the content and provide it to you, then you can extract the needed information.

RESPONSE FORMATS:
If you can infer a COMPLETE and VALID CLI command to run:
{
  "command": ["tfcmd", "subcommand", "--flag", "value"],
  "explanation": "I will run this command to..."
}

If you need more information from the user (including missing required flags or optional configs):
{
  "question": "I need to know...",
  "explanation": "I need this info because..."
}

If you need to fetch external information:
{
  "fetch_url": "https://hub.grid.tf/api/flist/tf-official-apps",
  "reason": "I will lookup available application flists...",
  "explanation": "Checking for official flist"
}
Note: The "reason" should be phrased as an action you're taking, e.g., "I will lookup required environment variables for WordPress"

If the user asks a general question, answer it:
{
  "answer": "The answer is...",
  "explanation": "..."
}

Always prefer using the CLI commands if possible.
The user might refer to previous context.
Never execute a command with missing required flags - always ask first.
Be consultative and educational - help users understand their options.
`, string(schemaJSON))

		model.SystemInstruction = genai.NewUserContent(genai.Text(systemPrompt))

		// Retry StartChat with exponential backoff (in case of rate limits or transient failures)
		var cs *genai.ChatSession
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			cs = model.StartChat()
			if cs != nil {
				break
			}
			if i < maxRetries-1 {
				waitTime := time.Duration(1<<uint(i)) * time.Second // 1s, 2s, 4s
				log.Warn().Msgf("Failed to start chat session, retrying in %v... (attempt %d/%d)", waitTime, i+1, maxRetries)
				time.Sleep(waitTime)
			}
		}

		if cs == nil {
			return fmt.Errorf("failed to start chat session with Gemini after %d attempts", maxRetries)
		}

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

		reader := bufio.NewReader(os.Stdin)
		fmt.Println(ColorGreen + "Welcome to ThreefoldGrid Agent! (type 'exit' to quit)" + ColorReset)

		for {
			fmt.Print(ColorBold + "> " + ColorReset)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "exit" {
				break
			}

			if input == "" {
				continue
			}

			resp, err := cs.SendMessage(ctx, genai.Text(input))
			if err != nil {
				log.Error().Err(err).Msg("Error sending message to Gemini")
				continue
			}

			if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
				log.Error().Msg("No response from Gemini")
				continue
			}

			part := resp.Candidates[0].Content.Parts[0]
			txt, ok := part.(genai.Text)
			if !ok {
				log.Error().Msg("Unexpected response format")
				continue
			}

			// Process the response in a loop to handle retries/chains
			for {
				type Response struct {
					Command     []string `json:"command,omitempty"`
					Question    string   `json:"question,omitempty"`
					Answer      string   `json:"answer,omitempty"`
					Explanation string   `json:"explanation,omitempty"`
					FetchURL    string   `json:"fetch_url,omitempty"`
					Reason      string   `json:"reason,omitempty"`
				}

				var responses []Response

				// Try to unmarshal as a list first
				if err := json.Unmarshal([]byte(txt), &responses); err != nil {
					// If that fails, try as a single object
					var singleResponse Response
					if err := json.Unmarshal([]byte(txt), &singleResponse); err != nil {
						log.Error().Err(err).Msg("Error parsing response")
						fmt.Println(ColorCyan + "Agent: " + string(txt) + ColorReset)
						break
					}
					responses = []Response{singleResponse}
				}

				// Process all responses
				for _, response := range responses {
					if response.Question != "" {
						fmt.Println(ColorCyan + "Agent: " + response.Question + ColorReset)
						// For questions, we break to get user input
						// But if there are multiple items, this might be tricky.
						// Usually questions come alone.
						goto EndProcessing
					} else if response.Answer != "" {
						fmt.Println(ColorCyan + "Agent: " + response.Answer + ColorReset)
					} else if response.FetchURL != "" {
						// Handle URL fetching
						fmt.Println(ColorCyan + "Agent: " + response.Reason + ColorReset)
						fmt.Printf(ColorYellow+"Fetching: %s"+ColorReset+"\n", response.FetchURL)

						// Use a simple HTTP GET to fetch the content
						// Note: In production, you might want to use the read_url_content tool
						// For now, we'll use Go's http package
						httpResp, err := http.Get(response.FetchURL)
						if err != nil {
							log.Error().Err(err).Msg("Failed to fetch URL")
							feedback := fmt.Sprintf("Failed to fetch URL: %v", err)
							resp, err := cs.SendMessage(ctx, genai.Text(feedback))
							if err != nil {
								log.Error().Err(err).Msg("Error sending feedback to Gemini")
								break
							}
							if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
								if t, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
									txt = t
									continue
								}
							}
							break
						}
						defer httpResp.Body.Close()

						body, err := io.ReadAll(httpResp.Body)
						if err != nil {
							log.Error().Err(err).Msg("Failed to read response body")
							break
						}

						// Send the fetched content back to the agent
						feedback := fmt.Sprintf("Fetched content from %s:\n\n%s", response.FetchURL, string(body))
						fmt.Println(ColorGray + "Agent is processing the fetched content..." + ColorReset)
						resp, err := cs.SendMessage(ctx, genai.Text(feedback))
						if err != nil {
							log.Error().Err(err).Msg("Error sending feedback to Gemini")
							goto EndProcessing
						}

						if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
							part := resp.Candidates[0].Content.Parts[0]
							if t, ok := part.(genai.Text); ok {
								txt = t
								// Loop continues with new response
								// We need to break out of the range loop and continue the outer "process response" loop
								// But wait, the outer loop IS the process response loop.
								// We actually need to restart the outer loop with new 'txt'.
								// Since we are inside a range loop, we can't just 'continue' the outer loop easily.
								// Let's use a flag or goto.
								goto ProcessNewResponse
							}
						}
						goto EndProcessing
					} else if len(response.Command) > 0 {
						fmt.Println(ColorCyan + "Agent: " + response.Explanation + ColorReset)
						fmt.Printf(ColorYellow+"Running: %s"+ColorReset+"\n", strings.Join(response.Command, " "))

						// Execute the command
						var cmd *exec.Cmd
						if response.Command[0] == "tfcmd" {
							cmdArgs := response.Command[1:]
							exe, err := os.Executable()
							if err != nil {
								log.Error().Err(err).Msg("Failed to get executable path")
								goto EndProcessing
							}
							cmd = exec.Command(exe, cmdArgs...)
						} else {
							cmd = exec.Command(response.Command[0], response.Command[1:]...)
						}

						// Use rolling output helper
						output, err := executeWithRollingOutput(cmd)

						// Print output to user (it was already printed by rolling output, but we might want to keep it?
						// User said: "completely disappear and agent start to talk"
						// executeWithRollingOutput clears the output at the end.
						// So we don't need to print it again.
						// But we need to capture it for the agent feedback.

						// Prepare feedback for the agent
						feedback := fmt.Sprintf("Command executed.\nOutput:\n%s", output)
						if err != nil {
							feedback += fmt.Sprintf("\nError: %v", err)
						}

						fmt.Println(ColorGray + "Agent is analyzing the output..." + ColorReset)
						resp, err := cs.SendMessage(ctx, genai.Text(feedback))
						if err != nil {
							log.Error().Err(err).Msg("Error sending feedback to Gemini")
							goto EndProcessing
						}

						if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
							part := resp.Candidates[0].Content.Parts[0]
							if t, ok := part.(genai.Text); ok {
								txt = t
								// Same here, restart processing with new response
								goto ProcessNewResponse
							}
						}
						goto EndProcessing
					} else {
						// No command, question, or answer?
						// Just continue to next item
					}
				}
				// If we finished the range loop without jumping, we are done
				break

			ProcessNewResponse:
				continue

			EndProcessing:
				break
			}
		}

		return nil
	},
}

type CommandSchema struct {
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	Args          string           `json:"args,omitempty"`
	Flags         []FlagSchema     `json:"flags,omitempty"`
	RequiredFlags []string         `json:"required_flags,omitempty"`
	FlagGroups    [][]string       `json:"flag_groups,omitempty"`
	SubCommands   []*CommandSchema `json:"subcommands,omitempty"`
}

type FlagSchema struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

func generateSchema(cmd *cobra.Command) *CommandSchema {
	schema := &CommandSchema{
		Name:        cmd.Name(),
		Description: cmd.Short,
	}

	// Capture positional argument requirements
	if cmd.Args != nil {
		// Try to infer the args requirement from common validators
		// This is a best-effort approach since cobra doesn't expose Args info directly
		switch cmd.Use {
		case "vm":
			if cmd.Parent() != nil && cmd.Parent().Name() == "get" {
				schema.Args = "<vm-name> (required positional argument)"
			}
		case "kubernetes":
			if cmd.Parent() != nil && cmd.Parent().Name() == "get" {
				schema.Args = "<kubernetes-name> (required positional argument)"
			}
		case "gateway":
			if cmd.Parent() != nil && cmd.Parent().Name() == "get" {
				schema.Args = "<gateway-name> (required positional argument)"
			}
		case "zdb":
			if cmd.Parent() != nil && cmd.Parent().Name() == "get" {
				schema.Args = "<zdb-name> (required positional argument)"
			}
		}
	}

	// Collect required flags
	requiredFlags := make(map[string]bool)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Check if flag is required via annotations
		if annotations := f.Annotations; annotations != nil {
			if _, ok := annotations[cobra.BashCompOneRequiredFlag]; ok {
				requiredFlags[f.Name] = true
			}
		}
	})

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		isRequired := requiredFlags[f.Name]
		defaultValue := f.DefValue
		schema.Flags = append(schema.Flags, FlagSchema{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			Description: f.Usage,
			Type:        f.Value.Type(),
			Required:    isRequired,
			Default:     defaultValue,
		})
		if isRequired {
			schema.RequiredFlags = append(schema.RequiredFlags, f.Name)
		}
	})

	for _, sub := range cmd.Commands() {
		if sub.Name() == "help" || sub.Name() == "completion" || sub.Name() == "chat" {
			continue
		}
		schema.SubCommands = append(schema.SubCommands, generateSchema(sub))
	}

	return schema
}

func init() {
	// We don't add chatCmd to rootCmd here to avoid circular dependency if we were in a different package,
	// but since we are in `cmd`, we can do it in root.go or here.
	// However, root.go is where other commands are added usually.
	// Let's just export it or add it in init() if root is available.
	// Actually, creating it here is fine, we just need to register it in root.go
}
