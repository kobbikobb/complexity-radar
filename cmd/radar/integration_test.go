package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

func TestIntegrationScan(t *testing.T) {
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("integration test: gh CLI not found")
	}

	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("integration test: gh not authenticated")
	}

	// Create database in current directory for the scan command
	dbPath := ".complexity-radar.db"
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = s.Close()
		_ = os.Remove(dbPath)
	}()

	project := &model.Project{
		Name:        "complexity-radar",
		Description: "Integration test project",
	}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}

	repo := &model.Repository{
		ProjectID: project.ID,
		URL:       "github.com/kobbikobb/complexity-radar",
		Branch:    "main",
	}
	if err := s.CreateRepository(repo); err != nil {
		t.Fatal(err)
	}

	_ = s.Close()

	cmd := exec.Command("go", "run", ".", "scan")
	out, err := cmd.CombinedOutput()
	t.Logf("Output:\n%s", out)

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	output := string(out)
	if !contains(output, "Collecting data...") {
		t.Error("missing collection progress message")
	}
	if !contains(output, "Generating report...") {
		t.Error("missing report progress message")
	}
	if !contains(output, "OVERALL SCORE:") {
		t.Error("missing overall score in output")
	}
	if !contains(output, "Dimension Scores:") {
		t.Error("missing dimension scores")
	}
	if !contains(output, "Metric Details:") {
		t.Error("missing metric details")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
