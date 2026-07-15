package runner

import (
	"context"
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
)

type Store interface {
	ListProjects() ([]model.Project, error)
	GetProjectByName(name string) (*model.Project, error)
	CreateProject(p *model.Project) error
	ListRepositories(projectID int64) ([]model.Repository, error)
	FindOrCreateRepository(projectID int64, url, branch string) (*model.Repository, error)
	GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error)
	CreateMetric(m *model.Metric) error
	CreateDimensionScore(ds *model.DimensionScore) error
	Close() error
}

type Runner struct {
	store   Store
	project *model.Project
	cfg     *config.Config
	repos   []model.Repository
	source  model.Source
}

func NewFromStore(s Store, projectName string, source model.Source) (*Runner, error) {
	project, err := FindOrCreateProject(s, projectName)
	if err != nil {
		return nil, err
	}

	cfg, err := BuildConfigFromDB(s, project)
	if err != nil {
		return nil, err
	}

	repos, err := resolveRepos(s, project, cfg)
	if err != nil {
		return nil, err
	}

	return &Runner{
		store:   s,
		project: project,
		cfg:     cfg,
		repos:   repos,
		source:  source,
	}, nil
}

func (r *Runner) Project() *model.Project {
	return r.project
}

func (r *Runner) Config() *config.Config {
	return r.cfg
}

func (r *Runner) Repositories() []model.Repository {
	return r.repos
}

func (r *Runner) Run(ctx context.Context, onProgress collector.ProgressFunc) (*collector.CollectionResult, error) {
	return collector.Collect(ctx, r.cfg, r.store, r.project, r.repos, r.source, onProgress)
}

func (r *Runner) Close() error {
	return r.store.Close()
}

func FindOrCreateProject(s Store, name string) (*model.Project, error) {
	if name == "" {
		projects, err := s.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("listing projects: %w", err)
		}
		if len(projects) == 0 {
			return nil, fmt.Errorf("no project configured. Run 'radar init' first")
		}
		return &projects[0], nil
	}

	p, err := s.GetProjectByName(name)
	if err == nil {
		return p, nil
	}

	p = &model.Project{Name: name}
	if err := s.CreateProject(p); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	return p, nil
}

func BuildConfigFromDB(s Store, project *model.Project) (*config.Config, error) {
	repos, err := s.ListRepositories(project.ID)
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories configured. Run 'radar init' to add repositories")
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:        project.Name,
			Description: project.Description,
		},
		Weights: config.DefaultWeights(),
	}

	for _, r := range repos {
		cfg.Repositories = append(cfg.Repositories, config.RepositoryConfig{
			URL:                r.URL,
			Branch:             r.Branch,
			GitopsRepoURL:      r.GitopsRepoURL,
			DeployDetection:    r.DeployDetection,
			IncludePrereleases: r.IncludePrereleases,
			TagPrefix:          r.TagPrefix,
		})
	}

	return cfg, nil
}

func resolveRepos(s Store, project *model.Project, cfg *config.Config) ([]model.Repository, error) {
	var repos []model.Repository
	for _, repoCfg := range cfg.Repositories {
		repo, err := s.FindOrCreateRepository(project.ID, repoCfg.URL, repoCfg.Branch)
		if err != nil {
			return nil, fmt.Errorf("finding repository %s: %w", repoCfg.URL, err)
		}
		repos = append(repos, *repo)
	}
	return repos, nil
}
