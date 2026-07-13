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
