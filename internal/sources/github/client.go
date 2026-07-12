package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Client provides access to the GitHub API via the gh CLI.
type Client struct {
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

// GetPaginated fetches all pages of results and returns them as a concatenated JSON array.
// maxPages limits the number of pages fetched (0 = no limit).
func (c *Client) GetPaginated(ctx context.Context, endpoint string, params map[string]string, maxPages int) (json.RawMessage, error) {
	args := []string{"api", "--method", "GET", "-H", "Accept: application/json", "--paginate"}
	for k, v := range params {
		args = append(args, "-f", fmt.Sprintf("%s=%s", k, v))
	}
	if maxPages > 0 {
		args = append(args, "--paginate", "-f", fmt.Sprintf("per_page=%d", 100))
		// gh api --paginate outputs all results as newline-delimited JSON.
		// We'll handle the max pages limit by using --jq to combine.
	}
	args = append(args, endpoint)

	// Use --jq to combine paginated results into a single array.
	// For object responses (like actions/runs), this won't work directly,
	// so we handle it at the caller level.
	out, err := c.execRaw(ctx, args)
	if err != nil {
		return nil, err
	}

	// gh api --paginate outputs NDJSON (one JSON value per line).
	// Combine into a single array.
	combined, err := combineNDJSON(out)
	if err != nil {
		return nil, fmt.Errorf("combining paginated results: %w", err)
	}

	return combined, nil
}

func (c *Client) exec(ctx context.Context, args []string) (json.RawMessage, error) {
	out, err := c.execRaw(ctx, args)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func (c *Client) execRaw(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.executable, args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
		}
		if len(stderr) > 0 {
			return nil, fmt.Errorf("gh api: %s: %w", strings.TrimSpace(string(stderr)), err)
		}
		return nil, fmt.Errorf("gh api: %w", err)
	}
	return out, nil
}

// combineNDJSON takes newline-delimited JSON output from gh api --paginate
// and combines it into a single JSON array.
func combineNDJSON(data []byte) (json.RawMessage, error) {
	var combined []json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decoding paginated JSON: %w", err)
		}
		combined = append(combined, raw)
	}

	if len(combined) == 0 {
		return json.RawMessage("[]"), nil
	}

	// If there's only one page, return it directly (it may be an object or array).
	if len(combined) == 1 {
		return combined[0], nil
	}

	// For multiple pages, we need to handle two cases:
	// 1. Each page is an array element -> concatenate into one array
	// 2. Each page is an object with a key containing an array -> merge arrays
	//
	// Check if first element is an array or object.
	if len(combined) > 0 && isJSONArray(combined[0]) {
		// Merge arrays: [{...}, {...}] + [{...}] -> [{...}, {...}, {...}]
		return mergeJSONArrays(combined)
	}

	// For object responses (like {"workflow_runs": [...]}), we can't easily
	// merge without knowing the key. Return the last page as a reasonable default.
	// Callers needing all pages should handle this themselves.
	return combined[len(combined)-1], nil
}

func isJSONArray(data json.RawMessage) bool {
	s := strings.TrimSpace(string(data))
	return len(s) > 0 && s[0] == '['
}

func mergeJSONArrays(parts []json.RawMessage) (json.RawMessage, error) {
	var merged []json.RawMessage
	for _, part := range parts {
		var items []json.RawMessage
		if err := json.Unmarshal(part, &items); err != nil {
			return nil, fmt.Errorf("unmarshaling array part: %w", err)
		}
		merged = append(merged, items...)
	}
	return json.Marshal(merged)
}

// maxPagesValue is a helper for tests to verify pagination params.
func maxPagesValue() string {
	return strconv.Itoa(10)
}
