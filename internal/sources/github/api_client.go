package github

import (
	"context"
	"encoding/json"
)

// APIClient abstracts GitHub API access for testability.
type APIClient interface {
	Get(ctx context.Context, endpoint string) (json.RawMessage, error)
	GetWithParams(ctx context.Context, endpoint string, params map[string]string) (json.RawMessage, error)
}
