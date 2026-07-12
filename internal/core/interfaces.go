package core

import "context"

// Source defines the interface that all metric sources must implement.
// A source is responsible for collecting specific metrics from a repository.
type Source interface {
	// Name returns the human-readable name of this source.
	Name() string

	// Collect retrieves metrics from the given repository.
	// It uses the provided context for cancellation and timeout control.
	Collect(ctx context.Context, repo Repository) ([]Metric, error)

	// SupportedMetrics returns the list of metric types this source can collect.
	SupportedMetrics() []MetricType
}

// OutputFormatter defines the interface that all report formatters must implement.
// A formatter converts a Report into a string representation for display or export.
type OutputFormatter interface {
	// Format converts a report into a formatted string.
	Format(report Report) string
}
