package collector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
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
		model.MetricTypeBuildSuccessRatio,
		model.MetricTypeDependencyCount,
	}
}

type fakeMetricStore struct {
	metricTypes     map[model.MetricTypeName]*model.MetricType
	metrics         []model.Metric
	dimensionScores []model.DimensionScore
	nextMetricID    int64
	nextDSID        int64
}

func newFakeMetricStore() *fakeMetricStore {
	types := make(map[model.MetricTypeName]*model.MetricType)
	for _, mt := range model.MetricTypes() {
		mt := mt
		types[mt.Name] = &mt
	}
	for _, mt := range model.DisplayMetricTypes() {
		mt := mt
		types[mt.Name] = &mt
	}
	return &fakeMetricStore{metricTypes: types}
}

func (f *fakeMetricStore) GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error) {
	mt, ok := f.metricTypes[name]
	if !ok {
		return nil, fmt.Errorf("metric type %q not found", name)
	}
	return mt, nil
}

func (f *fakeMetricStore) CreateMetric(m *model.Metric) error {
	f.nextMetricID++
	m.ID = f.nextMetricID
	m.CollectedAt = time.Now().UTC()
	f.metrics = append(f.metrics, *m)
	return nil
}

func (f *fakeMetricStore) CreateDimensionScore(ds *model.DimensionScore) error {
	f.nextDSID++
	ds.ID = f.nextDSID
	f.dimensionScores = append(f.dimensionScores, *ds)
	return nil
}

func TestCollect(t *testing.T) {
	s := newFakeMetricStore()

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
		metrics: []model.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 5.0},
			{Type: model.MetricTypeSecurityVulnerabilities, Value: 2.0},
			{Type: model.MetricTypeBuildSuccessRatio, Value: 0.95},
			{Type: model.MetricTypeDependencyCount, Value: 42.0},
		},
	}

	project := &model.Project{Name: "test-project", Description: "A test project"}
	repo := model.Repository{ProjectID: project.ID, URL: "github.com/org/repo", Branch: "main"}

	result, err := Collect(context.Background(), cfg, s, project, []model.Repository{repo}, src, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got %q", result.Project.Name)
	}

	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(result.Repositories))
	}

	repoResult := result.Repositories[0]
	if repoResult.Repository.URL != "github.com/org/repo" {
		t.Errorf("expected repo URL 'github.com/org/repo', got %q", repoResult.Repository.URL)
	}

	if len(repoResult.Metrics) != 4 {
		t.Errorf("expected 4 metrics, got %d", len(repoResult.Metrics))
	}

	if repoResult.OverallScore == 0 {
		t.Error("expected non-zero overall score")
	}

	if len(repoResult.Dimensions) == 0 {
		t.Error("expected dimension scores")
	}

	if len(s.metrics) != 4 {
		t.Errorf("expected 4 stored metrics, got %d", len(s.metrics))
	}

	if len(s.dimensionScores) == 0 {
		t.Error("expected stored dimension scores")
	}
}

func TestCollectWithMultipleRepos(t *testing.T) {
	s := newFakeMetricStore()

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
		metrics: []model.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: 3.0},
		},
	}

	project := &model.Project{Name: "multi-repo-project"}
	repo1 := model.Repository{ProjectID: project.ID, URL: "github.com/org/repo1", Branch: "main"}
	repo2 := model.Repository{ProjectID: project.ID, URL: "github.com/org/repo2", Branch: "develop"}

	result, err := Collect(context.Background(), cfg, s, project, []model.Repository{repo1, repo2}, src, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(result.Repositories))
	}

	if len(s.metrics) != 2 {
		t.Errorf("expected 2 stored metrics, got %d", len(s.metrics))
	}
}

func TestCollectWithSourceError(t *testing.T) {
	s := newFakeMetricStore()

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

	project := &model.Project{Name: "error-project"}
	repo := model.Repository{ProjectID: project.ID, URL: "github.com/org/repo", Branch: "main"}

	result, err := Collect(context.Background(), cfg, s, project, []model.Repository{repo}, src, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(result.Repositories))
	}

	repoResult := result.Repositories[0]
	if len(repoResult.Errors) == 0 {
		t.Error("expected errors from failed collection")
	}

	if len(s.metrics) != 0 {
		t.Errorf("expected 0 stored metrics on error, got %d", len(s.metrics))
	}
}
