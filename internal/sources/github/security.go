package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// devScopeWeight discounts dev-only dependency alerts: they don't ship to prod.
const devScopeWeight = 0.3

// maxAlertPages limits pagination to avoid fetching thousands of alerts.
const maxAlertPages = 10

// Vulnerability represents a Dependabot alert from the GitHub API.
type Vulnerability struct {
	State            string `json:"state"`
	Severity         string `json:"severity"`
	SecurityAdvisory *struct {
		GhsaID   string `json:"ghsa_id"`
		Severity string `json:"severity"`
	} `json:"security_advisory"`
	SecurityVulnerability *struct {
		Severity string `json:"severity"`
	} `json:"security_vulnerability"`
	Dependency *struct {
		Scope string `json:"scope"`
	} `json:"dependency"`
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
	params := map[string]string{
		"state":    "open",
		"per_page": "100",
	}
	endpoint := fmt.Sprintf("/repos/%s/%s/dependabot/alerts", owner, name)
	data, err := s.client.GetPaginated(ctx, endpoint, params, maxAlertPages)
	if err != nil {
		return nil, err
	}

	alerts, err := decodeAlertArray(data)
	if err != nil {
		return nil, fmt.Errorf("parsing dependabot alerts: %w", err)
	}

	// Dedup by advisory: the same CVE fires one alert per manifest it appears in.
	type advisory struct {
		severity   string
		anyRuntime bool // runtime if any occurrence is runtime; dev-scoped only if all are
	}
	distinct := map[string]*advisory{}
	for i, a := range alerts {
		if a.State != "open" {
			continue
		}

		sev := ""
		key := ""
		if a.SecurityAdvisory != nil {
			sev = a.SecurityAdvisory.Severity
			key = a.SecurityAdvisory.GhsaID
		}
		if sev == "" && a.SecurityVulnerability != nil {
			sev = a.SecurityVulnerability.Severity
		}
		if sev == "" {
			sev = a.Severity
		}
		if key == "" {
			key = fmt.Sprintf("idx-%d", i)
		}
		runtime := a.Dependency == nil || a.Dependency.Scope != "development"

		if adv, ok := distinct[key]; ok {
			adv.anyRuntime = adv.anyRuntime || runtime
		} else {
			distinct[key] = &advisory{severity: sev, anyRuntime: runtime}
		}
	}

	weightedSum := 0.0
	critCount := 0
	highCount := 0
	medCount := 0
	lowCount := 0
	for _, adv := range distinct {
		w := severityWeight(adv.severity)
		if !adv.anyRuntime {
			w *= devScopeWeight
		}
		weightedSum += w
		switch adv.severity {
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

	// Severity is baked into weightedSum; the per-severity counts below are display-only
	// (see model.DisplayMetricTypes) to avoid double-counting severity in the score.
	return []model.SourceMetric{
		{Type: model.MetricTypeSecurityVulnerabilities, Value: weightedSum},
		{Type: model.MetricTypeSecurityCritical, Value: float64(critCount)},
		{Type: model.MetricTypeSecurityHigh, Value: float64(highCount)},
		{Type: model.MetricTypeSecurityMedium, Value: float64(medCount)},
		{Type: model.MetricTypeSecurityLow, Value: float64(lowCount)},
	}, nil
}

// decodeAlertArray handles both single-array and merged-array responses from GetPaginated.
func decodeAlertArray(data json.RawMessage) ([]Vulnerability, error) {
	s := strings.TrimSpace(string(data))

	if len(s) > 0 && s[0] == '[' {
		var alerts []Vulnerability
		if err := json.Unmarshal(data, &alerts); err != nil {
			return nil, err
		}
		return alerts, nil
	}

	return nil, fmt.Errorf("unexpected alert response format")
}
