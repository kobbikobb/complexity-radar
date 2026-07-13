package report

import (
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func testConfig() *config.Config {
	return &config.Config{
		Weights: config.WeightsConfig{
			Security:       0.25,
			Delivery:       0.30,
			Infrastructure: 0.25,
			Code:           0.20,
		},
	}
}

func TestBuildFromResultProducesReports(t *testing.T) {
	builder := NewBuilder()

	result := collector.CollectionResult{
		Project: model.Project{
			Name:        "TestProject",
			Description: "A test",
		},
		Repositories: []collector.RepositoryResult{
			{
				Repository: model.Repository{
					ID:        1,
					CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Metrics: map[model.MetricTypeName]float64{
					model.MetricTypeSecurityVulnerabilities: 5,
					model.MetricTypeBuildSuccessRatio:       0.95,
				},
				Dimensions: []scorer.DimensionResult{
					{Dimension: model.DimensionSecurity, Score: 80, MetricCount: 1},
					{Dimension: model.DimensionDelivery, Score: 95, MetricCount: 1},
				},
				OverallScore: 87.5,
			},
		},
	}

	reports := builder.BuildFromResult(result, testConfig())

	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}

	r := reports[0]
	if r.ProjectName != "TestProject" {
		t.Errorf("project name = %q, want %q", r.ProjectName, "TestProject")
	}
	if r.ProjectDescription != "A test" {
		t.Errorf("project description = %q, want %q", r.ProjectDescription, "A test")
	}
	if r.OverallScore != 87.5 {
		t.Errorf("overall score = %v, want 87.5", r.OverallScore)
	}
	if len(r.Dimensions) != 2 {
		t.Errorf("got %d dimensions, want 2", len(r.Dimensions))
	}
	if len(r.Metrics) != 2 {
		t.Errorf("got %d metrics, want 2", len(r.Metrics))
	}
}

func TestBuildFromResultDimensionWeights(t *testing.T) {
	builder := NewBuilder()

	result := collector.CollectionResult{
		Project: model.Project{Name: "Test"},
		Repositories: []collector.RepositoryResult{
			{
				Dimensions: []scorer.DimensionResult{
					{Dimension: model.DimensionSecurity, Score: 80, MetricCount: 1},
					{Dimension: model.DimensionDelivery, Score: 90, MetricCount: 2},
				},
				Metrics: map[model.MetricTypeName]float64{},
			},
		},
	}

	reports := builder.BuildFromResult(result, testConfig())
	dims := reports[0].Dimensions

	for _, d := range dims {
		switch d.Dimension {
		case model.DimensionSecurity:
			if d.Weight != 25 {
				t.Errorf("security weight = %v, want 25", d.Weight)
			}
		case model.DimensionDelivery:
			if d.Weight != 30 {
				t.Errorf("delivery weight = %v, want 30", d.Weight)
			}
		}
	}
}

func TestBuildFromResultMultipleRepositories(t *testing.T) {
	builder := NewBuilder()

	result := collector.CollectionResult{
		Project: model.Project{Name: "Multi"},
		Repositories: []collector.RepositoryResult{
			{
				Repository: model.Repository{CreatedAt: time.Now()},
				Metrics:    map[model.MetricTypeName]float64{},
				Dimensions: []scorer.DimensionResult{},
			},
			{
				Repository: model.Repository{CreatedAt: time.Now()},
				Metrics:    map[model.MetricTypeName]float64{},
				Dimensions: []scorer.DimensionResult{},
			},
		},
	}

	reports := builder.BuildFromResult(result, testConfig())
	if len(reports) != 2 {
		t.Fatalf("got %d reports, want 2", len(reports))
	}
}

func TestBuildFromResultNoMetrics(t *testing.T) {
	builder := NewBuilder()

	result := collector.CollectionResult{
		Project: model.Project{Name: "Empty"},
		Repositories: []collector.RepositoryResult{
			{
				Repository: model.Repository{CreatedAt: time.Now()},
				Metrics:    map[model.MetricTypeName]float64{},
				Dimensions: []scorer.DimensionResult{},
			},
		},
	}

	reports := builder.BuildFromResult(result, testConfig())
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	if len(reports[0].Metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(reports[0].Metrics))
	}
}

func TestBuildFromDBProducesReports(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "DBTest", Description: "from db"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	mt, err := s.GetMetricTypeByName(model.MetricTypeSecurityVulnerabilities)
	if err != nil {
		t.Fatalf("GetMetricTypeByName: %v", err)
	}

	m := &model.Metric{RepositoryID: r.ID, MetricTypeID: mt.ID, Value: 3}
	if err := s.CreateMetric(m); err != nil {
		t.Fatalf("CreateMetric: %v", err)
	}

	builder := NewBuilder()
	reports, err := builder.BuildFromDB(s, *p, testConfig())
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}

	rep := reports[0]
	if rep.ProjectName != "DBTest" {
		t.Errorf("project name = %q, want %q", rep.ProjectName, "DBTest")
	}
	if rep.OverallScore == 0 {
		t.Error("overall score should not be 0")
	}
	if len(rep.Dimensions) != 4 {
		t.Errorf("got %d dimensions, want 4", len(rep.Dimensions))
	}
	if len(rep.Metrics) != 1 {
		t.Errorf("got %d metrics, want 1", len(rep.Metrics))
	}
	if rep.Metrics[0].Name != model.MetricTypeSecurityVulnerabilities {
		t.Errorf("metric name = %q, want %q", rep.Metrics[0].Name, model.MetricTypeSecurityVulnerabilities)
	}
}

func TestBuildFromDBSkipsEmptyRepositories(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Empty"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/empty", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	builder := NewBuilder()
	reports, err := builder.BuildFromDB(s, *p, testConfig())
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if len(reports) != 0 {
		t.Fatalf("got %d reports, want 0 (no metrics collected)", len(reports))
	}
}

func TestBuildFromDBWithMultipleMetrics(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Multi"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	metricData := []struct {
		name  model.MetricTypeName
		value float64
	}{
		{model.MetricTypeSecurityVulnerabilities, 5},
		{model.MetricTypeBuildSuccessRatio, 0.9},
		{model.MetricTypeDependencyCount, 100},
	}

	for _, md := range metricData {
		mt, err := s.GetMetricTypeByName(md.name)
		if err != nil {
			t.Fatalf("GetMetricTypeByName(%s): %v", md.name, err)
		}
		m := &model.Metric{RepositoryID: r.ID, MetricTypeID: mt.ID, Value: md.value}
		if err := s.CreateMetric(m); err != nil {
			t.Fatalf("CreateMetric(%s): %v", md.name, err)
		}
	}

	builder := NewBuilder()
	reports, err := builder.BuildFromDB(s, *p, testConfig())
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}

	if len(reports[0].Metrics) != 3 {
		t.Errorf("got %d metrics, want 3", len(reports[0].Metrics))
	}

	for _, m := range reports[0].Metrics {
		if m.Normalized == 0 && m.RawValue != 0 {
			t.Errorf("metric %s has normalized=0 with raw=%v", m.Name, m.RawValue)
		}
	}
}

func TestBuildFromDBNoRepositories(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "NoRepos"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	builder := NewBuilder()
	reports, err := builder.BuildFromDB(s, *p, testConfig())
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if len(reports) != 0 {
		t.Fatalf("got %d reports, want 0", len(reports))
	}
}
