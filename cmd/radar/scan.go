package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/output"
	"github.com/kobbikobb/complexity-radar/internal/output/terminal"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Collect data and generate report",
	Long: `Scan performs both collection and reporting in a single command.
This is equivalent to running 'radar collect' followed by 'radar report'.

Run 'radar init' first to configure your project.

Example:
  radar scan    # Full scan`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().String("db", ".complexity-radar.db", "Database file path")
	scanCmd.Flags().String("project", "", "Project name (default: first project)")
}

func runScan(cmd *cobra.Command, args []string) error {
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

	src := github.NewSource()

	result, err := collector.Collect(cmd.Context(), cfg, s, src, func(e collector.ProgressEvent) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Message)
	})
	if err != nil {
		return fmt.Errorf("collecting: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Generating report...")
	formatter := terminal.New()
	formatter.UseColor = true

	for _, repoResult := range result.Repositories {
		if len(repoResult.Errors) > 0 {
			for _, e := range repoResult.Errors {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: %s\n", e)
			}
		}

		dimReports := make([]output.DimensionReport, len(repoResult.Dimensions))
		for i, d := range repoResult.Dimensions {
			dr := output.DimensionReport{
				Dimension:   d.Dimension,
				Score:       d.Score,
				Weight:      cfg.Weights.Weight(string(d.Dimension)) * 100,
				MetricCount: d.MetricCount,
			}
			if d.Dimension == model.DimensionSecurity {
				dr.Breakdown = output.SecurityBreakdown(repoResult.Metrics)
			}
			dimReports[i] = dr
		}

		metricReports := make([]output.MetricReport, 0, len(repoResult.Metrics))
		for name, raw := range repoResult.Metrics {
			mt, err := s.GetMetricTypeByName(name)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: unknown metric %s, skipping\n", name)
				continue
			}
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
			ProjectName:        result.Project.Name,
			ProjectDescription: result.Project.Description,
			OverallScore:       repoResult.OverallScore,
			Dimensions:         dimReports,
			Metrics:            metricReports,
			CollectedAt:        repoResult.Repository.CreatedAt,
		}

		fmt.Println(formatter.Format(report))
	}

	return nil
}
