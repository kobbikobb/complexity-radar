package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/output"
	"github.com/kobbikobb/complexity-radar/internal/output/terminal"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate complexity report",
	Long: `Calculate scores and generate a complexity report from collected data.
Uses the most recent collection unless specified otherwise.

Run 'radar collect' first to gather data.

Examples:
  radar report    # Report for all projects`,
	RunE: runReport,
}

func runReport(cmd *cobra.Command, args []string) error {
	s, err := store.New(".complexity-radar.db")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = s.Close() }()

	projects, err := s.ListProjects()
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}
	if len(projects) == 0 {
		return fmt.Errorf("no project configured. Run 'radar init' first")
	}

	project := projects[0]

	cfg, err := buildConfigFromDB(s, &project)
	if err != nil {
		return err
	}

	weights := scorer.WeightsFromConfig(cfg.Weights)
	formatter := terminal.New()
	formatter.UseColor = true

	repos, err := s.ListRepositories(project.ID)
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	for _, repo := range repos {
		metrics, err := s.GetMetricsByRepository(repo.ID)
		if err != nil {
			return fmt.Errorf("getting metrics for %s: %w", repo.URL, err)
		}

		if len(metrics) == 0 {
			fmt.Printf("No metrics collected for %s. Run 'radar collect' first.\n", repo.URL)
			continue
		}

		rawMetrics := make(map[model.MetricTypeName]float64)
		for _, m := range metrics {
			mt, err := s.GetMetricTypeByID(m.MetricTypeID)
			if err != nil {
				continue
			}
			rawMetrics[mt.Name] = m.Value
		}

		scoreResult := scorer.Score(rawMetrics, weights)

		dimReports := make([]output.DimensionReport, len(scoreResult.Dimensions))
		for i, d := range scoreResult.Dimensions {
			dimReports[i] = output.DimensionReport{
				Dimension:   d.Dimension,
				Score:       d.Score,
				Weight:      cfg.Weights.Weight(string(d.Dimension)) * 100,
				MetricCount: d.MetricCount,
			}
		}

		metricReports := make([]output.MetricReport, 0, len(rawMetrics))
		for name, raw := range rawMetrics {
			mt, _ := s.GetMetricTypeByName(name)
			normalized := scorer.NormalizeMetric(name, raw)
			metricReports = append(metricReports, output.MetricReport{
				Name:       name,
				Dimension:  mt.Dimension,
				RawValue:   raw,
				Normalized: normalized,
				Unit:       mt.Unit,
			})
		}

		report := output.Report{
			ProjectName:        project.Name,
			ProjectDescription: project.Description,
			OverallScore:       scoreResult.Overall,
			Dimensions:         dimReports,
			Metrics:            metricReports,
		}

		fmt.Println(formatter.Format(report))
	}

	return nil
}
