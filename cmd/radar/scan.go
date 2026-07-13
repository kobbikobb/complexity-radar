package main

import (
	"fmt"
	"path/filepath"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/output"
	"github.com/kobbikobb/complexity-radar/internal/output/terminal"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Collect data and generate report (shorthand)",
	Long: `Scan performs both collection and reporting in a single command.
This is equivalent to running 'radar collect' followed by 'radar report'.

Examples:
  radar scan                       # Full scan with default config
  radar scan --config my.toml     # Use specific config file
  radar scan --output json        # Output as JSON`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringP("config", "c", "", "Config file path (default: .complexity-radar.toml)")
}

func runScan(cmd *cobra.Command, args []string) error {
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
	defer s.Close()

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
				fmt.Fprintf(cmd.OutOrStderr(), "  Warning: %s\n", e)
			}
		}

		dimReports := make([]output.DimensionReport, len(repoResult.Dimensions))
		for i, d := range repoResult.Dimensions {
			dimReports[i] = output.DimensionReport{
				Dimension:   d.Dimension,
				Score:       d.Score,
				Weight:      cfg.Weights.Weight(string(d.Dimension)),
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
