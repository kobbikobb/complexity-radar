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
	GetProjectMetrics(projectID int64) ([]model.ProjectMetric, error)
}

func BuildFromResult(result collector.CollectionResult, cfg *config.Config) []terminal.Report {
	typesByName := buildMetricTypeMaps()

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
			typesByName,
		))
	}
	return reports
}

// BuildProjectReport builds the project-level rollup: each repo metric is
// averaged across repos (skipping no-data sentinels), project-scoped metrics
// like feature_flag_debt are added directly, and the combined bag is scored so
// project-level metrics fold into the project's overall score.
func BuildProjectReport(result collector.CollectionResult, cfg *config.Config) terminal.Report {
	typesByName := buildMetricTypeMaps()
	weights := scorer.WeightsFromConfig(cfg.Weights)

	var repoSets []map[model.MetricTypeName]float64
	var collectedAt time.Time
	for _, rr := range result.Repositories {
		repoSets = append(repoSets, rr.Metrics)
		if rr.CollectedAt.After(collectedAt) {
			collectedAt = rr.CollectedAt
		}
	}

	bag := aggregateMetrics(repoSets, result.ProjectMetrics)
	return buildProjectReportView(result.Project, cfg, weights, bag, collectedAt, result.ProjectErrors, typesByName)
}

// BuildProjectReportFromDB builds the project rollup from stored metrics.
func BuildProjectReportFromDB(store Store, project model.Project, cfg *config.Config) (terminal.Report, error) {
	typesByName := buildMetricTypeMaps()
	weights := scorer.WeightsFromConfig(cfg.Weights)

	repos, err := store.ListRepositories(project.ID)
	if err != nil {
		return terminal.Report{}, fmt.Errorf("listing repositories: %w", err)
	}

	var repoSets []map[model.MetricTypeName]float64
	var collectedAt time.Time
	for _, repo := range repos {
		metrics, err := store.GetMetricsByRepository(repo.ID)
		if err != nil {
			return terminal.Report{}, fmt.Errorf("getting metrics for %s: %w", repo.URL, err)
		}
		raw, _, at := partitionMetrics(store, metrics, typesByName)
		if len(raw) == 0 {
			continue
		}
		repoSets = append(repoSets, raw)
		if at.After(collectedAt) {
			collectedAt = at
		}
	}

	projectMetrics, err := store.GetProjectMetrics(project.ID)
	if err != nil {
		return terminal.Report{}, fmt.Errorf("getting project metrics: %w", err)
	}
	projMap := make(map[model.MetricTypeName]float64)
	for _, pm := range projectMetrics {
		mt, err := store.GetMetricTypeByID(pm.MetricTypeID)
		if err != nil {
			continue
		}
		projMap[mt.Name] = pm.Value
		if pm.CollectedAt.After(collectedAt) {
			collectedAt = pm.CollectedAt
		}
	}

	bag := aggregateMetrics(repoSets, projMap)
	return buildProjectReportView(project, cfg, weights, bag, collectedAt, nil, typesByName), nil
}

func aggregateMetrics(repoSets []map[model.MetricTypeName]float64, projectMetrics map[model.MetricTypeName]float64) map[model.MetricTypeName]float64 {
	sums := make(map[model.MetricTypeName]float64)
	counts := make(map[model.MetricTypeName]int)
	for _, set := range repoSets {
		for name, v := range set {
			if v < 0 {
				continue
			}
			sums[name] += v
			counts[name]++
		}
	}

	bag := make(map[model.MetricTypeName]float64, len(sums)+len(projectMetrics))
	for name, sum := range sums {
		bag[name] = sum / float64(counts[name])
	}
	for name, v := range projectMetrics {
		bag[name] = v
	}
	return bag
}

func buildProjectReportView(project model.Project, cfg *config.Config, weights map[model.Dimension]float64, bag map[model.MetricTypeName]float64, collectedAt time.Time, errs []string, typesByName map[model.MetricTypeName]model.MetricType) terminal.Report {
	scoreResult := scorer.Score(bag, weights)
	report := buildRepoReport(project.Name, project.Description, scoreResult.Overall, scoreResult.Dimensions, bag, collectedAt, cfg.Weights, 0, false, nil, typesByName)
	report.Aggregate = true
	report.Errors = errs
	return report
}

func BuildFromDB(store Store, project model.Project, cfg *config.Config, warn io.Writer) ([]terminal.Report, error) {
	weights := scorer.WeightsFromConfig(cfg.Weights)
	typesByName := buildMetricTypeMaps()

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

		rawMetrics, prevMetrics, collectedAt := partitionMetrics(store, metrics, typesByName)

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
			typesByName,
		))
	}

	return reports, nil
}

func buildRepoReport(projectName, projectDesc string, overallScore float64, dimensions []scorer.DimensionResult, rawMetrics map[model.MetricTypeName]float64, collectedAt time.Time, weights config.WeightsConfig, overallDelta float64, hasTrend bool, dimDeltas map[model.Dimension]float64, typesByName map[model.MetricTypeName]model.MetricType) terminal.Report {
	dimReports := buildDimensionReports(dimensions, weights)

	if dimDeltas != nil {
		for i := range dimReports {
			dimReports[i].Delta = dimDeltas[dimReports[i].Dimension]
		}
	}

	metricReports := make([]terminal.MetricReport, 0, len(rawMetrics))
	for name, raw := range rawMetrics {
		mt, ok := typesByName[name]
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
			Weight:     mt.Weight,
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

func partitionMetrics(store Store, metrics []model.Metric, typesByName map[model.MetricTypeName]model.MetricType) (raw, prev map[model.MetricTypeName]float64, collectedAt time.Time) {
	raw = make(map[model.MetricTypeName]float64)
	prev = make(map[model.MetricTypeName]float64)
	for _, m := range metrics {
		mt, err := store.GetMetricTypeByID(m.MetricTypeID)
		if err != nil {
			continue
		}
		if _, ok := typesByName[mt.Name]; !ok {
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
	weightSums := make(map[model.Dimension]float64)
	for _, mt := range model.MetricTypes() {
		w := mt.Weight
		if w <= 0 {
			w = 1.0
		}
		weightSums[mt.Dimension] += w
	}

	dimReports := make([]terminal.DimensionReport, len(dimensions))
	for i, d := range dimensions {
		dimReports[i] = terminal.DimensionReport{
			Dimension:   d.Dimension,
			Score:       d.Score,
			Weight:      cfg.Weight(string(d.Dimension)) * 100,
			WeightSum:   weightSums[d.Dimension],
			MetricCount: d.MetricCount,
		}
	}
	return dimReports
}

func buildMetricTypeMaps() map[model.MetricTypeName]model.MetricType {
	m := make(map[model.MetricTypeName]model.MetricType, 16)
	for _, mt := range model.MetricTypes() {
		m[mt.Name] = mt
	}
	for _, mt := range model.DisplayMetricTypes() {
		m[mt.Name] = mt
	}
	return m
}
