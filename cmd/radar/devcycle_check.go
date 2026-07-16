package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/runner"
	"github.com/kobbikobb/complexity-radar/internal/sources/devcycle"
	"github.com/spf13/cobra"
)

var devcycleCheckCmd = &cobra.Command{
	Use:   "devcycle-check",
	Short: "Verify DevCycle integration",
	Long: `Test DevCycle connectivity and credential validity.

Logs request/response details to help diagnose issues.
Sensitive values (client secret) are shown as length only.`,
	RunE: runDevCycleCheck,
}

func init() {
	devcycleCheckCmd.Flags().String("db", ".complexity-radar.db", "Database file path")
	devcycleCheckCmd.Flags().String("project", "", "Project name (default: first project)")
	rootCmd.AddCommand(devcycleCheckCmd)
}

func runDevCycleCheck(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Flags().GetString("db")
	projectName, _ := cmd.Flags().GetString("project")

	s, err := openStore(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	project, err := runner.FindOrCreateProject(s, projectName)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()

	_, _ = fmt.Fprintf(out, "Project:          %s\n", project.Name)
	_, _ = fmt.Fprintf(out, "DevCycle key:     %s\n", project.DevCycleProjectKey)
	_, _ = fmt.Fprintf(out, "Client ID:        %s (len=%d)\n", maskID(project.DevCycleClientID), len(project.DevCycleClientID))
	_, _ = fmt.Fprintf(out, "Client secret:    len=%d\n", len(project.DevCycleClientSecret))
	_, _ = fmt.Fprintln(out)

	if project.DevCycleProjectKey == "" || project.DevCycleClientID == "" || project.DevCycleClientSecret == "" {
		_, _ = fmt.Fprintln(out, "DevCycle not configured. Run 'radar init' and enable DevCycle.")
		return nil
	}

	transport := &loggingTransport{inner: http.DefaultTransport, out: out}
	httpClient := &http.Client{Transport: transport}

	client := devcycle.NewClient(project.DevCycleClientID, project.DevCycleClientSecret, httpClient)

	_, _ = fmt.Fprintln(out, "Authenticating...")
	features, err := client.ListFeatures(context.Background(), project.DevCycleProjectKey)
	if err != nil {
		_, _ = fmt.Fprintf(out, "\nError: %v\n", err)
		_, _ = fmt.Fprintln(out, "\nTroubleshooting:")
		_, _ = fmt.Fprintln(out, "  - Verify client ID and secret are correct")
		_, _ = fmt.Fprintln(out, "  - Verify the project key exists in DevCycle")
		_, _ = fmt.Fprintln(out, "  - Check network connectivity to auth.devcycle.com and api.devcycle.com")
		return nil
	}

	stale := 0
	for _, f := range features {
		if f.Status == "active" && f.Staleness != nil {
			stale++
		}
	}

	_, _ = fmt.Fprintln(out, "Success!")
	_, _ = fmt.Fprintf(out, "Features found:   %d\n", len(features))
	_, _ = fmt.Fprintf(out, "Stale (active):   %d\n", stale)

	return nil
}

func maskID(id string) string {
	if len(id) <= 8 {
		return strings.Repeat("*", len(id))
	}
	return id[:4] + strings.Repeat("*", len(id)-8) + id[len(id)-4:]
}

type loggingTransport struct {
	inner http.RoundTripper
	out   io.Writer
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	_, _ = fmt.Fprintf(t.out, "→ %s %s\n", req.Method, req.URL)

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		_, _ = fmt.Fprintf(t.out, "← Error: %v\n", err)
		return resp, err
	}

	_, _ = fmt.Fprintf(t.out, "← %s (content-length: %s)\n", resp.Status, resp.Header.Get("Content-Length"))
	return resp, nil
}
