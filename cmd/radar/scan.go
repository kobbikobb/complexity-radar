package main

import (
	"fmt"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement scan logic (collect + report)
		fmt.Println("Running full scan...")
		fmt.Println("  1. Collecting data from configured sources...")
		fmt.Println("  2. Generating complexity report...")
		fmt.Println("(Not yet implemented)")
		return nil
	},
}
