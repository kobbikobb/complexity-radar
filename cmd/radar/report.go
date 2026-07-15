package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/report"
	"github.com/kobbikobb/complexity-radar/internal/runner"
	"github.com/kobbikobb/complexity-radar/internal/terminal"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate complexity report",
	Long: `Calculate scores and generate a complexity report from collected data.
Uses the most recent collection unless specified otherwise.

Run 'radar collect' first to gather data.

Examples:
  radar report              # Report for all projects
  radar report --history    # Show score change since the previous collection
  radar report --explain    # Show how each metric's raw value and score are computed`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().String("db", ".complexity-radar.db", "Database file path")
	reportCmd.Flags().String("project", "", "Project name (default: first project)")
	reportCmd.Flags().Bool("history", false, "Show score change vs the previous collection")
	reportCmd.Flags().Bool("explain", false, "Show each metric's raw definition, scoring function, and source")
}

func runReport(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Flags().GetString("db")
	projectName, _ := cmd.Flags().GetString("project")
	history, _ := cmd.Flags().GetBool("history")
	explain, _ := cmd.Flags().GetBool("explain")

	s, err := openStore(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	project, err := runner.FindOrCreateProject(s, projectName)
	if err != nil {
		return err
	}

	cfg, err := runner.BuildConfigFromDB(s, project)
	if err != nil {
		return err
	}

	formatter := terminal.New()
	formatter.UseColor = true
	formatter.ShowTrend = history
	formatter.ShowExplain = explain

	reports, err := report.BuildFromDB(s, *project, cfg, cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("building report: %w", err)
	}

	if len(reports) == 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No metrics collected. Run 'radar collect' first.\n")
	}

	for _, r := range reports {
		fmt.Println(formatter.Format(r))
	}

	return nil
}
