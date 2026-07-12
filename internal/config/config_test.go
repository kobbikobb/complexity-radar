package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidConfig(t *testing.T) {
	data := []byte(`
[project]
name = "Test Project"
description = "A test project"

[[repositories]]
url = "github.com/org/repo"
branch = "develop"

[[repositories]]
url = "github.com/org/other-repo"
branch = "main"

[weights]
security = 0.30
delivery = 0.25
infrastructure = 0.25
code = 0.20

[thresholds]
security_vulnerabilities_critical_max = 0
build_success_ratio_min = 0.95
stale_pull_requests_max = 5
`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project.Name != "Test Project" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "Test Project")
	}
	if cfg.Project.Description != "A test project" {
		t.Errorf("project description = %q, want %q", cfg.Project.Description, "A test project")
	}
	if len(cfg.Repositories) != 2 {
		t.Fatalf("got %d repositories, want 2", len(cfg.Repositories))
	}
	if cfg.Repositories[0].URL != "github.com/org/repo" {
		t.Errorf("repo[0].url = %q, want %q", cfg.Repositories[0].URL, "github.com/org/repo")
	}
	if cfg.Repositories[0].Branch != "develop" {
		t.Errorf("repo[0].branch = %q, want %q", cfg.Repositories[0].Branch, "develop")
	}
	if cfg.Repositories[1].URL != "github.com/org/other-repo" {
		t.Errorf("repo[1].url = %q, want %q", cfg.Repositories[1].URL, "github.com/org/other-repo")
	}
	if cfg.Weights.Security != 0.30 {
		t.Errorf("weights.security = %v, want 0.30", cfg.Weights.Security)
	}
	if cfg.Weights.Delivery != 0.25 {
		t.Errorf("weights.delivery = %v, want 0.25", cfg.Weights.Delivery)
	}
	if cfg.Weights.Infrastructure != 0.25 {
		t.Errorf("weights.infrastructure = %v, want 0.25", cfg.Weights.Infrastructure)
	}
	if cfg.Weights.Code != 0.20 {
		t.Errorf("weights.code = %v, want 0.20", cfg.Weights.Code)
	}

	if cfg.Thresholds.SecurityVulnerabilitiesCriticalMax == nil {
		t.Fatal("expected security_vulnerabilities_critical_max to be set")
	}
	if *cfg.Thresholds.SecurityVulnerabilitiesCriticalMax != 0 {
		t.Errorf("security_vulnerabilities_critical_max = %d, want 0", *cfg.Thresholds.SecurityVulnerabilitiesCriticalMax)
	}
	if cfg.Thresholds.BuildSuccessRatioMin == nil {
		t.Fatal("expected build_success_ratio_min to be set")
	}
	if *cfg.Thresholds.BuildSuccessRatioMin != 0.95 {
		t.Errorf("build_success_ratio_min = %v, want 0.95", *cfg.Thresholds.BuildSuccessRatioMin)
	}
	if cfg.Thresholds.StalePullRequestsMax == nil {
		t.Fatal("expected stale_pull_requests_max to be set")
	}
	if *cfg.Thresholds.StalePullRequestsMax != 5 {
		t.Errorf("stale_pull_requests_max = %d, want 5", *cfg.Thresholds.StalePullRequestsMax)
	}
}

func TestParseMissingProjectName(t *testing.T) {
	data := []byte(`
[project]
description = "Missing name"

[[repositories]]
url = "github.com/org/repo"
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing project name")
	}
	if !strings.Contains(err.Error(), "project.name is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "project.name is required")
	}
}

func TestParseNoRepositories(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[weights]
security = 0.25
delivery = 0.25
infrastructure = 0.25
code = 0.25
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing repositories")
	}
	if !strings.Contains(err.Error(), "at least one repository is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "at least one repository is required")
	}
}

func TestParseRepositoryMissingURL(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
branch = "main"
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing repository URL")
	}
	if !strings.Contains(err.Error(), "repositories[0].url is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "repositories[0].url is required")
	}
}

func TestParseInvalidRepositoryURL(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "not-a-url"
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid repository URL")
	}
	if !strings.Contains(err.Error(), "not a valid repository URL") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not a valid repository URL")
	}
}

func TestParseInvalidWeightSum(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "sum exceeds 1.0",
			config: `
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = 0.50
delivery = 0.50
infrastructure = 0.50
code = 0.50
`,
			wantErr: "weights must sum to 1.0, got 2.00",
		},
		{
			name: "sum below 1.0",
			config: `
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = 0.10
delivery = 0.10
infrastructure = 0.10
code = 0.10
`,
			wantErr: "weights must sum to 1.0, got 0.40",
		},
		{
			name: "negative weight",
			config: `
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = -0.25
delivery = 0.30
infrastructure = 0.25
code = 0.70
`,
			wantErr: "weights.security must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.config))
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDefaultWeights(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"
`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := cfg.Weights
	if w.Security != 0.25 {
		t.Errorf("default security weight = %v, want 0.25", w.Security)
	}
	if w.Delivery != 0.30 {
		t.Errorf("default delivery weight = %v, want 0.30", w.Delivery)
	}
	if w.Infrastructure != 0.25 {
		t.Errorf("default infrastructure weight = %v, want 0.25", w.Infrastructure)
	}
	if w.Code != 0.20 {
		t.Errorf("default code weight = %v, want 0.20", w.Code)
	}
}

func TestDefaultBranch(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"
`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Repositories[0].Branch != "main" {
		t.Errorf("default branch = %q, want %q", cfg.Repositories[0].Branch, "main")
	}
}

func TestParseInvalidTOML(t *testing.T) {
	data := []byte(`{{{{invalid toml`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
	if !strings.Contains(err.Error(), "invalid TOML") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "invalid TOML")
	}
}

func TestLoadValidFile(t *testing.T) {
	content := `
[project]
name = "File Project"
description = "Loaded from file"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = 0.25
delivery = 0.30
infrastructure = 0.25
code = 0.20
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project.Name != "File Project" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "File Project")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "reading config file")
	}
}

func TestMultipleValidationErrors(t *testing.T) {
	data := []byte(`
[project]

[[repositories]]
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "project.name is required") {
		t.Errorf("error should contain project.name validation, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "repositories[0].url is required") {
		t.Errorf("error should contain repositories[0].url validation, got: %q", errMsg)
	}
}

func TestParsePartialWeights(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = 0.30
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for partial weights")
	}
	if !strings.Contains(err.Error(), "weights must sum to 1.0") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "weights must sum to 1.0")
	}
}

func TestParseWeightExceedsOne(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = 1.5
delivery = 0
infrastructure = 0
code = 0
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for weight exceeding 1.0")
	}
	if !strings.Contains(err.Error(), "weights must sum to 1.0") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "weights must sum to 1.0")
	}
}

func TestParseWhitespaceProjectName(t *testing.T) {
	data := []byte(`
[project]
name = "   "

[[repositories]]
url = "github.com/org/repo"
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for whitespace-only project name")
	}
	if !strings.Contains(err.Error(), "project.name is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "project.name is required")
	}
}

func TestParseWhitespaceRepositoryURL(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "   "
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for whitespace-only repository URL")
	}
	if !strings.Contains(err.Error(), "repositories[0].url is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "repositories[0].url is required")
	}
}

func TestZeroWeightsAreSkipped(t *testing.T) {
	data := []byte(`
[project]
name = "Test"

[[repositories]]
url = "github.com/org/repo"

[weights]
security = 0
delivery = 0
infrastructure = 0
code = 0
`)

	_, err := Parse(data)
	if err != nil {
		t.Fatalf("zero weights should be valid (defaults not applied), got: %v", err)
	}
}

func TestParseMinimalConfig(t *testing.T) {
	data := []byte(`
[project]
name = "Minimal"

[[repositories]]
url = "github.com/org/repo"
`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project.Name != "Minimal" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "Minimal")
	}
	if cfg.Project.Description != "" {
		t.Errorf("project description = %q, want empty", cfg.Project.Description)
	}
	if cfg.Weights != DefaultWeights() {
		t.Errorf("weights should be defaults")
	}
}

func TestLoadExampleConfig(t *testing.T) {
	cfg, err := Load("../../configs/complexity-radar.example.toml")
	if err != nil {
		t.Fatalf("failed to load example config: %v", err)
	}

	if cfg.Project.Name != "My App" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "My App")
	}
	if len(cfg.Repositories) != 1 {
		t.Fatalf("got %d repositories, want 1", len(cfg.Repositories))
	}
	if cfg.Repositories[0].URL != "github.com/org/repo" {
		t.Errorf("repo URL = %q, want %q", cfg.Repositories[0].URL, "github.com/org/repo")
	}
	if cfg.Repositories[0].Branch != "main" {
		t.Errorf("repo branch = %q, want %q", cfg.Repositories[0].Branch, "main")
	}
}
