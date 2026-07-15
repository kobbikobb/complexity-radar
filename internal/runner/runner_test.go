package runner

import (
	"context"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

type mockSource struct {
	metrics []model.SourceMetric
	err     error
}

func (m *mockSource) Name() string { return "mock" }

func (m *mockSource) Collect(ctx context.Context, repo model.Repository) ([]model.SourceMetric, error) {
	return m.metrics, m.err
}

func (m *mockSource) SupportedMetrics() []model.MetricTypeName {
	return []model.MetricTypeName{
		model.MetricTypeDeployFrequency,
		model.MetricTypeSecurityVulnerabilities,
	}
}

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func seedProjectAndRepo(t *testing.T, s *store.Store, projectName, repoURL, branch string) *model.Project {
	t.Helper()
	p := &model.Project{Name: projectName}
	if err := s.CreateProject(p); err != nil {
		t.Fatal(err)
	}
	repo := &model.Repository{ProjectID: p.ID, URL: repoURL, Branch: branch}
	if err := s.CreateRepository(repo); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunnerRun(t *testing.T) {
	s := setupTestStore(t)
	seedProjectAndRepo(t, s, "test-project", "github.com/org/repo", "main")

	src := &mockSource{
		metrics: []model.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 5.0},
			{Type: model.MetricTypeSecurityVulnerabilities, Value: 2.0},
		},
	}

	r, err := NewFromStore(s, "test-project", src)
	if err != nil {
		t.Fatal(err)
	}

	if r.Project().Name != "test-project" {
		t.Errorf("expected project name 'test-project', got %q", r.Project().Name)
	}

	if len(r.Repositories()) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(r.Repositories()))
	}

	result, err := r.Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got %q", result.Project.Name)
	}

	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 repository result, got %d", len(result.Repositories))
	}

	repoResult := result.Repositories[0]
	if repoResult.Repository.URL != "github.com/org/repo" {
		t.Errorf("expected repo URL 'github.com/org/repo', got %q", repoResult.Repository.URL)
	}

	if len(repoResult.Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(repoResult.Metrics))
	}

	if repoResult.OverallScore == 0 {
		t.Error("expected non-zero overall score")
	}
}

func TestRunnerMultipleRepos(t *testing.T) {
	s := setupTestStore(t)
	seedProjectAndRepo(t, s, "multi-repo", "github.com/org/repo1", "main")
	repo2 := &model.Repository{ProjectID: 0, URL: "github.com/org/repo2", Branch: "develop"}
	// Need to find project first
	p, err := s.GetProjectByName("multi-repo")
	if err != nil {
		t.Fatal(err)
	}
	repo2.ProjectID = p.ID
	if err := s.CreateRepository(repo2); err != nil {
		t.Fatal(err)
	}

	src := &mockSource{
		metrics: []model.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 3.0},
		},
	}

	r, err := NewFromStore(s, "multi-repo", src)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Repositories()) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(r.Repositories()))
	}

	result, err := r.Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Repositories) != 2 {
		t.Fatalf("expected 2 repository results, got %d", len(result.Repositories))
	}
}

func TestRunnerSourceError(t *testing.T) {
	s := setupTestStore(t)
	seedProjectAndRepo(t, s, "error-project", "github.com/org/repo", "main")

	src := &mockSource{err: context.DeadlineExceeded}

	r, err := NewFromStore(s, "error-project", src)
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 repository result, got %d", len(result.Repositories))
	}

	if len(result.Repositories[0].Errors) == 0 {
		t.Error("expected errors from failed collection")
	}
}

func TestRunnerNoProjectConfigured(t *testing.T) {
	s := setupTestStore(t)

	src := &mockSource{}
	_, err := NewFromStore(s, "", src)
	if err == nil {
		t.Fatal("expected error when no project configured")
	}
}

func TestRunnerProgressCallback(t *testing.T) {
	s := setupTestStore(t)
	seedProjectAndRepo(t, s, "progress-project", "github.com/org/repo", "main")

	src := &mockSource{
		metrics: []model.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 1.0},
		},
	}

	r, err := NewFromStore(s, "progress-project", src)
	if err != nil {
		t.Fatal(err)
	}

	var progressCount int
	_, err = r.Run(context.Background(), func(e collector.ProgressEvent) {
		progressCount++
	})
	if err != nil {
		t.Fatal(err)
	}

	if progressCount == 0 {
		t.Error("expected progress events")
	}
}
