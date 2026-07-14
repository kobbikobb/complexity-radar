package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/output"
	"github.com/kobbikobb/complexity-radar/internal/output/terminal"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
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

func init() {
	reportCmd.Flags().String("db", ".complexity-radar.db", "Database file path")
	reportCmd.Flags().String("project", "", "Project name (default: first project)")
}

func runReport(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Flags().GetString("db")
	projectName, _ := cmd.Flags().GetString("project")

	s, err := openStore(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	project, err := findProject(s, projectName)
	if err != nil {
		return err
	}

	cfg, err := buildConfigFromDB(s, project)
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No metrics collected for %s. Run 'radar collect' first.\n", repo.URL)
			continue
		}

		rawMetrics := make(map[model.MetricTypeName]float64)
		for _, m := range metrics {
			mt, err := s.GetMetricTypeByID(m.MetricTypeID)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: unknown metric type %d, skipping\n", m.MetricTypeID)
				continue
			}
			rawMetrics[mt.Name] = m.Value
		}

		scoreResult := scorer.Score(rawMetrics, weights)

		dimReports := make([]output.DimensionReport, len(scoreResult.Dimensions))
		for i, d := range scoreResult.Dimensions {
			dr := output.DimensionReport{
				Dimension:   d.Dimension,
				Score:       d.Score,
				Weight:      cfg.Weights.Weight(string(d.Dimension)) * 100,
				MetricCount: d.MetricCount,
			}
			if d.Dimension == model.DimensionSecurity {
				dr.Breakdown = output.SecurityBreakdown(rawMetrics)
			}
			dimReports[i] = dr
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
