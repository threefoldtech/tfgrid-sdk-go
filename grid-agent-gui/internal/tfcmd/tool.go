package tfcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/agent/pkg/tools"
)

// Tool implements the agent.Tool interface for tfcmd
type Tool struct {
	schema   *CommandSchema
	executor *Executor
}

// NewTool creates a new tfcmd tool
func NewTool(rootCmd *cobra.Command) *Tool {
	return &Tool{
		schema:   GenerateSchema(rootCmd),
		executor: NewExecutor(),
	}
}

const ToolName = "tfcmd"

func (t *Tool) Name() string {
	return ToolName
}

func (t *Tool) Description() string {
	schemaJSON, _ := json.MarshalIndent(t.schema, "", "  ")
	return fmt.Sprintf("Execute Threefold Grid commands. Available commands schema:\n%s", string(schemaJSON))
}

func (t *Tool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Parse args to build command
	// Expecting args["command"] as string or list of strings
	var cmdArgs []string

	if cmdStr, ok := args["command"].(string); ok {
		cmdArgs = strings.Fields(cmdStr)
	} else if cmdList, ok := args["command"].([]interface{}); ok {
		for _, arg := range cmdList {
			cmdArgs = append(cmdArgs, fmt.Sprint(arg))
		}
	} else {
		return nil, fmt.Errorf("missing or invalid 'command' argument")
	}

	output, err := t.executor.Execute(cmdArgs)

	result := map[string]any{
		"output": output,
	}
	if err != nil {
		result["error"] = err.Error()
	}

	return result, nil
}

// Ensure Tool implements tools.Tool
var _ tools.Tool = (*Tool)(nil)
