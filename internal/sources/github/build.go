package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// WorkflowRun represents a GitHub Actions workflow run.
type WorkflowRun struct {
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	RunStartedAt string `json:"run_started_at"`
	RunAttempt   int    `json:"run_attempt"`
}

func buildSuccessRatio(runs []WorkflowRun) sources.SourceMetric {
	if len(runs) == 0 {
		return sources.SourceMetric{Type: model.MetricTypeBuildSuccessRatio, Value: 0}
	}

	successes := 0
	for _, r := range runs {
		if r.Conclusion == "success" {
			successes++
		}
	}

	ratio := float64(successes) / float64(len(runs))
	return sources.SourceMetric{Type: model.MetricTypeBuildSuccessRatio, Value: ratio}
}

func buildTime(runs []WorkflowRun) sources.SourceMetric {
	if len(runs) == 0 {
		return sources.SourceMetric{Type: model.MetricTypeBuildTime, Value: 0}
	}

	var totalSeconds float64
	count := 0
	for _, r := range runs {
		if r.Conclusion != "success" && r.Conclusion != "failure" {
			continue
		}
		started, err1 := time.Parse(time.RFC3339, r.RunStartedAt)
		finished, err2 := time.Parse(time.RFC3339, r.UpdatedAt)
		if err1 != nil || err2 != nil {
			continue
		}
		totalSeconds += finished.Sub(started).Seconds()
		count++
	}

	avgSeconds := 0.0
	if count > 0 {
		avgSeconds = totalSeconds / float64(count)
	}

	return sources.SourceMetric{Type: model.MetricTypeBuildTime, Value: avgSeconds}
}

func (s *Source) fetchWorkflowRuns(ctx context.Context, owner, name, branch string) ([]WorkflowRun, error) {
	params := map[string]string{
		"branch":   branch,
		"per_page": "30",
	}
	endpoint := fmt.Sprintf("/repos/%s/%s/actions/runs", owner, name)
	data, err := s.client.GetWithParams(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing workflow runs: %w", err)
	}

	return resp.WorkflowRuns, nil
}
