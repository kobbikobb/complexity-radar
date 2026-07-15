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

func collectDebt(t *testing.T, repo model.Repository, features []Feature) ([]model.SourceMetric, *stubLister) {
	t.Helper()

	stub := &stubLister{features: features}
	src := NewSourceWithClient(func(string, string) FeatureLister { return stub })

	metrics, err := src.Collect(context.Background(), repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return metrics, stub
}

func TestCollect(t *testing.T) {
	t.Setenv("DEVCYCLE_CLIENT_ID", "id")
	t.Setenv("DEVCYCLE_CLIENT_SECRET", "secret")

	t.Run("should count active flags flagged stale by devcycle", func(t *testing.T) {
		// Arrange
		repo := model.Repository{DevCycleProjectKey: "proj"}
		features := []Feature{
			{Status: "active", Staleness: &Staleness{Reason: "unused"}},
			{Status: "active", UpdatedAt: daysAgo(1)},
		}

		// Act
		metrics, _ := collectDebt(t, repo, features)

		// Assert
		if got := metrics[0].Value; got != 1 {
			t.Fatalf("debt = %v, want 1", got)
		}
	})

	t.Run("should count active flags untouched past the fallback threshold", func(t *testing.T) {
		// Arrange
		repo := model.Repository{DevCycleProjectKey: "proj"}
		features := []Feature{
			{Status: "active", UpdatedAt: daysAgo(45)},
			{Status: "active", UpdatedAt: daysAgo(10)},
		}

		// Act
		metrics, _ := collectDebt(t, repo, features)

		// Assert
		if got := metrics[0].Value; got != 1 {
			t.Fatalf("debt = %v, want 1", got)
		}
	})

	t.Run("should ignore non-active flags", func(t *testing.T) {
		// Arrange
		repo := model.Repository{DevCycleProjectKey: "proj"}
		features := []Feature{
			{Status: "complete", Staleness: &Staleness{Reason: "unused"}},
			{Status: "archived", UpdatedAt: daysAgo(90)},
		}

		// Act
		metrics, _ := collectDebt(t, repo, features)

		// Assert
		if got := metrics[0].Value; got != 0 {
			t.Fatalf("debt = %v, want 0", got)
		}
	})
}

func TestCollectSkipsWhenUnconfigured(t *testing.T) {
	t.Run("should skip when repo has no project key", func(t *testing.T) {
		// Arrange
		t.Setenv("DEVCYCLE_CLIENT_ID", "id")
		t.Setenv("DEVCYCLE_CLIENT_SECRET", "secret")
		repo := model.Repository{}

		// Act
		metrics, stub := collectDebt(t, repo, nil)

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
		t.Setenv("DEVCYCLE_CLIENT_ID", "")
		t.Setenv("DEVCYCLE_CLIENT_SECRET", "")
		repo := model.Repository{DevCycleProjectKey: "proj"}

		// Act
		metrics, stub := collectDebt(t, repo, nil)

		// Assert
		if metrics != nil {
			t.Fatalf("metrics = %v, want nil", metrics)
		}
		if stub.called {
			t.Fatal("client called despite missing credentials")
		}
	})
}
