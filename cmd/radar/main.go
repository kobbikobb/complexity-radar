package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "radar",
	Short: "Technical complexity scoring for software projects",
	Long: `ComplexityRadar helps you understand and measure the complexity 
of your software projects. It pulls data from multiple sources,
calculates weighted complexity scores, and tracks how your scores
evolve over time.

Get started:
  radar scan          # Collect data and generate report
  radar collect       # Pull data from configured sources
  radar report        # Generate complexity report

For more information, visit: https://github.com/kobbikobb/complexity-radar`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("ComplexityRadar %s\n", version)
	},
}
