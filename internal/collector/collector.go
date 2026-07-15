package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
)

type MetricStore interface {
	GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error)
	CreateMetric(m *model.Metric) error
	CreateProjectMetric(m *model.ProjectMetric) error
	CreateDimensionScore(ds *model.DimensionScore) error
}

type RepositoryResult struct {
	Repository   model.Repository
	Metrics      map[model.MetricTypeName]float64
	Dimensions   []scorer.DimensionResult
	OverallScore float64
	CollectedAt  time.Time
	Errors       []string
}

type CollectionResult struct {
	Project        model.Project
	Repositories   []RepositoryResult
	ProjectMetrics map[model.MetricTypeName]float64
	ProjectErrors  []string
}

// ProgressEvent describes a collection progress update.
type ProgressEvent struct {
	RepositoryURL string
	RepoIndex     int
	RepoTotal     int
	MetricName    string
	MetricIndex   int
	MetricTotal   int
	Message       string
}

// ProgressFunc is called during collection to report progress.
type ProgressFunc func(ProgressEvent)

// noopProgress does nothing when no progress function is provided.
func noopProgress(ProgressEvent) {}

func Collect(ctx context.Context, cfg *config.Config, s MetricStore, project *model.Project, repos []model.Repository, src model.Source, projectSources []model.ProjectSource, onProgress ProgressFunc) (*CollectionResult, error) {
	if onProgress == nil {
		onProgress = noopProgress
	}

	result := &CollectionResult{Project: *project, ProjectMetrics: make(map[model.MetricTypeName]float64)}

	weights := scorer.WeightsFromConfig(cfg.Weights)
	supported := src.SupportedMetrics()

	for repoIdx, repo := range repos {
		onProgress(ProgressEvent{
			RepositoryURL: repo.URL,
			RepoIndex:     repoIdx + 1,
			RepoTotal:     len(repos),
			Message:       fmt.Sprintf("Collecting from %s (branch: %s)", repo.URL, repo.Branch),
		})

		repoResult := RepositoryResult{
			Repository: repo,
			Metrics:    make(map[model.MetricTypeName]float64),
		}

		metrics, err := src.Collect(ctx, repo)
		if err != nil {
			repoResult.Errors = append(repoResult.Errors, fmt.Sprintf("collection failed: %v", err))
			result.Repositories = append(result.Repositories, repoResult)
			continue
		}

		for i, m := range metrics {
			onProgress(ProgressEvent{
				RepositoryURL: repo.URL,
				RepoIndex:     repoIdx + 1,
				RepoTotal:     len(repos),
				MetricName:    string(m.Type),
				MetricIndex:   i + 1,
				MetricTotal:   len(supported),
				Message:       fmt.Sprintf("  Storing %s = %.1f", m.Type, m.Value),
			})

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
			repoResult.CollectedAt = dbMetric.CollectedAt
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

		onProgress(ProgressEvent{
			RepositoryURL: repo.URL,
			RepoIndex:     repoIdx + 1,
			RepoTotal:     len(repos),
			Message:       fmt.Sprintf("Score: %.1f", repoResult.OverallScore),
		})

		result.Repositories = append(result.Repositories, repoResult)
	}

	collectProjectMetrics(ctx, s, project, projectSources, result, onProgress)

	return result, nil
}

func collectProjectMetrics(ctx context.Context, s MetricStore, project *model.Project, sources []model.ProjectSource, result *CollectionResult, onProgress ProgressFunc) {
	for _, src := range sources {
		metrics, err := src.CollectProject(ctx, *project)
		if err != nil {
			result.ProjectErrors = append(result.ProjectErrors, fmt.Sprintf("%s: %v", src.Name(), err))
			continue
		}
		for _, m := range metrics {
			mt, err := s.GetMetricTypeByName(m.Type)
			if err != nil {
				result.ProjectErrors = append(result.ProjectErrors, fmt.Sprintf("metric type %s: %v", m.Type, err))
				continue
			}
			pm := &model.ProjectMetric{ProjectID: project.ID, MetricTypeID: mt.ID, Value: m.Value}
			if err := s.CreateProjectMetric(pm); err != nil {
				result.ProjectErrors = append(result.ProjectErrors, fmt.Sprintf("storing project metric %s: %v", m.Type, err))
				continue
			}
			result.ProjectMetrics[m.Type] = m.Value
			onProgress(ProgressEvent{Message: fmt.Sprintf("  Storing project %s = %.1f", m.Type, m.Value)})
		}
	}
}
