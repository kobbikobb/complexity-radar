package report

import (
	"bytes"
	"strings"
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

	reports := BuildFromResult(result, testConfig())

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

func TestBuildFromResultUsesCollectionTimeNotRepoCreation(t *testing.T) {
	collected := time.Date(2026, 7, 14, 15, 3, 0, 0, time.UTC)
	result := collector.CollectionResult{
		Project: model.Project{Name: "TimeTest"},
		Repositories: []collector.RepositoryResult{
			{
				Repository:  model.Repository{CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
				Metrics:     map[model.MetricTypeName]float64{},
				Dimensions:  []scorer.DimensionResult{},
				CollectedAt: collected,
			},
		},
	}

	reports := BuildFromResult(result, testConfig())

	if !reports[0].CollectedAt.Equal(collected) {
		t.Errorf("CollectedAt = %v, want collection time %v (not repo creation date)", reports[0].CollectedAt, collected)
	}
}

func TestBuildFromDBUsesMetricCollectionTime(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "DBTime"}
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

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if reports[0].CollectedAt.IsZero() {
		t.Error("CollectedAt is zero; want the metric collection time")
	}
	if !reports[0].CollectedAt.Equal(m.CollectedAt) {
		t.Errorf("CollectedAt = %v, want metric collected_at %v", reports[0].CollectedAt, m.CollectedAt)
	}
}

func TestBuildFromResultDimensionWeights(t *testing.T) {
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

	reports := BuildFromResult(result, testConfig())
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

	reports := BuildFromResult(result, testConfig())
	if len(reports) != 2 {
		t.Fatalf("got %d reports, want 2", len(reports))
	}
}

func TestBuildFromResultNoMetrics(t *testing.T) {
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

	reports := BuildFromResult(result, testConfig())
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	if len(reports[0].Metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(reports[0].Metrics))
	}
}

func TestBuildFromResultSkipsUnknownMetrics(t *testing.T) {
	result := collector.CollectionResult{
		Project: model.Project{Name: "UnknownMetric"},
		Repositories: []collector.RepositoryResult{
			{
				Repository: model.Repository{CreatedAt: time.Now()},
				Metrics: map[model.MetricTypeName]float64{
					model.MetricTypeSecurityVulnerabilities: 3,
					"totally_unknown_metric":                42,
				},
				Dimensions: []scorer.DimensionResult{
					{Dimension: model.DimensionSecurity, Score: 80, MetricCount: 1},
				},
			},
		},
	}

	reports := BuildFromResult(result, testConfig())

	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}

	metrics := reports[0].Metrics
	if len(metrics) != 1 {
		t.Fatalf("got %d metrics, want 1 (unknown should be skipped)", len(metrics))
	}
	if metrics[0].Name != model.MetricTypeSecurityVulnerabilities {
		t.Errorf("metric name = %q, want %q", metrics[0].Name, model.MetricTypeSecurityVulnerabilities)
	}
	if metrics[0].Dimension == "" {
		t.Error("known metric should have a non-empty dimension")
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

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
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

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
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

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
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

func TestBuildFromDBNoTrendOnFirstCollection(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Trend"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	createMetric(t, s, r.ID, model.MetricTypeSecurityVulnerabilities, 10)

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if reports[0].HasTrend {
		t.Error("HasTrend = true on first collection, want false")
	}
}

func TestBuildFromDBComputesTrendDelta(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Trend"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	createMetric(t, s, r.ID, model.MetricTypeSecurityVulnerabilities, 10) // previous: worse
	createMetric(t, s, r.ID, model.MetricTypeSecurityVulnerabilities, 0)  // latest: no vulns

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if !reports[0].HasTrend {
		t.Fatal("HasTrend = false, want true after two collections")
	}
	if reports[0].OverallDelta <= 0 {
		t.Errorf("OverallDelta = %v, want > 0 (security improved)", reports[0].OverallDelta)
	}
	for _, d := range reports[0].Dimensions {
		if d.Dimension == model.DimensionSecurity && d.Delta <= 0 {
			t.Errorf("security delta = %v, want > 0", d.Delta)
		}
	}
}

func createMetric(t *testing.T, s *store.Store, repoID int64, name model.MetricTypeName, value float64) {
	t.Helper()
	mt, err := s.GetMetricTypeByName(name)
	if err != nil {
		t.Fatalf("GetMetricTypeByName(%s): %v", name, err)
	}
	if err := s.CreateMetric(&model.Metric{RepositoryID: repoID, MetricTypeID: mt.ID, Value: value}); err != nil {
		t.Fatalf("CreateMetric(%s): %v", name, err)
	}
}

func TestBuildFromDBNoRepositories(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "NoRepos"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	reports, err := BuildFromDB(s, *p, testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if len(reports) != 0 {
		t.Fatalf("got %d reports, want 0", len(reports))
	}
}

func TestBuildFromDBPerRepoWarning(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "WarnTest"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/warn", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	var warnBuf bytes.Buffer
	reports, err := BuildFromDB(s, *p, testConfig(), &warnBuf)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	if len(reports) != 0 {
		t.Fatalf("got %d reports, want 0", len(reports))
	}

	warning := warnBuf.String()
	if !strings.Contains(warning, "github.com/org/warn") {
		t.Errorf("warning should mention repo URL, got: %q", warning)
	}
	if !strings.Contains(warning, "No metrics collected") {
		t.Errorf("warning should say 'No metrics collected', got: %q", warning)
	}
}

func TestBuildFromDBListRepositoriesError(t *testing.T) {
	ms := &failingStore{listErr: true}
	_, err := BuildFromDB(ms, model.Project{ID: 1}, testConfig(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "listing repositories") {
		t.Errorf("error = %q, want it to contain 'listing repositories'", err.Error())
	}
}

func TestBuildFromDBGetMetricsError(t *testing.T) {
	ms := &failingStore{metricsErr: true}
	_, err := BuildFromDB(ms, model.Project{ID: 1}, testConfig(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting metrics for") {
		t.Errorf("error = %q, want it to contain 'getting metrics for'", err.Error())
	}
}

func TestBuildFromDBSkipsRetiredMetricTypes(t *testing.T) {
	// Arrange
	weights := scorer.WeightsFromConfig(testConfig().Weights)
	wantScore := scorer.Score(map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 3,
	}, weights).Overall

	s := &stubStore{
		metrics: []model.Metric{
			{MetricTypeID: 1, Value: 3},
			{MetricTypeID: 99, Value: 5000},
		},
		byID: map[int64]*model.MetricType{
			1:  {ID: 1, Name: model.MetricTypeSecurityVulnerabilities, Dimension: model.DimensionSecurity, Unit: "count"},
			99: {ID: 99, Name: "code_loc", Dimension: model.DimensionCode, Unit: "count"},
		},
		byName: map[model.MetricTypeName]*model.MetricType{
			model.MetricTypeSecurityVulnerabilities: {ID: 1, Name: model.MetricTypeSecurityVulnerabilities, Dimension: model.DimensionSecurity, Unit: "count"},
		},
	}

	// Act
	reports, err := BuildFromDB(s, model.Project{ID: 1, Name: "Stale"}, testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	// Assert
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	for _, m := range reports[0].Metrics {
		if m.Name == "code_loc" {
			t.Error("retired metric code_loc should not appear in report")
		}
	}
	if len(reports[0].Metrics) != 1 {
		t.Errorf("got %d metrics, want 1 (retired type skipped)", len(reports[0].Metrics))
	}
	if reports[0].OverallScore != wantScore {
		t.Errorf("overall score = %v, want %v (retired metric must not affect scoring)", reports[0].OverallScore, wantScore)
	}
}

type stubStore struct {
	metrics []model.Metric
	byID    map[int64]*model.MetricType
	byName  map[model.MetricTypeName]*model.MetricType
}

func (s *stubStore) GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error) {
	if mt, ok := s.byName[name]; ok {
		return mt, nil
	}
	return nil, &testError{"unknown metric type"}
}

func (s *stubStore) GetMetricTypeByID(id int64) (*model.MetricType, error) {
	if mt, ok := s.byID[id]; ok {
		return mt, nil
	}
	return nil, &testError{"unknown metric type id"}
}

func (s *stubStore) ListRepositories(_ int64) ([]model.Repository, error) {
	return []model.Repository{{ID: 1, URL: "github.com/org/repo"}}, nil
}

func (s *stubStore) GetMetricsByRepository(_ int64) ([]model.Metric, error) {
	return s.metrics, nil
}

type failingStore struct {
	listErr    bool
	metricsErr bool
}

func (f *failingStore) GetMetricTypeByName(_ model.MetricTypeName) (*model.MetricType, error) {
	return nil, nil
}

func (f *failingStore) GetMetricTypeByID(_ int64) (*model.MetricType, error) {
	return nil, nil
}

func (f *failingStore) ListRepositories(_ int64) ([]model.Repository, error) {
	if f.listErr {
		return nil, &testError{"db connection failed"}
	}
	return []model.Repository{{ID: 1, URL: "github.com/org/repo"}}, nil
}

func (f *failingStore) GetMetricsByRepository(_ int64) ([]model.Metric, error) {
	if f.metricsErr {
		return nil, &testError{"query failed"}
	}
	return nil, nil
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

func TestBuildFromDBExcludesUnknownMetricTypes(t *testing.T) {
	stub := &stubStore{
		metrics: []model.Metric{
			{RepositoryID: 1, MetricTypeID: 1, Value: 26},
			{RepositoryID: 1, MetricTypeID: 2, Value: 960355}, // stale code_loc, removed from model
		},
		byID: map[int64]*model.MetricType{
			1: {ID: 1, Name: model.MetricTypeDependencyCount, Dimension: model.DimensionCode, Unit: "count"},
			2: {ID: 2, Name: model.MetricTypeName("code_loc"), Dimension: model.DimensionCode, Unit: "lines"},
		},
		byName: map[model.MetricTypeName]*model.MetricType{
			model.MetricTypeDependencyCount: {ID: 1, Name: model.MetricTypeDependencyCount, Dimension: model.DimensionCode, Unit: "count"},
		},
	}

	reports, err := BuildFromDB(stub, model.Project{ID: 1, Name: "P"}, testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}

	for _, m := range reports[0].Metrics {
		if m.Name == "code_loc" {
			t.Error("stale code_loc metric leaked into report")
		}
	}
	if len(reports[0].Metrics) != 1 {
		t.Errorf("got %d metrics, want 1 (only dependency_count)", len(reports[0].Metrics))
	}
}
