package main

import (
	"errors"
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

func openStore(dbPath string) (*store.Store, error) {
	s, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return s, nil
}

// findOrCreateProject returns the existing project by name, or creates a new one.
// When name is empty, returns the first project if any exist.
func findOrCreateProject(s *store.Store, name, description string) (*model.Project, error) {
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
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("looking up project: %w", err)
	}

	p = &model.Project{
		Name:        name,
		Description: description,
	}
	if err := s.CreateProject(p); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	return p, nil
}

func findProject(s *store.Store, name string) (*model.Project, error) {
	projects, err := s.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no project configured. Run 'radar init' first")
	}

	if name != "" {
		for _, p := range projects {
			if p.Name == name {
				return &p, nil
			}
		}
		return nil, fmt.Errorf("project %q not found", name)
	}

	return &projects[0], nil
}

func buildConfigFromDB(s *store.Store, project *model.Project) (*config.Config, error) {
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
			URL:             r.URL,
			Branch:          r.Branch,
			GitopsRepoURL:   r.GitopsRepoURL,
			DeployDetection: r.DeployDetection,
		})
	}

	return cfg, nil
}
