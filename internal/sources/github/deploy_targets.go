package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

func (s *Source) collectDeployTargets(ctx context.Context, owner, name, branch string) ([]sources.SourceMetric, error) {
	targets := make(map[string]bool)

	// Check GitHub Actions workflows
	s.collectTargetsFromWorkflows(ctx, owner, name, branch, targets)

	// Check common deploy config files
	s.collectTargetsFromDeployConfigs(ctx, owner, name, branch, targets)

	return []sources.SourceMetric{
		{Type: model.MetricTypeDeployTargets, Value: float64(len(targets))},
	}, nil
}

func (s *Source) collectTargetsFromWorkflows(ctx context.Context, owner, name, branch string, targets map[string]bool) {
	endpoint := fmt.Sprintf("/repos/%s/%s/git/trees/%s", owner, name, branch)
	data, err := s.client.GetWithParams(ctx, endpoint, map[string]string{"recursive": "1"})
	if err != nil {
		return
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
		} `json:"tree"`
	}
	if err := json.Unmarshal(data, &tree); err != nil {
		return
	}

	for _, item := range tree.Tree {
		if strings.HasPrefix(item.Path, ".github/workflows/") && strings.HasSuffix(item.Path, ".yml") {
			content, err := s.client.GetFileContent(ctx, owner, name, item.Path, branch)
			if err != nil {
				continue
			}
			parseWorkflowEnvironments(content, targets)
		}
	}
}

func (s *Source) collectTargetsFromDeployConfigs(ctx context.Context, owner, name, branch string, targets map[string]bool) {
	configFiles := []string{
		"appspec.json",
		"appspec.yml",
		"imagedefinitions.json",
		"buildspec.yml",
	}

	for _, file := range configFiles {
		content, err := s.client.GetFileContent(ctx, owner, name, file, branch)
		if err != nil {
			continue
		}
		// Extract environment names from deploy configs
		if strings.Contains(content, "environment") {
			targets[file] = true
		}
	}
}

// parseWorkflowEnvironments extracts environment names from GitHub Actions workflow YAML.
func parseWorkflowEnvironments(content string, targets map[string]bool) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for environment: lines
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
		// Look for deploy section
		if strings.HasPrefix(line, "deploy:") {
			targets["deploy"] = true
		}
	}
}
