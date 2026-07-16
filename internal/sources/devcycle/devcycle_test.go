package devcycle

import (
	"context"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

type stubLister struct {
	features []Feature
	err      error
	called   bool
}

func (s *stubLister) ListFeatures(context.Context, string) ([]Feature, error) {
	s.called = true
	return s.features, s.err
}

func daysAgo(d int) string {
	return time.Now().Add(-time.Duration(d) * 24 * time.Hour).UTC().Format(time.RFC3339)
}

func collectDebt(t *testing.T, project model.Project, features []Feature) ([]model.SourceMetric, *stubLister) {
	t.Helper()

	stub := &stubLister{features: features}
	src := NewSourceWithClient(func(string, string) FeatureLister { return stub })

	metrics, err := src.CollectProject(context.Background(), project)
	if err != nil {
		t.Fatalf("CollectProject: %v", err)
	}
	return metrics, stub
}

func configured(key string) model.Project {
	return model.Project{DevCycleProjectKey: key, DevCycleClientID: "id", DevCycleClientSecret: "secret"}
}

func TestCollectProject(t *testing.T) {
	t.Run("should count active flags flagged stale by devcycle", func(t *testing.T) {
		// Arrange
		project := configured("proj")
		features := []Feature{
			{Status: "active", Staleness: &Staleness{Reason: "unused"}},
			{Status: "active", UpdatedAt: daysAgo(1)},
		}

		// Act
		metrics, _ := collectDebt(t, project, features)

		// Assert
		if got := metrics[0].Value; got != 1 {
			t.Fatalf("debt = %v, want 1", got)
		}
	})

	t.Run("should count active flags untouched past the fallback threshold", func(t *testing.T) {
		// Arrange
		project := configured("proj")
		features := []Feature{
			{Status: "active", UpdatedAt: daysAgo(45)},
			{Status: "active", UpdatedAt: daysAgo(10)},
		}

		// Act
		metrics, _ := collectDebt(t, project, features)

		// Assert
		if got := metrics[0].Value; got != 1 {
			t.Fatalf("debt = %v, want 1", got)
		}
	})

	t.Run("should ignore non-active flags", func(t *testing.T) {
		// Arrange
		project := configured("proj")
		features := []Feature{
			{Status: "complete", Staleness: &Staleness{Reason: "unused"}},
			{Status: "archived", UpdatedAt: daysAgo(90)},
		}

		// Act
		metrics, _ := collectDebt(t, project, features)

		// Assert
		if got := metrics[0].Value; got != 0 {
			t.Fatalf("debt = %v, want 0", got)
		}
	})
}

func TestCollectProjectSkipsWhenUnconfigured(t *testing.T) {
	t.Run("should skip when project has no key", func(t *testing.T) {
		// Arrange
		project := model.Project{DevCycleClientID: "id", DevCycleClientSecret: "secret"}

		// Act
		metrics, stub := collectDebt(t, project, nil)

		// Assert
		if metrics != nil {
			t.Fatalf("metrics = %v, want nil", metrics)
		}
		if stub.called {
			t.Fatal("client called despite missing project key")
		}
	})

	t.Run("should skip when credentials are absent", func(t *testing.T) {
		// Arrange
		project := model.Project{DevCycleProjectKey: "proj"}

		// Act
		metrics, stub := collectDebt(t, project, nil)

		// Assert
		if metrics != nil {
			t.Fatalf("metrics = %v, want nil", metrics)
		}
		if stub.called {
			t.Fatal("client called despite missing credentials")
		}
	})
}
