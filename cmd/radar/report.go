package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate complexity report",
	Long: `Calculate scores and generate a complexity report from collected data.
Uses the most recent collection unless specified otherwise.

Examples:
  radar report                      # Report for all projects
  radar report --project "My App"  # Report for specific project
  radar report --output json       # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement reporting logic
		fmt.Println("Generating complexity report...")
		fmt.Println("(Not yet implemented)")
		return nil
	},
}
