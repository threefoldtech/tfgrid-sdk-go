# ANSI Rendering Implementation

## What Was Added

ANSI color code rendering in the GUI to preserve colored terminal output.

### Changes Made

1. **Package installed**: `ansi-to-html` (in `frontend/package.json`)
2. **Files modified**:
   - `frontend/src/components/ChatMessage.svelte`
   - `frontend/src/components/CommandOutput.svelte`

### How It Works

- Command output with ANSI codes (e.g., `[32mINF[0m`) is converted to HTML with colors
- Uses `ansi-to-html` library to parse ANSI escape sequences
- Renders as colored HTML using `{@html renderAnsi(text)}`

## To Revert (If Output Looks Bad)

### Option 1: Quick Revert (Strip ANSI Instead)

Add this to `grid-agent-gui/app.go`:

```go
import "regexp"

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
    return ansiRegex.ReplaceAllString(s, "")
}

// In executeCommand():
output, err := cmd.CombinedOutput()
cleanOutput := stripANSI(string(output))
return cleanOutput, err
```

### Option 2: Full Revert (Remove ANSI Rendering)

```bash
cd grid-agent-gui/frontend

# Remove package
npm uninstall ansi-to-html

# Revert ChatMessage.svelte
git checkout frontend/src/components/ChatMessage.svelte

# Revert CommandOutput.svelte
git checkout frontend/src/components/CommandOutput.svelte

# Rebuild
npm run build
```

### Option 3: Disable ANSI at Source (Cleanest)

Modify `grid-cli/cmd/root.go`:

```go
func init() {
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
    
    // Check if running from GUI
    if os.Getenv("NO_COLOR") != "" {
        log.Logger = log.Output(os.Stderr)  // Plain text
    } else {
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})  // Colored
    }
    
    rootCmd.PersistentFlags().Bool("disable-sentry", false, "disable sentry")
    rootCmd.AddCommand(chatCmd)
}
```

Then in `grid-agent-gui/app.go`:

```go
func (a *App) executeCommand(command []string) (string, error) {
    // ...
    cmd.Env = append(os.Environ(), "NO_COLOR=1")
    // ...
}
```

## Testing

Test with a command that produces colored output:
```
User: "cancel all deployments"
```

Expected output should show:
- Green `INF` logs
- Gray timestamps
- Cyan project names

If colors look wrong or unreadable, use one of the revert options above.
