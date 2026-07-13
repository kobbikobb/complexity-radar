package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
)

func TestPrompt(t *testing.T) {
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

func TestPromptEmptyRequired(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	got, err := prompt(reader, "Project name", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("prompt() = %q, want empty string", got)
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

	cfg, err := buildConfigFromDB(s, project)
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

	_, err = buildConfigFromDB(s, project)
	if err == nil {
		t.Fatal("expected error for no repositories")
	}
}
