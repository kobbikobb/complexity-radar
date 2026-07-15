package github

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
)

type mockClient struct {
	responses    map[string]json.RawMessage
	errors       map[string]error
	fileContents map[string]string
	fileReads    []string
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
	m.fileReads = append(m.fileReads, path)
	if content, ok := m.fileContents[path]; ok {
		return content, nil
	}
	return "", fmt.Errorf("file not found: %s", path)
}

func findMetric(metrics []model.SourceMetric, typ model.MetricTypeName) (model.SourceMetric, bool) {
	for _, m := range metrics {
		if m.Type == typ {
			return m, true
		}
	}
	return model.SourceMetric{}, false
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

// makeTreeJSON creates a JSON git tree response for the given paths.
func makeTreeJSON(paths []string) json.RawMessage {
	type entry struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
		Type string `json:"type"`
	}
	var tree []entry
	for _, p := range paths {
		tree = append(tree, entry{Path: p, Size: 100, Type: "blob"})
	}
	data, _ := json.Marshal(struct {
		Tree []entry `json:"tree"`
	}{Tree: tree})
	return data
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
		{State: "open", Severity: "high"},
		{State: "open", Severity: "critical"},
		{State: "dismissed", Severity: "low"},
	}
	data, _ := json.Marshal(alerts)

	responses := defaultResponses()
	responses["/repos/org/repo/dependabot/alerts"] = data
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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
	// critical=1.0 + high=0.7 = 1.7 (dismissed not counted)
	want := 1.7
	if m.Value != want {
		t.Errorf("security vulnerabilities = %v, want %v", m.Value, want)
	}
}

func TestCollectSecurityVulnerabilitiesDownweightsDevScope(t *testing.T) {
	devScope := &struct {
		Scope string `json:"scope"`
	}{Scope: "development"}
	alerts := []Vulnerability{
		{State: "open", Severity: "high"},
		{State: "open", Severity: "high", Dependency: devScope},
	}
	data, _ := json.Marshal(alerts)

	responses := defaultResponses()
	responses["/repos/org/repo/dependabot/alerts"] = data
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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
	// runtime high=0.7 + dev high=0.7*0.3=0.21 = 0.91
	want := 0.91
	if math.Abs(m.Value-want) > 1e-9 {
		t.Errorf("security vulnerabilities = %v, want %v", m.Value, want)
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
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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

func TestCollectDeployFrequencyNoReleasesIsNoData(t *testing.T) {
	responses := defaultResponses()
	responses["/repos/org/repo/releases"] = []byte("[]")
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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
	if m.Value != noDataValue {
		t.Errorf("deploy frequency = %v, want noDataValue %v", m.Value, noDataValue)
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
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

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
	client.responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"package.json"})

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
	client.responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"go.mod"})

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

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"Dockerfile"})

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

func TestCollectContainerImagesNoTag(t *testing.T) {
	dockerfile := `FROM scratch
COPY binary /app`

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"Dockerfile"})

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
	if m.Value != 1 {
		t.Errorf("container images = %v, want 1 (FROM scratch should count)", m.Value)
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

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{".github/workflows/ci.yml"})

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
	// jobs(10) + uses(2) + name(2) + if(3) + workflows(8) + secrets(2) = 27
	if m.Value != 27 {
		t.Errorf("CI/CD complexity = %v, want 27", m.Value)
	}
}

func TestCollectDeployTargets(t *testing.T) {
	// Arrange
	prod := `name: Prod
on: push
jobs:
  deploy:
    environment: production`
	staging := `name: Staging
on: push
jobs:
  deploy:
    environment: staging`

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{
		".github/workflows/prod.yml",
		".github/workflows/staging.yml",
	})

	client := &mockClient{
		responses: responses,
		fileContents: map[string]string{
			".github/workflows/prod.yml":    prod,
			".github/workflows/staging.yml": staging,
		},
	}

	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	// Act
	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Assert
	m, ok := findMetric(metrics, model.MetricTypeDeployTargets)
	if !ok {
		t.Fatal("missing deploy_targets metric")
	}
	if m.Value != 2 {
		t.Errorf("deploy targets = %v, want 2 (distinct count, not averaged over 2 workflows)", m.Value)
	}
}

func TestCollectK8sDeployments(t *testing.T) {
	// Arrange
	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{
		"k8s/deployment.yaml",
		"k8s/service.yaml",
		"deploy/ingress.yml",
		"src/main.go",
	})

	client := &mockClient{responses: responses}

	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	// Act
	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Assert
	m, ok := findMetric(metrics, model.MetricTypeK8sDeployments)
	if !ok {
		t.Fatal("missing k8s_deployments metric")
	}
	if m.Value != 3 {
		t.Errorf("k8s deployments = %v, want 3 (distinct count, not averaged over 2 subdirs)", m.Value)
	}
}

func TestComplexity(t *testing.T) {
	tests := []struct {
		name string
		code string
		want int
	}{
		{"should be base 1 for empty file", "", 1},
		{"should count python if/elif/for/except", "if a:\n    pass\nelif b:\n    pass\nfor x in y:\n    pass\nexcept E:\n    pass", 5},
		{"should count go if/for/case/&&/||", "if a && b {\n}\nfor {\n}\nswitch {\ncase 1:\n}\nif c || d {\n}", 7},
		{"should count a ternary once", "const x = a ? b : c;", 2},
		{"should count nullish coalescing once", "const y = a ?? b;", 2},
		{"should count ts ternary and nullish", "const x = a ? b : c;\nconst y = a ?? b;\nif (p) {}", 4},
		{"should count ruby when/while", "case x\nwhen 1\nend\nwhile y\nend", 4},
		{"should not count keywords inside identifiers", "notify(user)\nverify(x)\nclassifier()", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := complexity(tt.code)

			if got != tt.want {
				t.Errorf("complexity(%q) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		q      float64
		want   float64
	}{
		{"should return 0 for an empty slice", nil, 0.95, 0},
		{"should return the value for a single element", []float64{7}, 0.95, 7},
		{"should interpolate p95 toward the top for two elements", []float64{2, 10}, 0.95, 9.6},
		{"should return the median for odd-length input", []float64{1, 2, 3}, 0.5, 2},
		{"should compute p95 for ten elements", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0.95, 9.55},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.sorted, tt.q)

			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("percentile(%v, %v) = %v, want %v", tt.sorted, tt.q, got, tt.want)
			}
		})
	}
}

func TestServiceKey(t *testing.T) {
	manifestDirs := map[string]bool{"services/api": true, ".": true}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"should return the nearest ancestor manifest dir", "services/api/handler.go", "services/api"},
		{"should return the nearest manifest dir for a deep path", "services/api/internal/db/store.go", "services/api"},
		{"should return the root bucket for a top-level file", "main.go", "."},
		{"should return the root bucket for a script dir", "scripts/tool.py", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceKey(tt.path, manifestDirs)

			if got != tt.want {
				t.Errorf("serviceKey(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestServiceKeyNoManifestIsRootBucket(t *testing.T) {
	got := serviceKey("a/b/c.go", map[string]bool{})

	if got != "" {
		t.Errorf("serviceKey with no manifest dirs = %q, want \"\"", got)
	}
}

func TestCollectDecisionDensityCapsReadsLargestFirst(t *testing.T) {
	// Arrange
	const candidates = maxComplexityFileReads + 50
	paths := make([]string, candidates)
	contents := map[string]string{}
	type entry struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
		Type string `json:"type"`
	}
	var tree []entry
	for i := 0; i < candidates; i++ {
		p := fmt.Sprintf("f%04d.go", i)
		paths[i] = p
		contents[p] = "package p"
		tree = append(tree, entry{Path: p, Size: int64(i), Type: "blob"})
	}
	treeJSON, _ := json.Marshal(struct {
		Tree []entry `json:"tree"`
	}{Tree: tree})

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = treeJSON
	client := &mockClient{responses: responses, fileContents: contents}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	// Act
	_, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Assert
	var goReads []string
	for _, p := range client.fileReads {
		if strings.HasSuffix(p, ".go") {
			goReads = append(goReads, p)
		}
	}
	if len(goReads) != maxComplexityFileReads {
		t.Fatalf("candidate file reads = %d, want %d", len(goReads), maxComplexityFileReads)
	}
	if goReads[0] != fmt.Sprintf("f%04d.go", candidates-1) {
		t.Errorf("first read = %q, want largest file %q", goReads[0], fmt.Sprintf("f%04d.go", candidates-1))
	}
}

func TestCollectDecisionDensity(t *testing.T) {
	// Arrange
	simple := "package p\nfunc a() {}"
	complex := "package p\nfunc b() {\n  if x { }\n  for { }\n  if y && z { }\n}"

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"go.mod", "a.go", "b.go", "vendor/dep.go"})
	client := &mockClient{
		responses: responses,
		fileContents: map[string]string{
			"a.go":          simple,
			"b.go":          complex,
			"vendor/dep.go": complex,
		},
	}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	// Act
	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Assert
	m, ok := findMetric(metrics, model.MetricTypeDecisionDensity)
	if !ok {
		t.Fatal("missing decision_density metric")
	}
	// one service (root, go.mod); densities are tokens per 100 non-blank lines; p95 interpolates toward the larger.
	dSimple := float64(complexity(simple)) / float64(nonBlankLineCount(simple)) * 100
	dComplex := float64(complexity(complex)) / float64(nonBlankLineCount(complex)) * 100
	want := percentile([]float64{dSimple, dComplex}, 0.95)
	if math.Abs(m.Value-want) > 1e-9 {
		t.Errorf("decision density = %v, want %v", m.Value, want)
	}
	for _, p := range client.fileReads {
		if p == "vendor/dep.go" {
			t.Error("vendored file was read, should be excluded")
		}
	}
}

func TestCollectDecisionDensityNoSource(t *testing.T) {
	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"README.md", "docs/guide.txt"})

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeDecisionDensity)
	if !ok {
		t.Fatal("missing decision_density metric")
	}
	if m.Value != noDataValue {
		t.Errorf("decision density = %v, want %v", m.Value, noDataValue)
	}
}

func TestIsGenerated(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"should exclude a dist path segment", "services/web/dist/app.ts", true},
		{"should exclude a .min.js filename", "src/vendor/foo.min.js", true},
		{"should not exclude a normal source file", "src/foo.ts", false},
		{"should match segments only, not substrings", "my-dist-thing/foo.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGenerated(tt.path)

			if got != tt.want {
				t.Errorf("isGenerated(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestNonBlankLineCount(t *testing.T) {
	// Arrange
	s := "package p\n\n   \n\tif a {}\ncode"

	// Act
	got := nonBlankLineCount(s)

	// Assert
	if got != 3 {
		t.Errorf("nonBlankLineCount(%q) = %d, want 3 (blank and whitespace-only lines excluded)", s, got)
	}
}

func decisionDensity(content string) float64 {
	return float64(complexity(content)) / float64(nonBlankLineCount(content)) * 100
}

func TestDecisionDensityValue(t *testing.T) {
	// Arrange
	content := "if a {}\nif b {}\nx := 1\ny := 2" // 2 tokens (+1) over 4 non-blank lines

	// Act
	got := decisionDensity(content)

	// Assert
	if got != 75 {
		t.Errorf("decisionDensity = %v, want 75 (3 decision points per 4 lines ×100/4)", got)
	}
}

func TestDecisionDensityIsSizeNeutral(t *testing.T) {
	// Arrange
	denseSmall := "if a {}\nif b {}"
	largeSparse := "package p\n" + strings.Repeat("x := 1\n", 100) + "if a {}"

	// Act
	dense := decisionDensity(denseSmall)
	sparse := decisionDensity(largeSparse)

	// Assert
	if dense <= sparse {
		t.Errorf("dense small file should have higher density than large sparse file: dense=%v sparse=%v", dense, sparse)
	}
}

func TestLongestLineLen(t *testing.T) {
	// Arrange
	s := "ab\ncdef\ng"

	// Act
	got := longestLineLen(s)

	// Assert
	if got != 4 {
		t.Errorf("longestLineLen(%q) = %d, want 4", s, got)
	}
}

func TestCollectDecisionDensitySkipsMinifiedContent(t *testing.T) {
	// Arrange
	normal := "package p\nif a {}"
	minified := "package p\n" + strings.Repeat("if x ", 2000)

	responses := defaultResponses()
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON([]string{"go.mod", "normal.go", "huge.go"})
	client := &mockClient{
		responses: responses,
		fileContents: map[string]string{
			"normal.go": normal,
			"huge.go":   minified,
		},
	}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main"}

	// Act
	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Assert
	m, ok := findMetric(metrics, model.MetricTypeDecisionDensity)
	if !ok {
		t.Fatal("missing decision_density metric")
	}
	// normal.go: complexity 2 over 2 non-blank lines → 100 per 100 loc; minified file excluded.
	if m.Value != 100 {
		t.Errorf("decision density = %v, want 100 (minified file excluded, only normal.go counted)", m.Value)
	}
}

func TestCollectGitopsDeployFrequency(t *testing.T) {
	commits := []map[string]string{
		{"sha": "abc123"},
		{"sha": "def456"},
	}
	data, _ := json.Marshal(commits)

	responses := defaultResponses()
	responses["/repos/org/gitops/commits"] = data
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{
		URL:           "github.com/org/repo",
		Branch:        "main",
		GitopsRepoURL: "github.com/org/gitops",
	}

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	m, ok := findMetric(metrics, model.MetricTypeDeployFrequency)
	if !ok {
		t.Fatal("missing deploy_frequency metric")
	}
	if m.Value != 2 {
		t.Errorf("deploy frequency = %v, want 2", m.Value)
	}
}

func TestCollectDeployFrequencyReleasesMethod(t *testing.T) {
	now := time.Now()
	releases := []Release{
		{TagName: "v1.0.0", PublishedAt: now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)},
	}
	data, _ := json.Marshal(releases)

	responses := defaultResponses()
	responses["/repos/org/repo/releases"] = data
	responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

	client := &mockClient{responses: responses}
	src := NewSourceWithClient(client)
	repo := model.Repository{URL: "github.com/org/repo", Branch: "main", DeployDetection: config.DeployDetectionReleases}

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

func TestCollectDeployFrequencyReleaseOptions(t *testing.T) {
	now := time.Now()
	recent := now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	releases := []Release{
		{TagName: "v1.0.0", PublishedAt: recent},
		{TagName: "v1.1.0-rc1", PublishedAt: recent, Prerelease: true},
		{TagName: "app-2.0.0", PublishedAt: recent},
	}
	data, _ := json.Marshal(releases)

	collect := func(includePrereleases bool, tagPrefix string) float64 {
		responses := defaultResponses()
		responses["/repos/org/repo/releases"] = data
		responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

		client := &mockClient{responses: responses}
		src := NewSourceWithClient(client)
		repo := model.Repository{
			URL:                "github.com/org/repo",
			Branch:             "main",
			DeployDetection:    config.DeployDetectionReleases,
			IncludePrereleases: includePrereleases,
			TagPrefix:          tagPrefix,
		}

		metrics, err := src.Collect(context.Background(), repo)
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
		m, ok := findMetric(metrics, model.MetricTypeDeployFrequency)
		if !ok {
			t.Fatal("missing deploy_frequency metric")
		}
		return m.Value
	}

	if got := collect(false, ""); got != 2 {
		t.Errorf("default excludes prerelease: got %v, want 2", got)
	}
	if got := collect(true, ""); got != 3 {
		t.Errorf("includes prerelease: got %v, want 3", got)
	}
	if got := collect(true, "v"); got != 2 {
		t.Errorf("tag prefix filters non-matching: got %v, want 2", got)
	}
}

func TestCollectDeployFrequencyTags(t *testing.T) {
	now := time.Now()
	recent := now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	old := now.Add(-10 * 24 * time.Hour).Format(time.RFC3339)

	tags := []Tag{
		{Name: "promote/1"}, {Name: "promote/2"}, {Name: "release/1"},
	}
	tags[0].Commit.SHA = "sha1"
	tags[1].Commit.SHA = "sha2"
	tags[2].Commit.SHA = "sha3"
	tagsData, _ := json.Marshal(tags)

	commitJSON := func(date string) json.RawMessage {
		return []byte(fmt.Sprintf(`{"commit":{"committer":{"date":%q}}}`, date))
	}

	collect := func(tagPrefix string) float64 {
		responses := defaultResponses()
		responses["/repos/org/repo/tags"] = tagsData
		responses["/repos/org/repo/commits/sha1"] = commitJSON(recent)
		responses["/repos/org/repo/commits/sha2"] = commitJSON(old)
		responses["/repos/org/repo/commits/sha3"] = commitJSON(recent)
		responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(nil)

		client := &mockClient{responses: responses}
		src := NewSourceWithClient(client)
		repo := model.Repository{URL: "github.com/org/repo", Branch: "main", DeployDetection: config.DeployDetectionTags, TagPrefix: tagPrefix}

		metrics, err := src.Collect(context.Background(), repo)
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
		m, ok := findMetric(metrics, model.MetricTypeDeployFrequency)
		if !ok {
			t.Fatal("missing deploy_frequency metric")
		}
		return m.Value
	}

	if got := collect("promote/"); got != 1 {
		t.Errorf("prefix promote/ counts recent matching tags: got %v, want 1", got)
	}
	if got := collect(""); got != 2 {
		t.Errorf("empty prefix counts all recent tags: got %v, want 2", got)
	}
}

func TestParsePackageJSON(t *testing.T) {
	content := `{"dependencies": {"a": "1", "b": "2"}, "devDependencies": {"c": "3"}}`
	names := parsePackageJSON(content)
	if len(names) != 3 {
		t.Errorf("count = %v, want 3", len(names))
	}
}

func TestParseGoMod(t *testing.T) {
	content := `require (
	github.com/foo v1.0.0
	github.com/bar v2.0.0
)`
	names := parseGoMod(content)
	if len(names) != 2 {
		t.Errorf("count = %v, want 2", len(names))
	}
}

func TestParsePomXML(t *testing.T) {
	// Arrange
	content := `<project>
<dependencies>
<dependency><groupId>org.foo</groupId><artifactId>bar</artifactId></dependency>
<dependency><groupId>com.baz</groupId><artifactId>qux</artifactId></dependency>
</dependencies>
</project>`

	// Act
	names := parsePomXML(content)

	// Assert
	want := []string{"org.foo:bar", "com.baz:qux"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	content := `flask==2.0
requests>=2.25
# comment
-r base.txt`
	names := parseRequirementsTxt(content)
	if len(names) != 2 {
		t.Errorf("count = %v, want 2 (flask, requests)", len(names))
	}
}

func TestParseCargoToml(t *testing.T) {
	content := `[dependencies]
serde = "1.0"
tokio = { version = "1.0", features = ["full"] }

[dev-dependencies]
assert_cmd = "2.0"

[build-dependencies]
cc = "1.0"`
	names := parseCargoToml(content)
	if len(names) != 3 {
		t.Errorf("count = %v, want 3 (serde, tokio, assert_cmd)", len(names))
	}
}

func TestParseGemfile(t *testing.T) {
	content := `source 'https://rubygems.org'
gem 'rails', '~> 7.0'
gem 'puma', '~> 5.0'
group :test do
  gem 'rspec', '~> 3.0'
end`
	names := parseGemfile(content)
	if len(names) != 3 {
		t.Errorf("count = %v, want 3", len(names))
	}
}
