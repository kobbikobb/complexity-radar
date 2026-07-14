package github

import (
	"context"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

func collectDeployTargets(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree) []sources.SourceMetric {
	targets := make(map[string]bool)
	workflowCount := 0

	for _, entry := range tree.Tree {
		if strings.HasPrefix(entry.Path, ".github/workflows/") &&
			(strings.HasSuffix(entry.Path, ".yml") || strings.HasSuffix(entry.Path, ".yaml")) {
			workflowCount++
			content, err := client.GetFileContent(ctx, owner, name, entry.Path, branch)
			if err != nil {
				continue
			}
			parseWorkflowEnvironments(content, targets)
		}
	}

	collectTargetsFromDeployConfigs(ctx, client, owner, name, branch, targets)

	divisor := workflowCount
	if divisor == 0 {
		divisor = 1
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeDeployTargets, Value: float64(len(targets)) / float64(divisor)},
	}
}

func collectTargetsFromDeployConfigs(ctx context.Context, client APIClient, owner, name, branch string, targets map[string]bool) {
	configFiles := []string{
		"appspec.json",
		"appspec.yml",
		"imagedefinitions.json",
		"buildspec.yml",
	}

	for _, file := range configFiles {
		content, err := client.GetFileContent(ctx, owner, name, file, branch)
		if err != nil {
			continue
		}
		// Only count files that look like actual deploy configs
		if strings.Contains(content, "environment") || strings.Contains(content, "deploy") {
			targets[file] = true
		}
	}
}

// parseWorkflowEnvironments extracts environment names from GitHub Actions workflow YAML.
func parseWorkflowEnvironments(content string, targets map[string]bool) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for environment: lines (e.g., "environment: production")
		if strings.HasPrefix(line, "environment:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				env := strings.TrimSpace(parts[1])
				env = strings.Trim(env, "\"'")
				if env != "" {
					targets[env] = true
				}
			}
		}
	}
}
