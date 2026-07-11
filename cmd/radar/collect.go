package main

import (
	"fmt"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement collection logic
		fmt.Println("Collecting data from configured sources...")
		fmt.Println("(Not yet implemented)")
		return nil
	},
}
