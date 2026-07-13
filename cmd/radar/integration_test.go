package main

import (
	"os/exec"
	"path/filepath"
	"strings"
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

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, ".complexity-radar.db")

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}

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

	cmd := exec.Command("go", "run", ".", "scan", "--db", dbPath)
	out, err := cmd.CombinedOutput()
	t.Logf("Output:\n%s", out)

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "Collecting from") {
		t.Error("missing collection progress message")
	}
	if !strings.Contains(output, "Generating report...") {
		t.Error("missing report progress message")
	}
	if !strings.Contains(output, "OVERALL SCORE:") {
		t.Error("missing overall score in output")
	}
	if !strings.Contains(output, "Dimension Scores:") {
		t.Error("missing dimension scores")
	}
	if !strings.Contains(output, "Metric Details:") {
		t.Error("missing metric details")
	}
}
