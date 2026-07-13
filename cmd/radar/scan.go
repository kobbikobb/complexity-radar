package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/output"
	"github.com/kobbikobb/complexity-radar/internal/output/terminal"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
	"github.com/kobbikobb/complexity-radar/internal/store"
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

func runScan(cmd *cobra.Command, args []string) error {
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

	src := github.NewSource()

	fmt.Println("Collecting data...")
	result, err := collector.Collect(cmd.Context(), cfg, s, src)
	if err != nil {
		return fmt.Errorf("collecting: %w", err)
	}

	fmt.Println("Generating report...")
	formatter := terminal.New()
	formatter.UseColor = true

	for _, repoResult := range result.Repositories {
		if len(repoResult.Errors) > 0 {
			for _, e := range repoResult.Errors {
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Warning: %s\n", e)
			}
		}

		dimReports := make([]output.DimensionReport, len(repoResult.Dimensions))
		for i, d := range repoResult.Dimensions {
			dimReports[i] = output.DimensionReport{
				Dimension:   d.Dimension,
				Score:       d.Score,
				Weight:      cfg.Weights.Weight(string(d.Dimension)) * 100,
				MetricCount: d.MetricCount,
			}
		}

		metricReports := make([]output.MetricReport, 0, len(repoResult.Metrics))
		for name, raw := range repoResult.Metrics {
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
