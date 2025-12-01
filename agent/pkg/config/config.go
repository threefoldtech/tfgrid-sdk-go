package config

// Config holds agent configuration
type Config struct {
	SystemPrompt     string
	ModelName        string
	ResponseMIMEType string
	MaxRetries       int
	MaxJSONRetries   int
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		SystemPrompt:     "You are a helpful AI assistant. You can use tools to help the user.",
		ModelName:        "gemini-2.5-flash",
		ResponseMIMEType: "application/json",
		MaxRetries:       3,
		MaxJSONRetries:   2,
	}
}
