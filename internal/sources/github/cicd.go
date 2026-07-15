package github

import (
	"context"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

func collectCICDComplexity(workflowContents map[string]string, ctx context.Context, client APIClient, owner, name, branch string) []model.SourceMetric {
	score := 0.0

	// Score GitHub Actions workflows from cached contents
	score += scoreGitHubActions(workflowContents)

	// Score other CI systems (not in .github/workflows/)
	score += scoreOtherCISystems(ctx, client, owner, name, branch)

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return []model.SourceMetric{
		{Type: model.MetricTypeCICDComplexity, Value: score},
	}
}

func scoreGitHubActions(workflowContents map[string]string) float64 {
	score := 0.0

	for _, content := range workflowContents {
		score += scoreWorkflow(content)
	}

	workflowCount := len(workflowContents)
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
