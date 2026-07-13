package github

import (
	"context"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

func collectCICDComplexity(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree) []model.SourceMetric {
	score := 0.0

	// Score GitHub Actions workflows
	score += scoreGitHubActions(ctx, client, owner, name, branch, tree)

	// Score other CI systems
	score += scoreOtherCISystems(ctx, client, owner, name, branch)

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return []model.SourceMetric{
		{Type: model.MetricTypeCICDComplexity, Value: score},
	}
}

func scoreGitHubActions(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree) float64 {
	score := 0.0
	workflowCount := 0

	for _, entry := range tree.Tree {
		if strings.HasPrefix(entry.Path, ".github/workflows/") &&
			(strings.HasSuffix(entry.Path, ".yml") || strings.HasSuffix(entry.Path, ".yaml")) {
			workflowCount++
			content, err := client.GetFileContent(ctx, owner, name, entry.Path, branch)
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

	return score
}

func scoreWorkflow(content string) float64 {
	score := 0.0

	// Count jobs (each job adds complexity)
	score += float64(strings.Count(content, "jobs:")) * 10

	// Count steps (each step adds complexity)
	score += float64(strings.Count(content, "- uses:")) * 2
	score += float64(strings.Count(content, "- name:")) * 2

	// Count conditional logic
	score += float64(strings.Count(content, "if:")) * 3

	// Count matrix strategies
	score += float64(strings.Count(content, "matrix:")) * 5

	// Count reusable workflows
	score += float64(strings.Count(content, ".github/workflows/")) * 8

	// Count secrets usage
	score += float64(strings.Count(content, "secrets.")) * 2
	score += float64(strings.Count(content, "secrets:")) * 2

	// Count environment variables
	score += float64(strings.Count(content, "env:")) * 1

	return score
}

func scoreOtherCISystems(ctx context.Context, client APIClient, owner, name, branch string) float64 {
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
		_, err := client.GetFileContent(ctx, owner, name, file, branch)
		if err == nil {
			score += 15
		}
	}

	return score
}
