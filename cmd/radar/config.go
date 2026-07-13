package main

import (
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
			URL:    r.URL,
			Branch: r.Branch,
		})
	}

	return cfg, nil
}
