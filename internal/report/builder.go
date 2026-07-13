package report

import (
	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/terminal"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
)

type Store interface {
	GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error)
	GetMetricTypeByID(id int64) (*model.MetricType, error)
	ListRepositories(projectID int64) ([]model.Repository, error)
	GetMetricsByRepository(repoID int64) ([]model.Metric, error)
}

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) BuildFromResult(result collector.CollectionResult, cfg *config.Config) []terminal.Report {
	reports := make([]terminal.Report, 0, len(result.Repositories))
	for _, repoResult := range result.Repositories {
		dimReports := buildDimensionReports(repoResult.Dimensions, cfg.Weights)

		metricReports := make([]terminal.MetricReport, 0, len(repoResult.Metrics))
		for name, raw := range repoResult.Metrics {
			normalized := scorer.NormalizeMetric(name, raw)
			metricReports = append(metricReports, terminal.MetricReport{
				Name:       name,
				Dimension:  metricDimension(name),
				RawValue:   raw,
				Normalized: normalized,
				Unit:       metricUnit(name),
			})
		}

		reports = append(reports, terminal.Report{
			ProjectName:        result.Project.Name,
			ProjectDescription: result.Project.Description,
			OverallScore:       repoResult.OverallScore,
			Dimensions:         dimReports,
			Metrics:            metricReports,
			CollectedAt:        repoResult.Repository.CreatedAt,
		})
	}
	return reports
}

func (b *Builder) BuildFromDB(store Store, project model.Project, cfg *config.Config) ([]terminal.Report, error) {
	weights := scorer.WeightsFromConfig(cfg.Weights)

	repos, err := store.ListRepositories(project.ID)
	if err != nil {
		return nil, err
	}

	var reports []terminal.Report
	for _, repo := range repos {
		metrics, err := store.GetMetricsByRepository(repo.ID)
		if err != nil {
			return nil, err
		}
		if len(metrics) == 0 {
			continue
		}

		rawMetrics := make(map[model.MetricTypeName]float64)
		for _, m := range metrics {
			mt, err := store.GetMetricTypeByID(m.MetricTypeID)
			if err != nil {
				continue
			}
			rawMetrics[mt.Name] = m.Value
		}

		scoreResult := scorer.Score(rawMetrics, weights)

		dimReports := buildDimensionReports(scoreResult.Dimensions, cfg.Weights)

		metricReports := make([]terminal.MetricReport, 0, len(rawMetrics))
		for name, raw := range rawMetrics {
			mt, err := store.GetMetricTypeByName(name)
			if err != nil {
				continue
			}
			normalized := scorer.NormalizeMetric(name, raw)
			metricReports = append(metricReports, terminal.MetricReport{
				Name:       name,
				Dimension:  mt.Dimension,
				RawValue:   raw,
				Normalized: normalized,
				Unit:       mt.Unit,
			})
		}

		reports = append(reports, terminal.Report{
			ProjectName:        project.Name,
			ProjectDescription: project.Description,
			OverallScore:       scoreResult.Overall,
			Dimensions:         dimReports,
			Metrics:            metricReports,
		})
	}

	return reports, nil
}

func buildDimensionReports(dimensions []scorer.DimensionResult, cfg config.WeightsConfig) []terminal.DimensionReport {
	dimReports := make([]terminal.DimensionReport, len(dimensions))
	for i, d := range dimensions {
		dimReports[i] = terminal.DimensionReport{
			Dimension:   d.Dimension,
			Score:       d.Score,
			Weight:      cfg.Weight(string(d.Dimension)) * 100,
			MetricCount: d.MetricCount,
		}
	}
	return dimReports
}

func metricDimension(name model.MetricTypeName) model.Dimension {
	mt, ok := metricTypeLookup(name)
	if !ok {
		return ""
	}
	return mt.Dimension
}

func metricUnit(name model.MetricTypeName) string {
	mt, ok := metricTypeLookup(name)
	if !ok {
		return ""
	}
	return mt.Unit
}

func metricTypeLookup(name model.MetricTypeName) (model.MetricType, bool) {
	for _, mt := range model.MetricTypes() {
		if mt.Name == name {
			return mt, true
		}
	}
	return model.MetricType{}, false
}
