package collector

import (
	"context"
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/sources"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

type RepositoryResult struct {
	Repository   model.Repository
	Metrics      map[model.MetricTypeName]float64
	Dimensions   []scorer.DimensionResult
	OverallScore float64
	Errors       []string
}

type CollectionResult struct {
	Project      model.Project
	Repositories []RepositoryResult
}

func Collect(ctx context.Context, cfg *config.Config, s *store.Store, src sources.Source) (*CollectionResult, error) {
	project := &model.Project{
		Name:        cfg.Project.Name,
		Description: cfg.Project.Description,
	}
	if err := s.CreateProject(project); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	result := &CollectionResult{Project: *project}

	weights := scorer.WeightsFromConfig(cfg.Weights)

	for _, repoCfg := range cfg.Repositories {
		repo := &model.Repository{
			ProjectID: project.ID,
			URL:       repoCfg.URL,
			Branch:    repoCfg.Branch,
		}
		if err := s.CreateRepository(repo); err != nil {
			return nil, fmt.Errorf("creating repository %s: %w", repoCfg.URL, err)
		}

		repoResult := RepositoryResult{
			Repository: *repo,
			Metrics:    make(map[model.MetricTypeName]float64),
		}

		metrics, err := src.Collect(ctx, *repo)
		if err != nil {
			repoResult.Errors = append(repoResult.Errors, fmt.Sprintf("collection failed: %v", err))
			result.Repositories = append(result.Repositories, repoResult)
			continue
		}

		for _, m := range metrics {
			mt, err := s.GetMetricTypeByName(m.Type)
			if err != nil {
				repoResult.Errors = append(repoResult.Errors, fmt.Sprintf("metric type %s: %v", m.Type, err))
				continue
			}

			dbMetric := &model.Metric{
				RepositoryID: repo.ID,
				MetricTypeID: mt.ID,
				Value:        m.Value,
			}
			if err := s.CreateMetric(dbMetric); err != nil {
				repoResult.Errors = append(repoResult.Errors, fmt.Sprintf("storing metric %s: %v", m.Type, err))
				continue
			}

			repoResult.Metrics[m.Type] = m.Value
		}

		scoreResult := scorer.Score(repoResult.Metrics, weights)
		repoResult.OverallScore = scoreResult.Overall
		repoResult.Dimensions = scoreResult.Dimensions

		for _, d := range scoreResult.Dimensions {
			ds := &model.DimensionScore{
				RepositoryID: repo.ID,
				Dimension:    d.Dimension,
				Score:        d.Score,
				Weight:       cfg.Weights.Weight(string(d.Dimension)),
			}
			if err := s.CreateDimensionScore(ds); err != nil {
				repoResult.Errors = append(repoResult.Errors, fmt.Sprintf("storing dimension score: %v", err))
			}
		}

		result.Repositories = append(result.Repositories, repoResult)
	}

	return result, nil
}
