package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// URLTool fetches content from a URL
type URLTool struct{}

func NewURLTool() *URLTool {
	return &URLTool{}
}

func (t *URLTool) Name() string {
	return "fetch_url"
}

func (t *URLTool) Description() string {
	return "Fetch content from a URL"
}

func (t *URLTool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'url' argument")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"content": string(body),
		"status":  resp.Status,
	}, nil
}
