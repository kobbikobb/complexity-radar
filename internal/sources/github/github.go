package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// Source collects metrics from GitHub via the gh CLI.
type Source struct {
	client APIClient
}

// NewSource creates a new GitHub source with the default gh client.
func NewSource() *Source {
	return &Source{client: NewClient()}
}

// NewSourceWithClient creates a GitHub source with a custom client (for testing).
func NewSourceWithClient(client APIClient) *Source {
	return &Source{client: client}
}

func (s *Source) Name() string {
	return "github"
}

func (s *Source) SupportedMetrics() []model.MetricTypeName {
	return []model.MetricTypeName{
		model.MetricTypeSecurityVulnerabilities,
		model.MetricTypeBuildSuccessRatio,
		model.MetricTypeBuildTime,
		model.MetricTypeDeployFrequency,
		model.MetricTypeStalePRs,
	}
}

func (s *Source) Collect(ctx context.Context, repo model.Repository) ([]sources.SourceMetric, error) {
	owner, name, err := parseRepoURL(repo.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing repo URL %q: %w", repo.URL, err)
	}

	branch := repo.Branch
	if branch == "" {
		branch = "main"
	}

	var metrics []sources.SourceMetric

	m, err := s.collectSecurityVulnerabilities(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("collecting security vulnerabilities: %w", err)
	}
	metrics = append(metrics, m...)

	// Fetch workflow runs once and reuse for both build metrics.
	runs, err := s.fetchWorkflowRuns(ctx, owner, name, branch)
	if err != nil {
		return nil, fmt.Errorf("fetching workflow runs: %w", err)
	}
	metrics = append(metrics, buildSuccessRatio(runs), buildTime(runs))

	m, err = s.collectDeployFrequency(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("collecting deploy frequency: %w", err)
	}
	metrics = append(metrics, m...)

	m, err = s.collectStalePRs(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("collecting stale PRs: %w", err)
	}
	metrics = append(metrics, m...)

	return metrics, nil
}

// parseRepoURL extracts owner and repo name from a GitHub URL.
// Supports formats like:
//   - github.com/owner/repo
//   - https://github.com/owner/repo
//   - git@github.com:owner/repo
func parseRepoURL(raw string) (owner, name string, err error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")

	// Handle SSH format: git@github.com:owner/repo
	if strings.HasPrefix(raw, "git@") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format")
		}
		parts = strings.SplitN(parts[1], "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format")
		}
		return parts[0], parts[1], nil
	}

	// Handle HTTPS/host format: github.com/owner/repo or https://github.com/owner/repo
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	host := u.Hostname()
	if host == "" {
		// Might be bare "owner/repo" or "github.com/owner/repo"
		parts := strings.SplitN(raw, "/", 3)
		if len(parts) == 3 && parts[0] == "github.com" {
			return parts[1], parts[2], nil
		}
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
		return "", "", fmt.Errorf("cannot parse repo URL: %s", raw)
	}

	if host != "github.com" {
		return "", "", fmt.Errorf("unsupported host %q, only github.com is supported", host)
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot parse repo URL: %s", raw)
	}

	return parts[0], parts[1], nil
}
