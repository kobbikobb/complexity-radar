package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

type fakeSource struct {
	name    string
	metrics []model.SourceMetric
	err     error
}

func (f fakeSource) Name() string                             { return f.name }
func (f fakeSource) SupportedMetrics() []model.MetricTypeName { return nil }
func (f fakeSource) Collect(context.Context, model.Repository) ([]model.SourceMetric, error) {
	return f.metrics, f.err
}

func TestMultiCollect(t *testing.T) {
	t.Run("should concatenate metrics from all children", func(t *testing.T) {
		// Arrange
		m := Multi{
			fakeSource{metrics: []model.SourceMetric{{Type: "a", Value: 1}}},
			fakeSource{metrics: []model.SourceMetric{{Type: "b", Value: 2}}},
		}

		// Act
		got, err := m.Collect(context.Background(), model.Repository{})

		// Assert
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
	})

	t.Run("should fail when a child errors", func(t *testing.T) {
		// Arrange
		m := Multi{fakeSource{err: errors.New("boom")}}

		// Act
		_, err := m.Collect(context.Background(), model.Repository{})

		// Assert
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
