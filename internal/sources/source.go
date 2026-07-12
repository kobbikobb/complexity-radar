package sources

import (
	"context"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// SourceMetric is a metric value collected from a source, identified by metric type name.
type SourceMetric struct {
	Type  model.MetricTypeName
	Value float64
}

// Source defines the interface for collecting metrics from external systems.
//
// Each implementation knows how to collect a specific set of metrics
// from a source like GitHub, Jira, or Grafana.
type Source interface {
	// Name returns the source identifier (e.g., "github").
	Name() string

	// Collect gathers all supported metrics for the given repository.
	Collect(ctx context.Context, repo model.Repository) ([]SourceMetric, error)

	// SupportedMetrics returns the metric types this source can collect.
	SupportedMetrics() []model.MetricTypeName
}
