package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kobbikobb/complexity-radar/internal/collector"
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/spf13/cobra"
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Pull data from configured sources",
	Long: `Collect metrics from all configured repositories and sources.
Data is stored locally in SQLite for later reporting.

Examples:
  radar collect                    # Collect from all configured repos
  radar collect --config my.toml  # Use specific config file`,
	RunE: runCollect,
}

func init() {
	collectCmd.Flags().StringP("config", "c", "", "Config file path (default: .complexity-radar.toml)")
}

func runCollect(cmd *cobra.Command, args []string) error {
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
	defer func() { _ = s.Close() }()

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
