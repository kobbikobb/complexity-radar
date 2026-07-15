// Package sources composes the metric sources radar collects from.
package sources

import (
	"context"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources/devcycle"
	"github.com/kobbikobb/complexity-radar/internal/sources/github"
)

// Default returns the composite of every per-repo source radar knows about.
// This is the single place new repo-scoped vendor sources get registered.
func Default() model.Source {
	return Multi{github.NewSource()}
}

// DefaultProject returns every project-scoped source. Opt-in sources self-skip
// when unconfigured, so listing one here has no effect until a user configures
// it; this is the single place new project-scoped vendors get registered.
func DefaultProject() []model.ProjectSource {
	return []model.ProjectSource{devcycle.NewSource()}
}

// Multi merges several sources into one, concatenating their metrics and
// unioning their supported-metric lists. A child that returns an error fails
// the collection, matching single-source behavior.
type Multi []model.Source

func (m Multi) Name() string {
	names := make([]string, len(m))
	for i, s := range m {
		names[i] = s.Name()
	}
	return strings.Join(names, "+")
}

func (m Multi) SupportedMetrics() []model.MetricTypeName {
	var all []model.MetricTypeName
	for _, s := range m {
		all = append(all, s.SupportedMetrics()...)
	}
	return all
}

func (m Multi) Collect(ctx context.Context, repo model.Repository) ([]model.SourceMetric, error) {
	var all []model.SourceMetric
	for _, s := range m {
		metrics, err := s.Collect(ctx, repo)
		if err != nil {
			return nil, err
		}
		all = append(all, metrics...)
	}
	return all, nil
}
