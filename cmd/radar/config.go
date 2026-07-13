package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

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
