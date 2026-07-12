package github

import (
	"context"
	"encoding/json"
)

// APIClient abstracts GitHub API access for testability.
type APIClient interface {
	Get(ctx context.Context, endpoint string) (json.RawMessage, error)
	GetWithParams(ctx context.Context, endpoint string, params map[string]string) (json.RawMessage, error)
	// GetPaginated fetches all pages and returns the concatenated results.
	// For array endpoints (like /pulls), returns a JSON array of all items.
	// maxPages limits the number of pages fetched (0 = no limit).
	GetPaginated(ctx context.Context, endpoint string, params map[string]string, maxPages int) (json.RawMessage, error)
}
