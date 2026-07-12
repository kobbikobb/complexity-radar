package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	State     string `json:"state"`
	UpdatedAt string `json:"updated_at"`
}

func (s *Source) collectStalePRs(ctx context.Context, owner, name string) ([]sources.SourceMetric, error) {
	params := map[string]string{
		"state":     "open",
		"per_page":  "100",
		"sort":      "updated",
		"direction": "desc",
	}
	endpoint := fmt.Sprintf("/repos/%s/%s/pulls", owner, name)
	data, err := s.client.GetWithParams(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var prs []PullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("parsing pull requests: %w", err)
	}

	now := time.Now()
	staleThreshold := now.AddDate(0, 0, -14)
	staleCount := 0

	for _, pr := range prs {
		if pr.State != "open" {
			continue
		}
		updated, err := time.Parse(time.RFC3339, pr.UpdatedAt)
		if err != nil {
			continue
		}
		if updated.Before(staleThreshold) {
			staleCount++
		}
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeStalePRs, Value: float64(staleCount)},
	}, nil
}
