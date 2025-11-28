package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/pkg/schema"
)

// Response struct for agent responses
type Response struct {
	Command     []string `json:"command,omitempty"`
	Question    string   `json:"question,omitempty"`
	Answer      string   `json:"answer,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
	FetchURL    string   `json:"fetch_url,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

// Service handles chat interactions with Gemini
type Service struct {
	client *genai.Client
	cs     *genai.ChatSession
	History []*genai.Content
	cfg    config.Config
	model  *genai.GenerativeModel
}

// NewServiceWithSchema creates a new chat service with dynamic schema generation from a root command
// This is the recommended way to create a service as it ensures the agent knows all available commands
func NewServiceWithSchema(apiKey string, rootCmd *cobra.Command) (*Service, error) {
	// Generate schema from root command
	schemaDef := schema.GenerateSchema(rootCmd)
	schemaJSON, err := json.MarshalIndent(schemaDef, "", "  ")
	if err != nil {
		// Fall back to empty schema on error
		return NewService(apiKey, config.LoadConfig(""))
	}

	// Load config with schema
	cfg := config.LoadConfig(string(schemaJSON))
	
	return NewService(apiKey, cfg)
}

// NewService creates a new chat service
func NewService(apiKey string, cfg config.Config) (*Service, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel(cfg.ModelName)
	model.ResponseMIMEType = cfg.ResponseMIMEType
	model.SystemInstruction = genai.NewUserContent(genai.Text(cfg.SystemPrompt))

	s := &Service{
		client: client,
		cfg:    cfg,
		model:  model,
	}

	cs, err := s.startChatSession()
	if err != nil {
		return nil, err
	}
	s.cs = cs

	return s, nil
}

func (s *Service) startChatSession() (*genai.ChatSession, error) {
	var session *genai.ChatSession
	for i := 0; i < s.cfg.MaxRetries; i++ {
		session = s.model.StartChat()
		if session != nil {
			return session, nil
		}
		if i < s.cfg.MaxRetries-1 {
			waitTime := time.Duration(1<<uint(i)) * time.Second
			log.Warn().Msgf("Failed to start chat session, retrying in %v... (attempt %d/%d)", waitTime, i+1, s.cfg.MaxRetries)
			time.Sleep(waitTime)
		}
	}
	return nil, fmt.Errorf("failed to start chat session with Gemini after %d attempts", s.cfg.MaxRetries)
}

// SendMessage sends a message to the Gemini model
func (s *Service) SendMessage(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Recovered from panic in SendMessage: %v. Restarting session...", r)

			var oldHistory []*genai.Content
			if s.cs != nil {
				oldHistory = s.cs.History
			}

			newCs, err := s.startChatSession()
			if err == nil {
				if len(oldHistory) > 0 {
					newCs.History = oldHistory
				}
				s.cs = newCs
				log.Info().Msg("Session restarted successfully and history restored")
			} else {
				log.Error().Err(err).Msg("Failed to restart session after panic")
			}
		}
	}()
	return s.cs.SendMessage(ctx, parts...)
}

// ParseResponse parses the Gemini response and returns structured responses
func (s *Service) ParseResponse(resp *genai.GenerateContentResponse) ([]Response, error) {
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	txt, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Try to unmarshal as a list first
	var responses []Response
	if err := json.Unmarshal([]byte(txt), &responses); err != nil {
		// If that fails, try as a single object
		var singleResponse Response
		if err := json.Unmarshal([]byte(txt), &singleResponse); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		responses = []Response{singleResponse}
	}

	return responses, nil
}

// GetConfig returns the service configuration
func (s *Service) GetConfig() config.Config {
	return s.cfg
}

// Close closes the Gemini client
func (s *Service) Close() {
	s.client.Close()
}

