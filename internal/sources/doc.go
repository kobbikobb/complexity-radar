// Package sources defines the Source interface and built-in implementations.
//
// Each source implements the Source interface to collect metrics from
// an external system (GitHub, Jira, Grafana, etc.).
//
// Interface:
//
//	type Source interface {
//	    Name() string
//	    Collect(ctx context.Context, repo Repository) ([]Metric, error)
//	    SupportedMetrics() []MetricType
//	}
package sources
