package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

func prJSON(prs ...string) json.RawMessage {
	return json.RawMessage("[" + strings.Join(prs, ",") + "]")
}

func daysAgo(d float64) string {
	return time.Now().Add(-time.Duration(d*24) * time.Hour).UTC().Format(time.RFC3339)
}

func botPR(updatedAt string) string {
	return fmt.Sprintf(`{"state":"open","updated_at":%q,"user":{"type":"Bot"},"draft":false}`, updatedAt)
}

func humanPR(updatedAt string, draft bool) string {
	return fmt.Sprintf(`{"state":"open","updated_at":%q,"user":{"type":"User"},"draft":%t}`, updatedAt, draft)
}

func staleCount(t *testing.T, prs json.RawMessage) float64 {
	t.Helper()

	client := &mockClient{responses: map[string]json.RawMessage{"/repos/org/repo/pulls": prs}}
	source := NewSourceWithClient(client)

	metrics, err := source.collectStalePRs(context.Background(), "org", "repo")
	if err != nil {
		t.Fatalf("collectStalePRs: %v", err)
	}
	metric, ok := findMetric(metrics, model.MetricTypeStalePRs)
	if !ok {
		t.Fatalf("stale_prs metric not found")
	}
	return metric.Value
}

func TestCollectStalePRsBucketing(t *testing.T) {
	t.Run("should count a bot PR idle over 7 days as stale", func(t *testing.T) {
		// Arrange
		prs := prJSON(botPR(daysAgo(30)))

		// Act
		got := staleCount(t, prs)

		// Assert
		if got != 1 {
			t.Errorf("expected 1 stale PR, got %v", got)
		}
	})

	t.Run("should exclude draft PRs", func(t *testing.T) {
		// Arrange
		prs := prJSON(humanPR(daysAgo(30), true))

		// Act
		got := staleCount(t, prs)

		// Assert
		if got != 0 {
			t.Errorf("expected 0 stale PRs for a draft, got %v", got)
		}
	})

	t.Run("should count a PR at the 7-day boundary as stale", func(t *testing.T) {
		// Arrange
		prs := prJSON(humanPR(daysAgo(7.01), false))

		// Act
		got := staleCount(t, prs)

		// Assert
		if got != 1 {
			t.Errorf("expected 1 stale PR at the 7-day boundary, got %v", got)
		}
	})

	t.Run("should not count a PR idle under 7 days", func(t *testing.T) {
		// Arrange
		prs := prJSON(humanPR(daysAgo(6), false))

		// Act
		got := staleCount(t, prs)

		// Assert
		if got != 0 {
			t.Errorf("expected 0 stale PRs under 7 days, got %v", got)
		}
	})
}
