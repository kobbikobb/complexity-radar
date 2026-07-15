package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/runner"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

func TestPromptWithInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		label    string
		defValue string
		want     string
	}{
		{
			name:     "user provides input",
			input:    "my-project\n",
			label:    "Project name",
			defValue: "",
			want:     "my-project",
		},
		{
			name:     "user uses default",
			input:    "\n",
			label:    "Branch",
			defValue: "main",
			want:     "main",
		},
		{
			name:     "user overrides default",
			input:    "develop\n",
			label:    "Branch",
			defValue: "main",
			want:     "develop",
		},
		{
			name:     "trims whitespace",
			input:    "  my-project  \n",
			label:    "Project name",
			defValue: "",
			want:     "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := prompt(reader, tt.label, tt.defValue)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("prompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromptReturnsEmptyWhenNoDefault(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	got, err := prompt(reader, "Project name", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("prompt() = %q, want empty string", got)
	}
}

func TestPromptProjectRequiresName(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	reader := bufio.NewReader(strings.NewReader("\n"))
	_, err = promptProject(reader, s)
	if err == nil {
		t.Fatal("expected error for empty project name")
	}
	if !strings.Contains(err.Error(), "project name is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildConfigFromDB(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	project := &model.Project{
		Name:        "test-project",
		Description: "A test project",
	}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}

	repo := &model.Repository{
		ProjectID: project.ID,
		URL:       "github.com/org/repo",
		Branch:    "main",
	}
	if err := s.CreateRepository(repo); err != nil {
		t.Fatal(err)
	}

	cfg, err := runner.BuildConfigFromDB(s, project)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got %q", cfg.Project.Name)
	}

	if cfg.Project.Description != "A test project" {
		t.Errorf("expected description 'A test project', got %q", cfg.Project.Description)
	}

	if len(cfg.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(cfg.Repositories))
	}

	if cfg.Repositories[0].URL != "github.com/org/repo" {
		t.Errorf("expected repo URL 'github.com/org/repo', got %q", cfg.Repositories[0].URL)
	}

	if cfg.Repositories[0].Branch != "main" {
		t.Errorf("expected branch 'main', got %q", cfg.Repositories[0].Branch)
	}
}

func TestBuildConfigFromDBNoRepos(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	project := &model.Project{
		Name: "empty-project",
	}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}

	_, err = runner.BuildConfigFromDB(s, project)
	if err == nil {
		t.Fatal("expected error for no repositories")
	}
}

func TestIsValidRepoURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"github.com/org/repo", true},
		{"gitlab.com/group/project", true},
		{"not-a-url", false},
		{"", false},
		{"  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := config.IsValidRepoURL(tt.url)
			if got != tt.want {
				t.Errorf("isValidRepoURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestFindOrCreateProjectCreatesNew(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	p, err := runner.FindOrCreateProject(s, "new-project")
	if err != nil {
		t.Fatal(err)
	}

	if p.Name != "new-project" {
		t.Errorf("expected project name 'new-project', got %q", p.Name)
	}
	if p.ID == 0 {
		t.Error("expected non-zero project ID")
	}
}

func TestFindOrCreateProjectReturnsExisting(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	existing := &model.Project{Name: "existing-project", Description: "original"}
	if err := s.CreateProject(existing); err != nil {
		t.Fatal(err)
	}

	p, err := runner.FindOrCreateProject(s, "existing-project")
	if err != nil {
		t.Fatal(err)
	}

	if p.ID != existing.ID {
		t.Errorf("expected same project ID %d, got %d", existing.ID, p.ID)
	}
	if p.Description != "original" {
		t.Errorf("expected original description, got %q", p.Description)
	}
}
