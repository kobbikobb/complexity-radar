package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

func (s *Source) collectCICDComplexity(ctx context.Context, owner, name, branch string) ([]sources.SourceMetric, error) {
	score := 0.0

	// Check GitHub Actions workflows
	workflowScore, err := s.scoreGitHubActions(ctx, owner, name, branch)
	if err == nil {
		score += workflowScore
	}

	// Check for other CI systems
	ciScore, err := s.scoreOtherCISystems(ctx, owner, name, branch)
	if err == nil {
		score += ciScore
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeCICDComplexity, Value: score},
	}, nil
}

func (s *Source) scoreGitHubActions(ctx context.Context, owner, name, branch string) (float64, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/git/trees/%s", owner, name, branch)
	data, err := s.client.GetWithParams(ctx, endpoint, map[string]string{"recursive": "1"})
	if err != nil {
		return 0, err
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
		} `json:"tree"`
	}
	if err := json.Unmarshal(data, &tree); err != nil {
		return 0, err
	}

	score := 0.0
	workflowCount := 0

	for _, item := range tree.Tree {
		if strings.HasPrefix(item.Path, ".github/workflows/") && (strings.HasSuffix(item.Path, ".yml") || strings.HasSuffix(item.Path, ".yaml")) {
			workflowCount++
			content, err := s.client.GetFileContent(ctx, owner, name, item.Path, branch)
			if err != nil {
				continue
			}
			score += scoreWorkflow(content)
		}
	}

	// Bonus for having multiple workflows
	if workflowCount > 3 {
		score += float64(workflowCount-3) * 5
	}

	return score, nil
}

func scoreWorkflow(content string) float64 {
	score := 0.0
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Count jobs
		if strings.HasPrefix(line, "jobs:") {
			score += 10
		}

		// Count steps
		if strings.HasPrefix(line, "- uses:") || strings.HasPrefix(line, "- name:") {
			score += 2
		}

		// Count conditional logic
		if strings.Contains(line, "if:") {
			score += 3
		}

		// Count matrix strategies
		if strings.Contains(line, "matrix:") {
			score += 5
		}

		// Count reusable workflows
		if strings.Contains(line, "uses:") && strings.Contains(line, ".github/workflows/") {
			score += 8
		}

		// Count secrets usage
		if strings.Contains(line, "secrets.") {
			score += 2
		}

		// Count environment variables
		if strings.HasPrefix(line, "env:") {
			score += 1
		}
	}

	return score
}

func (s *Source) scoreOtherCISystems(ctx context.Context, owner, name, branch string) (float64, error) {
	ciFiles := []string{
		".travis.yml",
		".circleci/config.yml",
		"Jenkinsfile",
		".gitlab-ci.yml",
		"azure-pipelines.yml",
		"bitbucket-pipelines.yml",
		"cloudbuild.yaml",
	}

	score := 0.0
	for _, file := range ciFiles {
		_, err := s.client.GetFileContent(ctx, owner, name, file, branch)
		if err == nil {
			score += 15 // Each CI system adds complexity
		}
	}

	return score, nil
}
