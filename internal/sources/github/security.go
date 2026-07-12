package github

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// Vulnerability represents a Dependabot alert from the GitHub API.
type Vulnerability struct {
	State string `json:"state"`
}

func (s *Source) collectSecurityVulnerabilities(ctx context.Context, owner, name string) ([]sources.SourceMetric, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/dependabot/alerts", owner, name)
	data, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var alerts []Vulnerability
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("parsing dependabot alerts: %w", err)
	}

	count := 0
	for _, a := range alerts {
		if a.State == "open" {
			count++
		}
	}

	return []sources.SourceMetric{
		{
			Type:  model.MetricTypeSecurityVulnerabilities,
			Value: float64(count),
		},
	}, nil
}
