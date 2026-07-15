package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/runner"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/spf13/cobra"
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Pull data from configured sources",
	Long: `Collect metrics from all configured repositories and sources.
Data is stored locally in SQLite for later reporting.

Run 'radar init' first to configure your project.

Examples:
  radar collect    # Collect from all configured repos`,
	RunE: runCollect,
}

func init() {
	collectCmd.Flags().String("db", ".complexity-radar.db", "Database file path")
	collectCmd.Flags().String("project", "", "Project name (default: first project)")
}

func runCollect(cmd *cobra.Command, args []string) error {
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

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", result.Project.Name)
	for _, repoResult := range result.Repositories {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRepository: %s (branch: %s)\n", repoResult.Repository.URL, repoResult.Repository.Branch)
		if len(repoResult.Errors) > 0 {
			for _, e := range repoResult.Errors {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Error: %s\n", e)
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Metrics collected: %d\n", len(repoResult.Metrics))
		for _, d := range repoResult.Dimensions {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %.1f\n", d.Dimension, d.Score)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nCollected successfully.\n")
	return nil
}
