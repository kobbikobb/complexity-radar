package config

import (
	"strings"
	"testing"
)

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{
			Name:        "Test Project",
			Description: "A test project",
		},
		Repositories: []RepositoryConfig{
			{URL: "github.com/org/repo", Branch: "develop"},
			{URL: "github.com/org/other-repo", Branch: "main"},
		},
		Weights: WeightsConfig{
			Security:       0.30,
			Delivery:       0.25,
			Infrastructure: 0.25,
			Code:           0.20,
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMissingProjectName(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{
			Description: "Missing name",
		},
		Repositories: []RepositoryConfig{
			{URL: "github.com/org/repo"},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing project name")
	}
	if !strings.Contains(err.Error(), "project.name is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "project.name is required")
	}
}

func TestValidateNoRepositories(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{
			Name: "Test",
		},
		Repositories: []RepositoryConfig{},
		Weights:      DefaultWeights(),
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing repositories")
	}
	if !strings.Contains(err.Error(), "at least one repository is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "at least one repository is required")
	}
}

func TestValidateRepositoryMissingURL(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{
			Name: "Test",
		},
		Repositories: []RepositoryConfig{
			{Branch: "main"},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing repository URL")
	}
	if !strings.Contains(err.Error(), "repositories[0].url is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "repositories[0].url is required")
	}
}

func TestValidateInvalidRepositoryURL(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{
			Name: "Test",
		},
		Repositories: []RepositoryConfig{
			{URL: "not-a-url"},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid repository URL")
	}
	if !strings.Contains(err.Error(), "not a valid repository URL") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not a valid repository URL")
	}
}

func TestValidateInvalidWeightSum(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "sum exceeds 1.0",
			config: Config{
				Project:      ProjectConfig{Name: "Test"},
				Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
				Weights:      WeightsConfig{Security: 0.50, Delivery: 0.50, Infrastructure: 0.50, Code: 0.50},
			},
			wantErr: "weights must sum to 1.0, got 2.00",
		},
		{
			name: "sum below 1.0",
			config: Config{
				Project:      ProjectConfig{Name: "Test"},
				Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
				Weights:      WeightsConfig{Security: 0.10, Delivery: 0.10, Infrastructure: 0.10, Code: 0.10},
			},
			wantErr: "weights must sum to 1.0, got 0.40",
		},
		{
			name: "negative weight",
			config: Config{
				Project:      ProjectConfig{Name: "Test"},
				Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
				Weights:      WeightsConfig{Security: -0.25, Delivery: 0.30, Infrastructure: 0.25, Code: 0.70},
			},
			wantErr: "weights.security must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.config)
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
	w := DefaultWeights()
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

func TestMultipleValidationErrors(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{},
		Repositories: []RepositoryConfig{{}},
	}

	err := Validate(cfg)
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

func TestValidatePartialWeights(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{Name: "Test"},
		Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
		Weights:      WeightsConfig{Security: 0.30},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for partial weights")
	}
	if !strings.Contains(err.Error(), "weights must sum to 1.0") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "weights must sum to 1.0")
	}
}

func TestValidateWeightExceedsOne(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{Name: "Test"},
		Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
		Weights:      WeightsConfig{Security: 1.5},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for weight exceeding 1.0")
	}
	if !strings.Contains(err.Error(), "weights must sum to 1.0") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "weights must sum to 1.0")
	}
}

func TestValidateWhitespaceProjectName(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{Name: "   "},
		Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for whitespace-only project name")
	}
	if !strings.Contains(err.Error(), "project.name is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "project.name is required")
	}
}

func TestValidateWhitespaceRepositoryURL(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{Name: "Test"},
		Repositories: []RepositoryConfig{{URL: "   "}},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for whitespace-only repository URL")
	}
	if !strings.Contains(err.Error(), "repositories[0].url is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "repositories[0].url is required")
	}
}

func TestZeroWeightsAreSkipped(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{Name: "Test"},
		Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
		Weights:      WeightsConfig{Security: 0, Delivery: 0, Infrastructure: 0, Code: 0},
	}

	err := Validate(cfg)
	if err != nil {
		t.Fatalf("zero weights should be valid, got: %v", err)
	}
}

func TestValidateMinimalConfig(t *testing.T) {
	cfg := &Config{
		Project:      ProjectConfig{Name: "Minimal"},
		Repositories: []RepositoryConfig{{URL: "github.com/org/repo"}},
		Weights:      DefaultWeights(),
	}

	err := Validate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
