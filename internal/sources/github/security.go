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
	State                 string `json:"state"`
	Severity              string `json:"severity"`
	SecurityVulnerability *struct {
		Severity string `json:"severity"`
	} `json:"security_vulnerability"`
}

func severityWeight(sev string) float64 {
	switch sev {
	case "critical":
		return 1.0
	case "high":
		return 0.7
	case "medium":
		return 0.3
	case "low":
		return 0.1
	case "info":
		return 0.0
	default:
		return 0.1
	}
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

	total := 0
	weightedSum := 0.0
	for _, a := range alerts {
		if a.State != "open" {
			continue
		}
		total++

		sev := a.Severity
		if a.SecurityVulnerability != nil && a.SecurityVulnerability.Severity != "" {
			sev = a.SecurityVulnerability.Severity
		}
		weightedSum += severityWeight(sev)
	}

	return []sources.SourceMetric{
		{
			Type:  model.MetricTypeSecurityVulnerabilities,
			Value: weightedSum,
		},
	}, nil
}
