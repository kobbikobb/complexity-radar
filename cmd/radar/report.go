package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/report"
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

	formatter := terminal.New()
	formatter.UseColor = true

	builder := report.NewBuilder()
	reports, err := builder.BuildFromDB(s, *project, cfg)
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
