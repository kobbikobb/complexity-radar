package store

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s, err := NewFromDB(db)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestCreateAndGetProject(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Test Project", Description: "A test"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	if p.ID == 0 {
		t.Fatal("expected project ID to be set")
	}

	got, err := s.GetProject(p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}

	if got.Name != p.Name {
		t.Errorf("name = %q, want %q", got.Name, p.Name)
	}
	if got.Description != p.Description {
		t.Errorf("description = %q, want %q", got.Description, p.Description)
	}
}

func TestListProjects(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateProject(&model.Project{Name: "Alpha"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := s.CreateProject(&model.Project{Name: "Beta"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}
	if projects[0].Name != "Alpha" {
		t.Errorf("first project = %q, want %q", projects[0].Name, "Alpha")
	}
}

func TestUpdateProject(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Old", Description: "desc"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	p.Name = "New"
	if err := s.UpdateProject(p); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	got, err := s.GetProject(p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "New" {
		t.Errorf("name = %q, want %q", got.Name, "New")
	}
}

func TestDeleteProject(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "ToDelete"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	_, err := s.GetProject(p.ID)
	if err == nil {
		t.Fatal("expected error for deleted project")
	}
}

func TestDeleteProjectNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteProject(9999)
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
}

func TestCreateAndGetRepository(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	if r.ID == 0 {
		t.Fatal("expected repository ID to be set")
	}

	got, err := s.GetRepository(r.ID)
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}

	if got.URL != r.URL {
		t.Errorf("url = %q, want %q", got.URL, r.URL)
	}
	if got.Branch != r.Branch {
		t.Errorf("branch = %q, want %q", got.Branch, r.Branch)
	}
	if got.ProjectID != p.ID {
		t.Errorf("project_id = %d, want %d", got.ProjectID, p.ID)
	}
}

func TestRepositoryDeployDetectionRoundTrip(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", DeployDetection: config.DeployDetectionReleases}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	got, err := s.GetRepository(r.ID)
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}

	if got.DeployDetection != config.DeployDetectionReleases {
		t.Errorf("deploy_detection = %q, want %q", got.DeployDetection, config.DeployDetectionReleases)
	}
}

func TestRepositoryReleaseOptionsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", DeployDetection: config.DeployDetectionTags, IncludePrereleases: true, TagPrefix: "v"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	got, err := s.GetRepository(r.ID)
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}

	if !got.IncludePrereleases {
		t.Errorf("include_prereleases = %v, want true", got.IncludePrereleases)
	}
	if got.TagPrefix != "v" {
		t.Errorf("tag_prefix = %q, want %q", got.TagPrefix, "v")
	}
	if got.DeployDetection != config.DeployDetectionTags {
		t.Errorf("deploy_detection = %q, want %q", got.DeployDetection, config.DeployDetectionTags)
	}
}

func TestCreateRepositoryDefaultsDeployDetection(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	got, err := s.GetRepository(r.ID)
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}

	if got.DeployDetection != config.DeployDetectionReleases {
		t.Errorf("deploy_detection = %q, want %q", got.DeployDetection, config.DeployDetectionReleases)
	}
}

func TestListRepositories(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	if err := s.CreateRepository(&model.Repository{ProjectID: p.ID, URL: "github.com/org/repo1"}); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	if err := s.CreateRepository(&model.Repository{ProjectID: p.ID, URL: "github.com/org/repo2"}); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	repos, err := s.ListRepositories(p.ID)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
}

func TestDeleteRepository(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	if err := s.DeleteRepository(r.ID); err != nil {
		t.Fatalf("DeleteRepository: %v", err)
	}

	_, err := s.GetRepository(r.ID)
	if err == nil {
		t.Fatal("expected error for deleted repository")
	}
}

func TestForeignKeyCascadeDelete(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Cascade"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	_, err := s.GetRepository(r.ID)
	if err == nil {
		t.Fatal("expected repository to be cascade-deleted")
	}
}

func TestEnsureAndGetMetricTypes(t *testing.T) {
	s := newTestStore(t)

	mt, err := s.GetMetricTypeByName(model.MetricTypeSecurityVulnerabilities)
	if err != nil {
		t.Fatalf("GetMetricTypeByName: %v", err)
	}

	if mt.Dimension != model.DimensionSecurity {
		t.Errorf("dimension = %q, want %q", mt.Dimension, model.DimensionSecurity)
	}
	if mt.Unit != "count" {
		t.Errorf("unit = %q, want %q", mt.Unit, "count")
	}

	mt2, err := s.GetMetricTypeByName(model.MetricTypeBuildSuccessRatio)
	if err != nil {
		t.Fatalf("GetMetricTypeByName build_success_ratio: %v", err)
	}
	if mt2.Dimension != model.DimensionDelivery {
		t.Errorf("dimension = %q, want %q", mt2.Dimension, model.DimensionDelivery)
	}
}

func TestEnsureMetricTypesIdempotent(t *testing.T) {
	s := newTestStore(t)

	if err := s.EnsureMetricTypes(); err != nil {
		t.Fatalf("second EnsureMetricTypes: %v", err)
	}
}

func TestCreateAndGetMetric(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "P"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	mt, _ := s.GetMetricTypeByName(model.MetricTypeSecurityVulnerabilities)

	m := &model.Metric{
		RepositoryID: r.ID,
		MetricTypeID: mt.ID,
		Value:        3,
	}
	if err := s.CreateMetric(m); err != nil {
		t.Fatalf("CreateMetric: %v", err)
	}

	metrics, err := s.GetMetricsByRepository(r.ID)
	if err != nil {
		t.Fatalf("GetMetricsByRepository: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("got %d metrics, want 1", len(metrics))
	}
	if metrics[0].Value != 3 {
		t.Errorf("value = %v, want 3", metrics[0].Value)
	}
}

func TestCreateAndGetDimensionScore(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "P"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	ds := &model.DimensionScore{
		RepositoryID: r.ID,
		Dimension:    model.DimensionSecurity,
		Score:        85.5,
		Weight:       0.25,
	}
	if err := s.CreateDimensionScore(ds); err != nil {
		t.Fatalf("CreateDimensionScore: %v", err)
	}

	scores, err := s.GetDimensionScoresByRepository(r.ID)
	if err != nil {
		t.Fatalf("GetDimensionScoresByRepository: %v", err)
	}

	if len(scores) != 1 {
		t.Fatalf("got %d scores, want 1", len(scores))
	}
	if scores[0].Score != 85.5 {
		t.Errorf("score = %v, want 85.5", scores[0].Score)
	}
	if scores[0].Dimension != model.DimensionSecurity {
		t.Errorf("dimension = %q, want %q", scores[0].Dimension, model.DimensionSecurity)
	}
}

func TestCreateAndGetProjectReport(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "P"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	pr := &model.ProjectReport{
		ProjectID:  p.ID,
		TotalScore: 72.3,
	}
	if err := s.CreateProjectReport(pr); err != nil {
		t.Fatalf("CreateProjectReport: %v", err)
	}

	got, err := s.GetProjectReport(pr.ID)
	if err != nil {
		t.Fatalf("GetProjectReport: %v", err)
	}

	if got.TotalScore != 72.3 {
		t.Errorf("total_score = %v, want 72.3", got.TotalScore)
	}
	if got.ProjectID != p.ID {
		t.Errorf("project_id = %d, want %d", got.ProjectID, p.ID)
	}
}

func TestAddAndGetProjectReportScores(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "P"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	pr := &model.ProjectReport{ProjectID: p.ID, TotalScore: 80}
	if err := s.CreateProjectReport(pr); err != nil {
		t.Fatalf("CreateProjectReport: %v", err)
	}

	dimensions := []struct {
		dimension model.Dimension
		score     float64
		weight    float64
	}{
		{model.DimensionSecurity, 90, 0.25},
		{model.DimensionDelivery, 75, 0.30},
		{model.DimensionInfrastructure, 80, 0.25},
		{model.DimensionCode, 85, 0.20},
	}

	for _, d := range dimensions {
		prs := &model.ProjectReportScore{
			ProjectReportID: pr.ID,
			Dimension:       d.dimension,
			Score:           d.score,
			Weight:          d.weight,
		}
		if err := s.AddProjectReportScore(prs); err != nil {
			t.Fatalf("AddProjectReportScore(%s): %v", d.dimension, err)
		}
	}

	scores, err := s.GetProjectReportScores(pr.ID)
	if err != nil {
		t.Fatalf("GetProjectReportScores: %v", err)
	}

	if len(scores) != 4 {
		t.Fatalf("got %d scores, want 4", len(scores))
	}
}

func TestGetProjectNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetProject(9999)
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
}

func TestGetRepositoryNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetRepository(9999)
	if err == nil {
		t.Fatal("expected error for non-existent repository")
	}
}

func TestGetProjectReportNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetProjectReport(9999)
	if err == nil {
		t.Fatal("expected error for non-existent project report")
	}
}

func TestGetMetricTypeNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetMetricTypeByName("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent metric type")
	}
}

func TestGetProjectByNameNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetProjectByName("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestFindOrCreateRepositoryCreatesNew(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo", "main")
	if err != nil {
		t.Fatalf("FindOrCreateRepository: %v", err)
	}

	if r.ID == 0 {
		t.Fatal("expected repository ID to be set")
	}
	if r.URL != "github.com/org/repo" {
		t.Errorf("url = %q, want %q", r.URL, "github.com/org/repo")
	}
	if r.Branch != "main" {
		t.Errorf("branch = %q, want %q", r.Branch, "main")
	}
	if r.ProjectID != p.ID {
		t.Errorf("project_id = %d, want %d", r.ProjectID, p.ID)
	}
}

func TestFindOrCreateRepositoryIdempotent(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r1, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo", "main")
	if err != nil {
		t.Fatalf("FindOrCreateRepository (first): %v", err)
	}

	r2, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo", "main")
	if err != nil {
		t.Fatalf("FindOrCreateRepository (second): %v", err)
	}

	if r1.ID != r2.ID {
		t.Errorf("expected same repo ID, got %d and %d", r1.ID, r2.ID)
	}

	repos, err := s.ListRepositories(p.ID)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(repos))
	}
}

func TestFindOrCreateRepositoryDifferentURLs(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r1, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo1", "main")
	if err != nil {
		t.Fatalf("FindOrCreateRepository (repo1): %v", err)
	}

	r2, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo2", "develop")
	if err != nil {
		t.Fatalf("FindOrCreateRepository (repo2): %v", err)
	}

	if r1.ID == r2.ID {
		t.Error("expected different repo IDs for different URLs")
	}

	repos, err := s.ListRepositories(p.ID)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestFindOrCreateRepositoryUpdatesBranch(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Parent"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r1, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo", "main")
	if err != nil {
		t.Fatalf("FindOrCreateRepository (main): %v", err)
	}

	r2, err := s.FindOrCreateRepository(p.ID, "github.com/org/repo", "develop")
	if err != nil {
		t.Fatalf("FindOrCreateRepository (develop): %v", err)
	}

	if r1.ID != r2.ID {
		t.Errorf("expected same repo ID, got %d and %d", r1.ID, r2.ID)
	}
	if r2.Branch != "develop" {
		t.Errorf("branch = %q, want %q", r2.Branch, "develop")
	}

	got, err := s.GetRepository(r2.ID)
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if got.Branch != "develop" {
		t.Errorf("persisted branch = %q, want %q", got.Branch, "develop")
	}
}

func TestMigrationIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer func() { _ = db.Close() }()

	s1, err := NewFromDB(db)
	if err != nil {
		t.Fatalf("first NewFromDB: %v", err)
	}
	_ = s1

	s2, err := NewFromDB(db)
	if err != nil {
		t.Fatalf("second NewFromDB (migration idempotent): %v", err)
	}
	_ = s2
}

func TestFullWorkflow(t *testing.T) {
	s := newTestStore(t)

	p := &model.Project{Name: "Full Test", Description: "End-to-end"}
	if err := s.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	r := &model.Repository{ProjectID: p.ID, URL: "github.com/org/repo", Branch: "main"}
	if err := s.CreateRepository(r); err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}

	mt, err := s.GetMetricTypeByName(model.MetricTypeDependencyCount)
	if err != nil {
		t.Fatalf("GetMetricTypeByName: %v", err)
	}

	m := &model.Metric{RepositoryID: r.ID, MetricTypeID: mt.ID, Value: 42}
	if err := s.CreateMetric(m); err != nil {
		t.Fatalf("CreateMetric: %v", err)
	}

	ds := &model.DimensionScore{
		RepositoryID: r.ID,
		Dimension:    model.DimensionCode,
		Score:        70,
		Weight:       0.20,
	}
	if err := s.CreateDimensionScore(ds); err != nil {
		t.Fatalf("CreateDimensionScore: %v", err)
	}

	pr := &model.ProjectReport{ProjectID: p.ID, TotalScore: 75}
	if err := s.CreateProjectReport(pr); err != nil {
		t.Fatalf("CreateProjectReport: %v", err)
	}

	prs := &model.ProjectReportScore{
		ProjectReportID: pr.ID,
		Dimension:       model.DimensionCode,
		Score:           70,
		Weight:          0.20,
	}
	if err := s.AddProjectReportScore(prs); err != nil {
		t.Fatalf("AddProjectReportScore: %v", err)
	}

	gotP, err := s.GetProject(p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	gotR, err := s.GetRepository(r.ID)
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	gotMetrics, err := s.GetMetricsByRepository(r.ID)
	if err != nil {
		t.Fatalf("GetMetricsByRepository: %v", err)
	}
	gotDS, err := s.GetDimensionScoresByRepository(r.ID)
	if err != nil {
		t.Fatalf("GetDimensionScoresByRepository: %v", err)
	}
	gotPR, err := s.GetProjectReport(pr.ID)
	if err != nil {
		t.Fatalf("GetProjectReport: %v", err)
	}
	gotPRS, err := s.GetProjectReportScores(pr.ID)
	if err != nil {
		t.Fatalf("GetProjectReportScores: %v", err)
	}

	if gotP.Name != "Full Test" {
		t.Errorf("project name = %q", gotP.Name)
	}
	if gotR.URL != "github.com/org/repo" {
		t.Errorf("repo url = %q", gotR.URL)
	}
	if len(gotMetrics) != 1 {
		t.Errorf("got %d metrics, want 1", len(gotMetrics))
	}
	if len(gotDS) != 1 {
		t.Errorf("got %d dimension scores, want 1", len(gotDS))
	}
	if gotPR.TotalScore != 75 {
		t.Errorf("total score = %v, want 75", gotPR.TotalScore)
	}
	if len(gotPRS) != 1 {
		t.Errorf("got %d report scores, want 1", len(gotPRS))
	}
}
