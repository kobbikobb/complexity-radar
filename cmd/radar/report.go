package main

import (
	"fmt"
	"path/filepath"

	"github.com/kobbikobb/complexity-radar/internal/config"
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

Examples:
  radar report                      # Report for all projects
  radar report --project "My App"  # Report for specific project
  radar report --output json       # Output as JSON`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringP("config", "c", "", "Config file path (default: .complexity-radar.toml)")
}

func runReport(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	if cfgPath == "" {
		cfgPath = ".complexity-radar.toml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dbPath := filepath.Join(filepath.Dir(cfgPath), ".complexity-radar.db")
	s, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = s.Close() }()

	projects, err := s.ListProjects()
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	var project *model.Project
	for _, p := range projects {
		if p.Name == cfg.Project.Name {
			project = &p
			break
		}
	}

	if project == nil {
		return fmt.Errorf("project %q not found. Run 'radar collect' first", cfg.Project.Name)
	}

	repos, err := s.ListRepositories(project.ID)
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories found for project %q", project.Name)
	}

	weights := scorer.WeightsFromConfig(cfg.Weights)
	formatter := terminal.New()
	formatter.UseColor = true

	for _, repo := range repos {
		metrics, err := s.GetMetricsByRepository(repo.ID)
		if err != nil {
			return fmt.Errorf("getting metrics for %s: %w", repo.URL, err)
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
