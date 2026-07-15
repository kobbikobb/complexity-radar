package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/report"
	"github.com/kobbikobb/complexity-radar/internal/runner"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/kobbikobb/complexity-radar/internal/terminal"
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

	s, err := store.New(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	r, err := runner.NewFromStore(s, projectName, github.NewSource())
	if err != nil {
		return err
	}

	result, err := r.Run(cmd.Context(), func(e collector.ProgressEvent) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Message)
	})
	if err != nil {
		return fmt.Errorf("collecting: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Generating report...")
	formatter := terminal.New()
	formatter.UseColor = true

	reports := report.BuildFromResult(*result, r.Config())
	for _, rpt := range reports {
		fmt.Println(formatter.Format(rpt))
	}

	return nil
}
