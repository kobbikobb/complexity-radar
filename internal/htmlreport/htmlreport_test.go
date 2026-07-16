package htmlreport

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/terminal"
)

func sampleReport(name string, aggregate bool) terminal.Report {
	return terminal.Report{
		ProjectName:        name,
		ProjectDescription: "A test project",
		OverallScore:       78.0,
		Aggregate:          aggregate,
		Dimensions: []terminal.DimensionReport{
			{Dimension: model.DimensionSecurity, Score: 82.0, Weight: 30.0, MetricCount: 1},
			{Dimension: model.DimensionDelivery, Score: 55.0, Weight: 30.0, MetricCount: 1},
			{Dimension: model.DimensionCode, Score: 0.0, Weight: 40.0, MetricCount: 0},
		},
		Metrics: []terminal.MetricReport{
			{Name: model.MetricTypeSecurityVulnerabilities, Dimension: model.DimensionSecurity, RawValue: 3.0, Normalized: 82.0, Unit: "weighted"},
			{Name: model.MetricTypeSecurityHigh, Dimension: model.DimensionSecurity, RawValue: 2.0, Normalized: 0.0, Unit: "count"},
			{Name: model.MetricTypeDeployFrequency, Dimension: model.DimensionDelivery, RawValue: 2.5, Normalized: 55.0, Unit: "per_week"},
		},
		CollectedAt: time.Date(2026, 7, 15, 10, 30, 0, 0, time.UTC),
	}
}

func TestRender(t *testing.T) {
	t.Run("should include project name, overall score and all dimensions", func(t *testing.T) {
		// Arrange
		project := sampleReport("my-project", true)

		// Act
		out, err := Render(project, nil)

		// Assert
		if err != nil {
			t.Fatalf("Render returned error: %v", err)
		}
		if !strings.Contains(out, "my-project") {
			t.Error("missing project name")
		}
		if !strings.Contains(out, "78") {
			t.Error("missing overall score")
		}
		for _, dim := range []string{"Security", "Delivery", "Code"} {
			if !strings.Contains(out, dim) {
				t.Errorf("missing dimension %q", dim)
			}
		}
	})

	t.Run("should render metric rows with raw values", func(t *testing.T) {
		// Arrange
		project := sampleReport("my-project", true)

		// Act
		out, _ := Render(project, nil)

		// Assert
		if !strings.Contains(out, terminal.MetricDisplayName(model.MetricTypeDeployFrequency)) {
			t.Error("missing deploy frequency metric row")
		}
		if !strings.Contains(out, "/week") {
			t.Error("missing formatted unit")
		}
	})

	t.Run("should place display-only metrics in a details section", func(t *testing.T) {
		// Arrange
		project := sampleReport("my-project", true)

		// Act
		out, _ := Render(project, nil)

		// Assert
		if !strings.Contains(out, "<details") {
			t.Error("expected a <details> block for display-only metrics")
		}
		if !strings.Contains(out, "Raw context (not scored)") {
			t.Error("expected 'Raw context (not scored)' summary for display-only metrics")
		}
		if !strings.Contains(out, "—") {
			t.Error("expected an em dash for display-only metric score")
		}
	})

	t.Run("should not render per-repository sections", func(t *testing.T) {
		// Arrange
		project := sampleReport("my-project", true)
		repos := []terminal.Report{sampleReport("repo-a", false), sampleReport("repo-b", false)}

		// Act
		out, _ := Render(project, repos)

		// Assert
		if strings.Contains(out, "repo-a") || strings.Contains(out, "repo-b") {
			t.Error("per-repository sections should not be rendered")
		}
		if strings.Contains(out, "Per-repository detail") {
			t.Error("per-repository heading should not be rendered")
		}
	})

	t.Run("should escape dynamic text", func(t *testing.T) {
		// Arrange
		project := sampleReport("<script>alert(1)</script>", true)

		// Act
		out, _ := Render(project, nil)

		// Assert
		if strings.Contains(out, "<script>alert(1)</script>") {
			t.Error("dynamic text was not escaped")
		}
	})

	t.Run("should not reference any external resources", func(t *testing.T) {
		// Arrange
		project := sampleReport("my-project", true)

		// Act
		out, _ := Render(project, nil)

		// Assert
		for _, pat := range []string{`src=`, `href=`, `@import`, `url(`} {
			if strings.Contains(out, pat) {
				t.Errorf("output references external resource via %q", pat)
			}
		}
		if regexp.MustCompile(`https?://`).MatchString(out) {
			t.Error("output contains an http(s) URL")
		}
	})
}
