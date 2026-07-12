package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client provides access to the GitHub API via the gh CLI.
type Client struct {
	// executable is the path to the gh binary.
	executable string
}

// NewClient creates a new GitHub client that shells out to gh.
func NewClient() *Client {
	return &Client{executable: "gh"}
}

// Get performs a GitHub API GET request and returns the parsed JSON response.
func (c *Client) Get(ctx context.Context, endpoint string) (json.RawMessage, error) {
	args := []string{"api", "--method", "GET", "-H", "Accept: application/json", endpoint}
	return c.exec(ctx, args)
}

// GetWithParams performs a GitHub API GET request with query parameters.
func (c *Client) GetWithParams(ctx context.Context, endpoint string, params map[string]string) (json.RawMessage, error) {
	args := []string{"api", "--method", "GET", "-H", "Accept: application/json"}
	for k, v := range params {
		args = append(args, "-f", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, endpoint)
	return c.exec(ctx, args)
}

func (c *Client) exec(ctx context.Context, args []string) (json.RawMessage, error) {
	cmd := exec.CommandContext(ctx, c.executable, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return nil, fmt.Errorf("gh api: %s: %w", msg, err)
		}
		return nil, fmt.Errorf("gh api: %w", err)
	}
	return json.RawMessage(out), nil
}
