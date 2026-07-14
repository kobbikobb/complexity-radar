package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	State     string `json:"state"`
	UpdatedAt string `json:"updated_at"`
	Head      struct {
		Ref string `json:"ref"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
	Draft bool `json:"draft"`
}

// maxStalePRPages limits pagination to avoid fetching thousands of PRs.
const maxStalePRPages = 10

// isBotPR returns true for PRs likely created by bots/agents.
func isBotPR(pr PullRequest) bool {
	if pr.User.Type == "Bot" {
		return true
	}
	branch := pr.Head.Ref
	prefixes := []string{"agent-", "wf_", "dependabot/", "renovate/", "bot/", "merge-bot", "worktree/"}
	for _, p := range prefixes {
		if strings.HasPrefix(branch, p) {
			return true
		}
	}
	return false
}

func (s *Source) collectStalePRs(ctx context.Context, owner, name string) ([]model.SourceMetric, error) {
	params := map[string]string{
		"state":     "open",
		"per_page":  "100",
		"sort":      "updated",
		"direction": "desc",
	}
	endpoint := fmt.Sprintf("/repos/%s/%s/pulls", owner, name)
	data, err := s.client.GetPaginated(ctx, endpoint, params, maxStalePRPages)
	if err != nil {
		return nil, err
	}

	prs, err := decodePRArray(data)
	if err != nil {
		return nil, fmt.Errorf("parsing pull requests: %w", err)
	}

	now := time.Now()
	staleThreshold := now.AddDate(0, 0, -14)
	staleCount := 0

	for _, pr := range prs {
		if pr.State != "open" {
			continue
		}
		if isBotPR(pr) || pr.Draft {
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

	return []model.SourceMetric{
		{Type: model.MetricTypeStalePRs, Value: float64(staleCount)},
	}, nil
}

// decodePRArray handles both single-array and merged-array responses.
func decodePRArray(data json.RawMessage) ([]PullRequest, error) {
	s := strings.TrimSpace(string(data))

	// Single array: [{...}, {...}]
	if len(s) > 0 && s[0] == '[' {
		var prs []PullRequest
		if err := json.Unmarshal(data, &prs); err != nil {
			return nil, err
		}
		return prs, nil
	}

	return nil, fmt.Errorf("unexpected PR response format")
}
