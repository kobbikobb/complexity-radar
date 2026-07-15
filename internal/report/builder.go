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
		reports = append(reports, buildRepoReport(
			result.Project.Name,
			result.Project.Description,
			repoResult.OverallScore,
			repoResult.Dimensions,
			repoResult.Metrics,
			repoResult.CollectedAt,
			cfg.Weights,
			0, false, nil,
		))
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

		rawMetrics, prevMetrics, collectedAt := partitionMetrics(store, metrics)

		scoreResult := scorer.Score(rawMetrics, weights)

		hasTrend, overallDelta := false, 0.0
		var dimDeltas map[model.Dimension]float64
		if len(prevMetrics) > 0 {
			prev := scorer.Score(prevMetrics, weights)
			overallDelta = scoreResult.Overall - prev.Overall
			hasTrend = true
			prevByDim := make(map[model.Dimension]float64, len(prev.Dimensions))
			for _, d := range prev.Dimensions {
				prevByDim[d.Dimension] = d.Score
			}
			dimDeltas = make(map[model.Dimension]float64, len(scoreResult.Dimensions))
			for _, d := range scoreResult.Dimensions {
				dimDeltas[d.Dimension] = d.Score - prevByDim[d.Dimension]
			}
		}

		reports = append(reports, buildRepoReport(
			project.Name,
			project.Description,
			scoreResult.Overall,
			scoreResult.Dimensions,
			rawMetrics,
			collectedAt,
			cfg.Weights,
			overallDelta,
			hasTrend,
			dimDeltas,
		))
	}

	return reports, nil
}

func buildRepoReport(projectName, projectDesc string, overallScore float64, dimensions []scorer.DimensionResult, rawMetrics map[model.MetricTypeName]float64, collectedAt time.Time, weights config.WeightsConfig, overallDelta float64, hasTrend bool, dimDeltas map[model.Dimension]float64) terminal.Report {
	dimReports := buildDimensionReports(dimensions, weights)

	if dimDeltas != nil {
		for i := range dimReports {
			dimReports[i].Delta = dimDeltas[dimReports[i].Dimension]
		}
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

	return terminal.Report{
		ProjectName:        projectName,
		ProjectDescription: projectDesc,
		OverallScore:       overallScore,
		Dimensions:         dimReports,
		Metrics:            metricReports,
		CollectedAt:        collectedAt,
		HasTrend:           hasTrend,
		OverallDelta:       overallDelta,
	}
}

func partitionMetrics(store Store, metrics []model.Metric) (raw, prev map[model.MetricTypeName]float64, collectedAt time.Time) {
	raw = make(map[model.MetricTypeName]float64)
	prev = make(map[model.MetricTypeName]float64)
	for _, m := range metrics {
		mt, err := store.GetMetricTypeByID(m.MetricTypeID)
		if err != nil {
			continue
		}
		if _, ok := metricTypeLookup(mt.Name); !ok {
			continue
		}
		if v, ok := raw[mt.Name]; ok {
			prev[mt.Name] = v
		}
		raw[mt.Name] = m.Value
		if m.CollectedAt.After(collectedAt) {
			collectedAt = m.CollectedAt
		}
	}
	return raw, prev, collectedAt
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

func metricTypeLookup(name model.MetricTypeName) (model.MetricType, bool) {
	for _, mt := range append(model.MetricTypes(), model.DisplayMetricTypes()...) {
		if mt.Name == name {
			return mt, true
		}
	}
	return model.MetricType{}, false
}
