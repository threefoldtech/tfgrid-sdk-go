package tfcmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const getCommandName = "get"

// CommandSchema struct for command schema
type CommandSchema struct {
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	Args          string           `json:"args,omitempty"`
	Flags         []FlagSchema     `json:"flags,omitempty"`
	RequiredFlags []string         `json:"required_flags,omitempty"`
	FlagGroups    [][]string       `json:"flag_groups,omitempty"`
	SubCommands   []*CommandSchema `json:"subcommands,omitempty"`
}

// FlagSchema struct for flag schema
type FlagSchema struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

// GenerateSchema generates the schema for a cobra command
func GenerateSchema(cmd *cobra.Command) *CommandSchema {
	schema := &CommandSchema{
		Name:        cmd.Name(),
		Description: cmd.Short,
	}

	if cmd.Args != nil {
		switch cmd.Use {
		case "vm":
			if cmd.Parent() != nil && cmd.Parent().Name() == getCommandName {
				schema.Args = "<vm-name> (required positional argument)"
			}
		case "kubernetes":
			if cmd.Parent() != nil && cmd.Parent().Name() == getCommandName {
				schema.Args = "<kubernetes-name> (required positional argument)"
			}
		case "gateway":
			if cmd.Parent() != nil && cmd.Parent().Name() == getCommandName {
				schema.Args = "<gateway-name> (required positional argument)"
			}
		case "zdb":
			if cmd.Parent() != nil && cmd.Parent().Name() == getCommandName {
				schema.Args = "<zdb-name> (required positional argument)"
			}
		}
	}

	requiredFlags := make(map[string]bool)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
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
		schema.SubCommands = append(schema.SubCommands, GenerateSchema(sub))
	}

	return schema
}
