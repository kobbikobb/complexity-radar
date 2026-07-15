package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func (s *Source) collectStalePRs(ctx context.Context, owner, name string) ([]model.SourceMetric, error) {
	params := map[string]string{
		"state":     "open",
		"per_page":  "100",
		"sort":      "updated",
		"direction": "asc", // stalest PRs first so the stale tail survives page truncation
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
	var idle7, idle14, idle30, idle60 int

	for _, pr := range prs {
		if pr.State != "open" {
			continue
		}
		// Bots count: un-actioned dependabot/Snyk dep bumps are real dependency/security debt.
		if pr.Draft {
			continue
		}
		updated, err := time.Parse(time.RFC3339, pr.UpdatedAt)
		if err != nil {
			continue
		}
		days := now.Sub(updated).Hours() / 24
		switch {
		case days >= 60:
			idle60++
			fallthrough
		case days >= 30:
			idle30++
			fallthrough
		case days >= 14:
			idle14++
			fallthrough
		case days >= 7:
			idle7++
		}
	}

	log.Printf("stale PRs by idle days: >=7:%d >=14:%d >=30:%d >=60:%d", idle7, idle14, idle30, idle60)

	return []model.SourceMetric{
		{Type: model.MetricTypeStalePRs, Value: float64(idle7)},
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
