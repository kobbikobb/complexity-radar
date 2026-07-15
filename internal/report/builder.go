package report

import (
	"fmt"
	"io"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/terminal"
)

type Store interface {
	GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error)
	GetMetricTypeByID(id int64) (*model.MetricType, error)
	ListRepositories(projectID int64) ([]model.Repository, error)
	GetMetricsByRepository(repoID int64) ([]model.Metric, error)
}

func BuildFromResult(result collector.CollectionResult, cfg *config.Config) []terminal.Report {
	reports := make([]terminal.Report, 0, len(result.Repositories))
	for _, repoResult := range result.Repositories {
		dimReports := buildDimensionReports(repoResult.Dimensions, cfg.Weights)

		metricReports := make([]terminal.MetricReport, 0, len(repoResult.Metrics))
		for name, raw := range repoResult.Metrics {
			dim := metricDimension(name)
			if dim == "" {
				continue
			}
			normalized := scorer.NormalizeMetric(name, raw)
			metricReports = append(metricReports, terminal.MetricReport{
				Name:       name,
				Dimension:  dim,
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
			CollectedAt:        repoResult.CollectedAt,
		})
	}
	return reports
}

func BuildFromDB(store Store, project model.Project, cfg *config.Config, warn io.Writer) ([]terminal.Report, error) {
	weights := scorer.WeightsFromConfig(cfg.Weights)

	repos, err := store.ListRepositories(project.ID)
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	var reports []terminal.Report
	for _, repo := range repos {
		metrics, err := store.GetMetricsByRepository(repo.ID)
		if err != nil {
			return nil, fmt.Errorf("getting metrics for %s: %w", repo.URL, err)
		}
		if len(metrics) == 0 {
			if warn != nil {
				_, _ = fmt.Fprintf(warn, "No metrics collected for %s. Run 'radar collect' first.\n", repo.URL)
			}
			continue
		}

		rawMetrics := make(map[model.MetricTypeName]float64)
		prevMetrics := make(map[model.MetricTypeName]float64)
		var collectedAt time.Time
		for _, m := range metrics {
			mt, err := store.GetMetricTypeByID(m.MetricTypeID)
			if err != nil {
				continue
			}
			if _, ok := metricTypeLookup(mt.Name); !ok {
				continue
			}
			if v, ok := rawMetrics[mt.Name]; ok {
				prevMetrics[mt.Name] = v
			}
			rawMetrics[mt.Name] = m.Value
			if m.CollectedAt.After(collectedAt) {
				collectedAt = m.CollectedAt
			}
		}

		scoreResult := scorer.Score(rawMetrics, weights)

		dimReports := buildDimensionReports(scoreResult.Dimensions, cfg.Weights)

		hasTrend, overallDelta := false, 0.0
		if len(prevMetrics) > 0 {
			prev := scorer.Score(prevMetrics, weights)
			prevByDim := make(map[model.Dimension]float64, len(prev.Dimensions))
			for _, d := range prev.Dimensions {
				prevByDim[d.Dimension] = d.Score
			}
			for i := range dimReports {
				dimReports[i].Delta = dimReports[i].Score - prevByDim[dimReports[i].Dimension]
			}
			hasTrend = true
			overallDelta = scoreResult.Overall - prev.Overall
		}

		metricReports := make([]terminal.MetricReport, 0, len(rawMetrics))
		for name, raw := range rawMetrics {
			mt, ok := metricTypeLookup(name)
			if !ok {
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
			CollectedAt:        collectedAt,
			HasTrend:           hasTrend,
			OverallDelta:       overallDelta,
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
	for _, mt := range append(model.MetricTypes(), model.DisplayMetricTypes()...) {
		if mt.Name == name {
			return mt, true
		}
	}
	return model.MetricType{}, false
}
