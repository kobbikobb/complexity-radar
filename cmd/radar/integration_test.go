package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegrationScan(t *testing.T) {
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("integration test: gh CLI not found")
	}

	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("integration test: gh not authenticated")
	}

	tmpDir := t.TempDir()

	cfgContent := `
[project]
name = "complexity-radar"
description = "Integration test project"

[[repositories]]
url = "github.com/kobbikobb/complexity-radar"
branch = "main"

[weights]
security = 0.25
delivery = 0.30
infrastructure = 0.25
code = 0.20
`
	cfgPath := filepath.Join(tmpDir, ".complexity-radar.toml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", ".", "scan", "--config", cfgPath)
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
