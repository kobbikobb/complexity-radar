package collector

import (
	"context"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

type mockSource struct {
	metrics []sources.SourceMetric
	err     error
}

func (m *mockSource) Name() string { return "mock" }

func (m *mockSource) Collect(ctx context.Context, repo model.Repository) ([]sources.SourceMetric, error) {
	return m.metrics, m.err
}

func (m *mockSource) SupportedMetrics() []model.MetricTypeName {
	return []model.MetricTypeName{
		model.MetricTypeDeployFrequency,
		model.MetricTypeSecurityVulnerabilities,
		model.MetricTypeBuildSuccessRatio,
		model.MetricTypeDependencyCount,
	}
}

func TestCollect(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:        "test-project",
			Description: "A test project",
		},
		Repositories: []config.RepositoryConfig{
			{URL: "github.com/org/repo", Branch: "main"},
		},
		Weights: config.DefaultWeights(),
	}

	src := &mockSource{
		metrics: []sources.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 5.0},
			{Type: model.MetricTypeSecurityVulnerabilities, Value: 2.0},
			{Type: model.MetricTypeBuildSuccessRatio, Value: 0.95},
			{Type: model.MetricTypeDependencyCount, Value: 42.0},
		},
	}

	result, err := Collect(context.Background(), cfg, s, src)
	if err != nil {
		t.Fatal(err)
	}

	if result.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got %q", result.Project.Name)
	}

	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(result.Repositories))
	}

	repo := result.Repositories[0]
	if repo.Repository.URL != "github.com/org/repo" {
		t.Errorf("expected repo URL 'github.com/org/repo', got %q", repo.Repository.URL)
	}

	if len(repo.Metrics) != 4 {
		t.Errorf("expected 4 metrics, got %d", len(repo.Metrics))
	}

	if repo.OverallScore == 0 {
		t.Error("expected non-zero overall score")
	}

	if len(repo.Dimensions) == 0 {
		t.Error("expected dimension scores")
	}
}

func TestCollectWithMultipleRepos(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "multi-repo-project",
		},
		Repositories: []config.RepositoryConfig{
			{URL: "github.com/org/repo1", Branch: "main"},
			{URL: "github.com/org/repo2", Branch: "develop"},
		},
		Weights: config.DefaultWeights(),
	}

	src := &mockSource{
		metrics: []sources.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 3.0},
		},
	}

	result, err := Collect(context.Background(), cfg, s, src)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(result.Repositories))
	}
}

func TestCollectWithSourceError(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "error-project",
		},
		Repositories: []config.RepositoryConfig{
			{URL: "github.com/org/repo", Branch: "main"},
		},
		Weights: config.DefaultWeights(),
	}

	src := &mockSource{
		err: context.DeadlineExceeded,
	}

	result, err := Collect(context.Background(), cfg, s, src)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(result.Repositories))
	}

	repo := result.Repositories[0]
	if len(repo.Errors) == 0 {
		t.Error("expected errors from failed collection")
	}
}
