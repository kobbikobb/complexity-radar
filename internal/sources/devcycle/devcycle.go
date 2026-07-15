package devcycle

import (
	"context"
	"os"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// staleAfter is the fallback age threshold when DevCycle provides no
// staleness signal: an active flag untouched this long counts as debt.
const staleAfter = 30 * 24 * time.Hour

// FeatureLister is the subset of the DevCycle client the source needs (for testing).
type FeatureLister interface {
	ListFeatures(ctx context.Context, projectKey string) ([]Feature, error)
}

// Source collects feature-flag debt from DevCycle. It is opt-in: unless the
// project has a DevCycle project key and the DEVCYCLE_CLIENT_ID /
// DEVCYCLE_CLIENT_SECRET env vars are set, CollectProject is a no-op.
type Source struct {
	newClient func(id, secret string) FeatureLister
}

func NewSource() *Source {
	return &Source{newClient: func(id, secret string) FeatureLister {
		return NewClient(id, secret, nil)
	}}
}

// NewSourceWithClient injects a client factory for testing.
func NewSourceWithClient(newClient func(id, secret string) FeatureLister) *Source {
	return &Source{newClient: newClient}
}

func (s *Source) Name() string {
	return "devcycle"
}

func (s *Source) SupportedMetrics() []model.MetricTypeName {
	return []model.MetricTypeName{model.MetricTypeFeatureFlagDebt}
}

func (s *Source) CollectProject(ctx context.Context, project model.Project) ([]model.SourceMetric, error) {
	id, secret := os.Getenv("DEVCYCLE_CLIENT_ID"), os.Getenv("DEVCYCLE_CLIENT_SECRET")
	if project.DevCycleProjectKey == "" || id == "" || secret == "" {
		return nil, nil
	}

	features, err := s.newClient(id, secret).ListFeatures(ctx, project.DevCycleProjectKey)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	debt := 0
	for _, f := range features {
		if f.Status != "active" {
			continue
		}
		if isStale(f, now) {
			debt++
		}
	}

	return []model.SourceMetric{{Type: model.MetricTypeFeatureFlagDebt, Value: float64(debt)}}, nil
}

func isStale(f Feature, now time.Time) bool {
	if f.Staleness != nil {
		return true
	}
	updated, err := time.Parse(time.RFC3339, f.UpdatedAt)
	if err != nil {
		return false
	}
	return now.Sub(updated) > staleAfter
}
