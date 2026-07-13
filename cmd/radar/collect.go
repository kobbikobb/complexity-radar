package main

import (
	"fmt"
	"os"

	"github.com/kobbikobb/complexity-radar/internal/collector"
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

func runCollect(cmd *cobra.Command, args []string) error {
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

	result, err := collector.Collect(cmd.Context(), cfg, s, src)
	if err != nil {
		return fmt.Errorf("collecting: %w", err)
	}

	fmt.Printf("Project: %s\n", result.Project.Name)
	for _, r := range result.Repositories {
		fmt.Printf("\nRepository: %s (branch: %s)\n", r.Repository.URL, r.Repository.Branch)
		if len(r.Errors) > 0 {
			for _, e := range r.Errors {
				fmt.Fprintf(os.Stderr, "  Error: %s\n", e)
			}
		}
		fmt.Printf("  Metrics collected: %d\n", len(r.Metrics))
		for _, d := range r.Dimensions {
			fmt.Printf("  %s: %.1f\n", d.Dimension, d.Score)
		}
	}

	fmt.Printf("\nCollected successfully.\n")
	return nil
}
