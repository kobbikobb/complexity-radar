package github

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/model"
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

func (s *Source) collectSecurityVulnerabilities(ctx context.Context, owner, name string) ([]model.SourceMetric, error) {
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
	critCount := 0
	highCount := 0
	medCount := 0
	lowCount := 0
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
		switch sev {
		case "critical":
			critCount++
		case "high":
			highCount++
		case "medium":
			medCount++
		case "low":
			lowCount++
		}
	}

	return []model.SourceMetric{
		{Type: model.MetricTypeSecurityVulnerabilities, Value: weightedSum},
		{Type: model.MetricTypeSecurityCritical, Value: float64(critCount)},
		{Type: model.MetricTypeSecurityHigh, Value: float64(highCount)},
		{Type: model.MetricTypeSecurityMedium, Value: float64(medCount)},
		{Type: model.MetricTypeSecurityLow, Value: float64(lowCount)},
	}, nil
}
