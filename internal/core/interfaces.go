package core

import (
	"context"
	"io"
)

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
// A formatter writes the formatted representation of a Report to the provided writer.
type OutputFormatter interface {
	// Format writes the formatted report to w and returns any error encountered.
	Format(report Report, w io.Writer) error
}
