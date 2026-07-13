package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
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

	s, err := openStore(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	project, err := findOrCreateProject(s, projectName, "")
	if err != nil {
		return err
	}

	cfg, err := buildConfigFromDB(s, project)
	if err != nil {
		return err
	}

	var repos []model.Repository
	for _, repoCfg := range cfg.Repositories {
		repo, err := s.FindOrCreateRepository(project.ID, repoCfg.URL, repoCfg.Branch)
		if err != nil {
			return fmt.Errorf("finding repository %s: %w", repoCfg.URL, err)
		}
		repos = append(repos, *repo)
	}

	src := github.NewSource()

	result, err := collector.Collect(cmd.Context(), cfg, s, project, repos, src, func(e collector.ProgressEvent) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Message)
	})
	if err != nil {
		return fmt.Errorf("collecting: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", result.Project.Name)
	for _, r := range result.Repositories {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRepository: %s (branch: %s)\n", r.Repository.URL, r.Repository.Branch)
		if len(r.Errors) > 0 {
			for _, e := range r.Errors {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Error: %s\n", e)
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Metrics collected: %d\n", len(r.Metrics))
		for _, d := range r.Dimensions {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %.1f\n", d.Dimension, d.Score)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nCollected successfully.\n")
	return nil
}
