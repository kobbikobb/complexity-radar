package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// GitTree represents a GitHub git tree response.
type GitTree struct {
	Tree []GitTreeEntry `json:"tree"`
}

// GitTreeEntry represents a single entry in a git tree.
type GitTreeEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

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
		model.MetricTypeSecurityCritical,
		model.MetricTypeSecurityHigh,
		model.MetricTypeSecurityMedium,
		model.MetricTypeSecurityLow,
		model.MetricTypeBuildSuccessRatio,
		model.MetricTypeBuildTime,
		model.MetricTypeDeployFrequency,
		model.MetricTypeStalePRs,
		model.MetricTypeDependencyCount,
		model.MetricTypeDecisionDensity,
		model.MetricTypeK8sDeployments,
		model.MetricTypeContainerImages,
		model.MetricTypeDeployTargets,
		model.MetricTypeCICDComplexity,
	}
}

func (s *Source) Collect(ctx context.Context, repo model.Repository) ([]model.SourceMetric, error) {
	owner, name, err := parseRepoURL(repo.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing repo URL %q: %w", repo.URL, err)
	}

	branch := repo.Branch
	if branch == "" {
		branch = "main"
	}

	var metrics []model.SourceMetric

	// Fetch workflow runs once and reuse for both build metrics.
	runs, err := s.fetchWorkflowRuns(ctx, owner, name, branch)
	if err != nil {
		return nil, fmt.Errorf("fetching workflow runs: %w", err)
	}
	metrics = append(metrics, buildSuccessRatio(runs), buildTime(runs))

	m, err := s.collectDeployFrequency(ctx, owner, name, repo.GitopsRepoURL, repo.DeployDetection, repo.IncludePrereleases, repo.TagPrefix)
	if err != nil {
		return nil, fmt.Errorf("collecting deploy frequency: %w", err)
	}
	metrics = append(metrics, m...)

	m, err = s.collectStalePRs(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("collecting stale PRs: %w", err)
	}
	metrics = append(metrics, m...)

	// File-based metrics: fetch git tree once, pass to all collectors.
	// If the tree can't be fetched (e.g., empty repo), use an empty tree
	// so file-based metrics gracefully return 0.
	tree, _ := s.fetchGitTree(ctx, owner, name, branch)
	if tree == nil {
		tree = &GitTree{}
	}

	// Service count normalizes size-scaling metrics so a monorepo isn't
	// penalized for holding many services in one repo (health over size).
	services := countServices(tree)

	m, err = s.collectSecurityVulnerabilities(ctx, owner, name, services)
	if err != nil {
		return nil, fmt.Errorf("collecting security vulnerabilities: %w", err)
	}
	metrics = append(metrics, m...)

	m, err = collectDependencyCount(ctx, s.client, owner, name, branch, tree)
	if err != nil {
		return nil, fmt.Errorf("collecting dependency count: %w", err)
	}
	metrics = append(metrics, m...)

	metrics = append(metrics, collectDecisionDensity(ctx, s.client, owner, name, branch, tree)...)

	// Fetch workflow file contents once for both cicd and deploy_targets collectors.
	workflowContents := s.fetchWorkflowContents(ctx, owner, name, branch, tree)

	metrics = append(metrics, collectK8sDeployments(tree, services)...)
	metrics = append(metrics, collectContainerImages(ctx, s.client, owner, name, branch, tree, services)...)
	metrics = append(metrics, collectDeployTargets(ctx, s.client, owner, name, branch, workflowContents)...)
	metrics = append(metrics, collectCICDComplexity(workflowContents, ctx, s.client, owner, name, branch)...)

	return metrics, nil
}

// countServices counts distinct directories holding a recognized dependency
// manifest — the repo's deployable units. Returns at least 1 so callers can
// divide safely. Mirrors the service definition used by collectDependencyCount.
func countServices(tree *GitTree) int {
	dirs := map[string]bool{}
	for _, entry := range tree.Tree {
		if isVendored(entry.Path) {
			continue
		}
		fileName := entry.Path
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 {
			fileName = fileName[idx+1:]
		}
		if _, ok := matchManifest(fileName); ok {
			dirs[path.Dir(entry.Path)] = true
		}
	}
	return max(1, len(dirs))
}

func (s *Source) fetchGitTree(ctx context.Context, owner, name, branch string) (*GitTree, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/git/trees/%s", owner, name, branch)
	data, err := s.client.GetWithParams(ctx, endpoint, map[string]string{"recursive": "1"})
	if err != nil {
		return nil, err
	}

	var tree GitTree
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("parsing git tree: %w", err)
	}

	return &tree, nil
}

// parseRepoURL extracts owner and repo name from a GitHub URL.
// Supports formats like:
//   - github.com/owner/repo
//   - https://github.com/owner/repo
//   - git@github.com:owner/repo
func (s *Source) fetchWorkflowContents(ctx context.Context, owner, name, branch string, tree *GitTree) map[string]string {
	contents := make(map[string]string)
	for _, entry := range tree.Tree {
		if strings.HasPrefix(entry.Path, ".github/workflows/") &&
			(strings.HasSuffix(entry.Path, ".yml") || strings.HasSuffix(entry.Path, ".yaml")) {
			content, err := s.client.GetFileContent(ctx, owner, name, entry.Path, branch)
			if err != nil {
				continue
			}
			contents[entry.Path] = content
		}
	}
	return contents
}

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
