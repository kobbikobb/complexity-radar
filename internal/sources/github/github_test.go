package github

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

type mockClient struct {
	responses    map[string]json.RawMessage
	errors       map[string]error
	fileContents map[string]string
}

func (m *mockClient) Get(_ context.Context, endpoint string) (json.RawMessage, error) {
	if err, ok := m.errors[endpoint]; ok {
		return nil, err
	}
	if resp, ok := m.responses[endpoint]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("no mock response for %s", endpoint)
}

func (m *mockClient) GetWithParams(_ context.Context, endpoint string, _ map[string]string) (json.RawMessage, error) {
	if err, ok := m.errors[endpoint]; ok {
		return nil, err
	}
	if resp, ok := m.responses[endpoint]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("no mock response for %s", endpoint)
}

func (m *mockClient) GetPaginated(_ context.Context, endpoint string, _ map[string]string, _ int) (json.RawMessage, error) {
	if err, ok := m.errors[endpoint]; ok {
		return nil, err
	}
	if resp, ok := m.responses[endpoint]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("no mock response for %s", endpoint)
}

func (m *mockClient) GetFileContent(_ context.Context, _, _, path, _ string) (string, error) {
	if content, ok := m.fileContents[path]; ok {
		return content, nil
	}
	return "", fmt.Errorf("file not found: %s", path)
}

func findMetric(metrics []sources.SourceMetric, typ model.MetricTypeName) (sources.SourceMetric, bool) {
	for _, m := range metrics {
		if m.Type == typ {
			return m, true
		}
	}
	return sources.SourceMetric{}, false
}

// defaultResponses returns mock responses for all endpoints with empty/default data.
func defaultResponses() map[string]json.RawMessage {
	return map[string]json.RawMessage{
		"/repos/org/repo/dependabot/alerts": []byte(`[]`),
		"/repos/org/repo/actions/runs":      []byte(`{"workflow_runs": []}`),
		"/repos/org/repo/releases":          []byte(`[]`),
		"/repos/org/repo/pulls":             []byte(`[]`),
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"github.com/org/my-project", "org/my-project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, name, err := parseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("parseRepoURL(%q) error: %v", tt.input, err)
			}
			got := owner + "/" + name
			if got != tt.want {
				t.Errorf("parseRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRepoURLErrors(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"git@github.com:owner", "incomplete SSH"},
		{"githb.com/owner/repo", "wrong host"},
		{"owner", "single segment"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, _, err := parseRepoURL(tt.input)
			if err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
		})
	}
}

func TestCollectSecurityVulnerabilities(t *testing.T) {
	alerts := []Vulnerability{
		{State: "open"},
		{State: "open"},
		{State: "dismissed"},
	}
	data, _ := json.Marshal(alerts)

	responses := defaultResponses()
	responses["/repos/org/repo/dependabot/alerts"] = data

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeSecurityVulnerabilities)
	if !ok {
		t.Fatal("missing security_vulnerabilities metric")
	}
	if m.Value != 2 {
		t.Errorf("security vulnerabilities = %v, want 2", m.Value)
	}
}

func TestCollectBuildMetrics(t *testing.T) {
	now := time.Now()
	runs := []WorkflowRun{
		{Conclusion: "success", RunStartedAt: now.Add(-10 * time.Minute).Format(time.RFC3339), UpdatedAt: now.Add(-8 * time.Minute).Format(time.RFC3339)},
		{Conclusion: "success", RunStartedAt: now.Add(-20 * time.Minute).Format(time.RFC3339), UpdatedAt: now.Add(-15 * time.Minute).Format(time.RFC3339)},
		{Conclusion: "failure", RunStartedAt: now.Add(-30 * time.Minute).Format(time.RFC3339), UpdatedAt: now.Add(-25 * time.Minute).Format(time.RFC3339)},
	}
	resp := struct {
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}{WorkflowRuns: runs}
	data, _ := json.Marshal(resp)

	responses := defaultResponses()
	responses["/repos/org/repo/actions/runs"] = data

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	sr, ok := findMetric(metrics, model.MetricTypeBuildSuccessRatio)
	if !ok {
		t.Fatal("missing build_success_ratio metric")
	}
	wantRatio := 2.0 / 3.0
	if sr.Value != wantRatio {
		t.Errorf("build success ratio = %v, want %v", sr.Value, wantRatio)
	}

	bt, ok := findMetric(metrics, model.MetricTypeBuildTime)
	if !ok {
		t.Fatal("missing build_time metric")
	}
	// avg of 2 min + 5 min + 5 min = 4 min = 240s
	if bt.Value != 240 {
		t.Errorf("build time = %v, want 240", bt.Value)
	}
}

func TestCollectDeployFrequency(t *testing.T) {
	now := time.Now()
	releases := []Release{
		{TagName: "v1.0.0", PublishedAt: now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)},
		{TagName: "v1.1.0", PublishedAt: now.Add(-10 * 24 * time.Hour).Format(time.RFC3339)},
		{TagName: "v0.9.0-draft", PublishedAt: now.Add(-1 * 24 * time.Hour).Format(time.RFC3339), Draft: true},
		{TagName: "v1.2.0-rc1", PublishedAt: now.Add(-1 * 24 * time.Hour).Format(time.RFC3339), Prerelease: true},
	}
	data, _ := json.Marshal(releases)

	responses := defaultResponses()
	responses["/repos/org/repo/releases"] = data

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeDeployFrequency)
	if !ok {
		t.Fatal("missing deploy_frequency metric")
	}
	if m.Value != 1 {
		t.Errorf("deploy frequency = %v, want 1", m.Value)
	}
}

func TestCollectStalePRs(t *testing.T) {
	now := time.Now()
	prs := []PullRequest{
		{State: "open", UpdatedAt: now.Add(-20 * 24 * time.Hour).Format(time.RFC3339)},
		{State: "open", UpdatedAt: now.Add(-5 * 24 * time.Hour).Format(time.RFC3339)},
		{State: "closed", UpdatedAt: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)},
	}
	data, _ := json.Marshal(prs)

	responses := defaultResponses()
	responses["/repos/org/repo/pulls"] = data

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeStalePRs)
	if !ok {
		t.Fatal("missing stale_prs metric")
	}
	if m.Value != 1 {
		t.Errorf("stale PRs = %v, want 1", m.Value)
	}
}

func TestCollectEmptyWorkflowRuns(t *testing.T) {
	responses := defaultResponses()
	responses["/repos/org/repo/actions/runs"] = []byte(`{"workflow_runs": []}`)

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeBuildSuccessRatio)
	if !ok {
		t.Fatal("missing build_success_ratio metric")
	}
	if m.Value != 0 {
		t.Errorf("build success ratio = %v, want 0", m.Value)
	}
}

func TestSupportedMetrics(t *testing.T) {
	src := NewSource()
	metrics := src.SupportedMetrics()

	expected := []model.MetricTypeName{
		model.MetricTypeSecurityVulnerabilities,
		model.MetricTypeBuildSuccessRatio,
		model.MetricTypeBuildTime,
		model.MetricTypeDeployFrequency,
		model.MetricTypeStalePRs,
		model.MetricTypeDependencyCount,
		model.MetricTypeK8sDeployments,
		model.MetricTypeContainerImages,
		model.MetricTypeDeployTargets,
		model.MetricTypeCICDComplexity,
	}

	if len(metrics) != len(expected) {
		t.Fatalf("SupportedMetrics() returned %d metrics, want %d", len(metrics), len(expected))
	}

	for i, m := range metrics {
		if m != expected[i] {
			t.Errorf("SupportedMetrics()[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestCollectDependencyCount(t *testing.T) {
	packageJSON := `{
		"dependencies": {"react": "^18.0.0", "next": "^13.0.0"},
		"devDependencies": {"typescript": "^5.0.0"}
	}`

	client := &mockClient{
		responses:    defaultResponses(),
		fileContents: map[string]string{"package.json": packageJSON},
	}

	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeDependencyCount)
	if !ok {
		t.Fatal("missing dependency_count metric")
	}
	if m.Value != 3 {
		t.Errorf("dependency count = %v, want 3", m.Value)
	}
}

func TestCollectDependencyCountGoMod(t *testing.T) {
	goMod := `module github.com/org/repo

go 1.21

require (
	github.com/foo/bar v1.0.0
	github.com/baz/qux v2.0.0
)`

	client := &mockClient{
		responses:    defaultResponses(),
		fileContents: map[string]string{"go.mod": goMod},
	}

	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeDependencyCount)
	if !ok {
		t.Fatal("missing dependency_count metric")
	}
	if m.Value != 2 {
		t.Errorf("dependency count = %v, want 2", m.Value)
	}
}

func TestCollectContainerImages(t *testing.T) {
	dockerfile := `FROM nginx:1.21-alpine
COPY . /usr/share/nginx/html
FROM node:18-alpine AS builder`

	treeData := []struct {
		Path string `json:"path"`
	}{
		{Path: "Dockerfile"},
	}
	treeJSON, _ := json.Marshal(struct {
		Tree []struct {
			Path string `json:"path"`
		} `json:"tree"`
	}{Tree: treeData})

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = treeJSON

	client := &mockClient{
		responses:    responses,
		fileContents: map[string]string{"Dockerfile": dockerfile},
	}

	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeContainerImages)
	if !ok {
		t.Fatal("missing container_images metric")
	}
	if m.Value != 2 {
		t.Errorf("container images = %v, want 2", m.Value)
	}
}

func TestCollectCICDComplexity(t *testing.T) {
	workflow := `name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run tests
        run: make test
      - if: success()
        name: Deploy
        uses: ./.github/workflows/deploy.yml
        secrets: inherit`

	treeData := []struct {
		Path string `json:"path"`
	}{
		{Path: ".github/workflows/ci.yml"},
	}
	treeJSON, _ := json.Marshal(struct {
		Tree []struct {
			Path string `json:"path"`
		} `json:"tree"`
	}{Tree: treeData})

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = treeJSON

	client := &mockClient{
		responses:    responses,
		fileContents: map[string]string{".github/workflows/ci.yml": workflow},
	}

	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeCICDComplexity)
	if !ok {
		t.Fatal("missing ci_cd_complexity metric")
	}
	// Should have non-zero complexity score
	if m.Value <= 0 {
		t.Errorf("CI/CD complexity = %v, want > 0", m.Value)
	}
}
