package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// --- helpers ---

func sampleReport() Report {
	return Report{
		ProjectName:        "my-project",
		ProjectDescription: "A test project",
		OverallScore:       75.0,
		Dimensions: []DimensionReport{
			{Dimension: model.DimensionCode, Score: 82.0, Weight: 40.0, MetricCount: 3},
			{Dimension: model.DimensionDelivery, Score: 65.0, Weight: 30.0, MetricCount: 3},
		},
		Metrics: []MetricReport{
			{Name: model.MetricTypeDeployFrequency, Dimension: model.DimensionDelivery, RawValue: 3.5, Normalized: 70.0, Unit: "per_week"},
			{Name: model.MetricTypeBuildTime, Dimension: model.DimensionDelivery, RawValue: 120.0, Normalized: 55.0, Unit: "seconds"},
			{Name: model.MetricTypeDependencyCount, Dimension: model.DimensionCode, RawValue: 42.0, Normalized: 85.0, Unit: "count"},
		},
		CollectedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Errors:      nil,
	}
}

func TestFormatEmptyReport(t *testing.T) {
	f := New()
	f.UseColor = false
	report := Report{
		ProjectName: "empty",
		CollectedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Dimensions:  nil,
		Metrics:     nil,
		Errors:      nil,
	}
	out := f.Format(report)

	if !strings.Contains(out, "ComplexityRadar Report") {
		t.Error("missing header")
	}
	if !strings.Contains(out, "Project: empty") {
		t.Error("missing project name")
	}
	if !strings.Contains(out, "OVERALL SCORE: 0.0") {
		t.Error("missing overall score")
	}
	if !strings.Contains(out, "Collected:") {
		t.Error("missing footer")
	}
}

func TestFormatWithProjectInfo(t *testing.T) {
	f := New()
	f.UseColor = false
	report := sampleReport()
	out := f.Format(report)

	if !strings.Contains(out, "Project: my-project") {
		t.Error("missing project name")
	}
	if !strings.Contains(out, "A test project") {
		t.Error("missing project description")
	}
}

func TestFormatWithOverallScore(t *testing.T) {
	f := New()
	f.UseColor = false
	report := sampleReport()
	report.OverallScore = 85.5
	out := f.Format(report)

	if !strings.Contains(out, "OVERALL SCORE: 85.5 [B]") {
		t.Errorf("expected 85.5 [B] in output, got:\n%s", out)
	}
}

func TestOverallGradeFloorsAtWorstCritical(t *testing.T) {
	dims := []DimensionReport{
		{Dimension: model.DimensionSecurity, Score: 45, MetricCount: 1}, // D
		{Dimension: model.DimensionDelivery, Score: 95, MetricCount: 1},
	}

	if got := overallGrade(70, dims); got != "C" {
		t.Errorf("overallGrade = %q, want C (capped at security D + 1)", got)
	}
}

func TestOverallGradeIgnoresCriticalWithoutData(t *testing.T) {
	dims := []DimensionReport{
		{Dimension: model.DimensionSecurity, Score: 0, MetricCount: 0}, // no data, not collected
		{Dimension: model.DimensionDelivery, Score: 95, MetricCount: 1},
	}

	if got := overallGrade(92, dims); got != "A" {
		t.Errorf("overallGrade = %q, want A (uncollected security must not cap)", got)
	}
}

func TestFormatShowsTrendWhenEnabled(t *testing.T) {
	f := New()
	f.UseColor = false
	f.ShowTrend = true
	report := sampleReport()
	report.HasTrend = true
	report.OverallDelta = 4.2
	out := f.Format(report)

	if !strings.Contains(out, "▲ +4.2 vs previous") {
		t.Errorf("expected overall trend delta in output, got:\n%s", out)
	}
}

func TestFormatHidesTrendWhenDisabled(t *testing.T) {
	f := New()
	f.UseColor = false
	f.ShowTrend = false
	report := sampleReport()
	report.HasTrend = true
	report.OverallDelta = 4.2
	out := f.Format(report)

	if strings.Contains(out, "vs previous") {
		t.Errorf("trend shown without --history, got:\n%s", out)
	}
}

func TestFormatShouldIncludeScoreLegend(t *testing.T) {
	f := New()
	f.UseColor = false

	out := f.Format(sampleReport())

	if !strings.Contains(out, "higher is healthier") {
		t.Error("missing score direction legend")
	}
	if !strings.Contains(out, "A ≥90") {
		t.Errorf("missing grade band legend, output:\n%s", out)
	}
}

func TestFormatWithDimensions(t *testing.T) {
	f := New()
	f.UseColor = false
	out := f.Format(sampleReport())

	if !strings.Contains(out, "Dimension Scores:") {
		t.Error("missing dimension scores section")
	}
	if !strings.Contains(out, "code") {
		t.Error("missing code dimension")
	}
	if !strings.Contains(out, "delivery") {
		t.Error("missing delivery dimension")
	}
	if !strings.Contains(out, "40.0%") {
		t.Errorf("expected weight 40.0%%, output:\n%s", out)
	}
}

func TestFormatWithMetrics(t *testing.T) {
	f := New()
	f.UseColor = false
	out := f.Format(sampleReport())

	if !strings.Contains(out, "Metric Details:") {
		t.Error("missing metric details section")
	}
	// deploy frequency: snake_case → title case
	if !strings.Contains(out, "Deploy Frequency") {
		t.Error("missing deploy frequency metric name")
	}
	// per_week unit → /week
	if !strings.Contains(out, "/week") {
		t.Errorf("expected /week unit, output:\n%s", out)
	}
}

func TestFormatWithErrors(t *testing.T) {
	f := New()
	f.UseColor = false
	report := sampleReport()
	report.Errors = []string{"auth failed", "timeout"}
	out := f.Format(report)

	if !strings.Contains(out, "Errors:") {
		t.Error("missing errors section")
	}
	if !strings.Contains(out, "auth failed") {
		t.Error("missing error message 1")
	}
	if !strings.Contains(out, "timeout") {
		t.Error("missing error message 2")
	}
}

func TestFormatWithoutErrors(t *testing.T) {
	f := New()
	f.UseColor = false
	out := f.Format(sampleReport())

	if strings.Contains(out, "Errors:") {
		t.Error("errors section should not appear")
	}
}

// --- color tests ---

func TestColorScoreGreen(t *testing.T) {
	f := New()
	out := f.colorScore(85.0)
	if !strings.Contains(out, "\033[32m") {
		t.Errorf("expected green ANSI, got %q", out)
	}
	if !strings.Contains(out, "85.0") {
		t.Errorf("expected 85.0, got %q", out)
	}
}

func TestColorScoreYellow(t *testing.T) {
	f := New()
	out := f.colorScore(65.0)
	if !strings.Contains(out, "\033[33m") {
		t.Errorf("expected yellow ANSI, got %q", out)
	}
	if !strings.Contains(out, "65.0") {
		t.Errorf("expected 65.0, got %q", out)
	}
}

func TestColorScoreRed(t *testing.T) {
	f := New()
	out := f.colorScore(40.0)
	if !strings.Contains(out, "\033[31m") {
		t.Errorf("expected red ANSI, got %q", out)
	}
	if !strings.Contains(out, "40.0") {
		t.Errorf("expected 40.0, got %q", out)
	}
}

func TestNoColor(t *testing.T) {
	f := New()
	f.UseColor = false
	out := f.colorScore(85.0)
	if strings.Contains(out, "\033") {
		t.Errorf("expected no ANSI codes, got %q", out)
	}
}

// --- helper function tests ---

func TestFormatMetricName(t *testing.T) {
	tests := []struct {
		input model.MetricTypeName
		want  string
	}{
		{model.MetricTypeDeployFrequency, "Deploy Frequency"},
		{model.MetricTypeSecurityVulnerabilities, "Security Vulnerabilities"},
		{model.MetricTypeBuildSuccessRatio, "Build Success Ratio"},
		{model.MetricTypeDependencyCount, "Dependency Count"},
		{model.MetricTypeCICDComplexity, "CI/CD Complexity"},
		{model.MetricTypeStalePRs, "Stale PRs"},
		{model.MetricTypeK8sDeployments, "K8s Deployments"},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := formatMetricName(tt.input)
			if got != tt.want {
				t.Errorf("formatMetricName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatUnit(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"per_week", "/week"},
		{"seconds", "sec"},
		{"ratio", "%"},
		{"count", "count"},
		{"score", "score"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatUnit(tt.input)
			if got != tt.want {
				t.Errorf("formatUnit(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatRawValue(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		unit  string
		want  string
	}{
		{"ratio 0.756", 0.756, "ratio", "75.60"},
		{"ratio 1.0", 1.0, "ratio", "100.00"},
		{"seconds integer", 120.0, "seconds", "120.0"},
		{"seconds decimal", 120.5, "seconds", "120.5"},
		{"count integer", 42.0, "count", "42.0"},
		{"count decimal", 42.5, "count", "42.50"},
		{"score integer", 10.0, "score", "10.0"},
		{"zero", 0.0, "count", "0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRawValue(tt.value, tt.unit)
			if got != tt.want {
				t.Errorf("formatRawValue(%v, %q) = %q, want %q", tt.value, tt.unit, got, tt.want)
			}
		})
	}
}

func TestScoreGrade(t *testing.T) {
	tests := []struct {
		score float64
		grade string
	}{
		{95.0, "A"},
		{90.0, "A"},
		{85.0, "B"},
		{80.0, "B"},
		{75.0, "B"},
		{70.0, "C"},
		{65.0, "C"},
		{60.0, "C"},
		{55.0, "D"},
		{45.0, "D"},
		{40.0, "D"},
		{39.0, "F"},
		{0.0, "F"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.score), func(t *testing.T) {
			got := scoreGrade(tt.score)
			if got != tt.grade {
				t.Errorf("scoreGrade(%v) = %q, want %q", tt.score, got, tt.grade)
			}
		})
	}
}

// --- integration tests ---

func TestFormatterImplementsInterface(t *testing.T) {
	var _ OutputFormatter = &TerminalFormatter{}
}

func TestFormatSampleReport(t *testing.T) {
	f := New()
	f.UseColor = false
	out := f.Format(sampleReport())

	// header
	if !strings.Contains(out, "═══════════════════════════════════════════════════") {
		t.Error("missing box border")
	}
	if !strings.Contains(out, "ComplexityRadar Report") {
		t.Error("missing title")
	}

	// project
	if !strings.Contains(out, "Project: my-project") {
		t.Error("missing project")
	}
	if !strings.Contains(out, "A test project") {
		t.Error("missing description")
	}

	// overall score with grade B (75.0)
	if !strings.Contains(out, "OVERALL SCORE: 75.0 [B]") {
		t.Errorf("unexpected overall score line, output:\n%s", out)
	}

	// dimensions
	if !strings.Contains(out, "Dimension Scores:") {
		t.Error("missing dimensions header")
	}
	if !strings.Contains(out, "code") {
		t.Error("missing code dimension")
	}
	if !strings.Contains(out, "delivery") {
		t.Error("missing delivery dimension")
	}

	// metrics
	if !strings.Contains(out, "Metric Details:") {
		t.Error("missing metrics header")
	}
	if !strings.Contains(out, "Deploy Frequency") {
		t.Error("missing deploy frequency")
	}
	if !strings.Contains(out, "Build Time") {
		t.Error("missing build time")
	}
	if !strings.Contains(out, "Dependency Count") {
		t.Error("missing dependency count")
	}

	// footer
	if !strings.Contains(out, "Collected: 2025-01-15T10:30:00Z") {
		t.Errorf("unexpected collected time, output:\n%s", out)
	}

	// no errors section
	if strings.Contains(out, "Errors:") {
		t.Error("should not have errors section")
	}
}

// --- stripANSI tests ---

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no escape", "hello", "hello"},
		{"empty", "", ""},
		{"green", "\033[32m85.0\033[0m", "85.0"},
		{"yellow", "\033[33m65.0\033[0m", "65.0"},
		{"red", "\033[31m40.0\033[0m", "40.0"},
		{"nested", "\033[32m\033[1m85.0\033[0m\033[0m", "85.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- colorScore boundary tests ---

func TestColorScoreBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		wantColor string
	}{
		{"90 is green", 90.0, "\033[32m"},
		{"75 is green", 75.0, "\033[32m"},
		{"74.9 is yellow", 74.9, "\033[33m"},
		{"60 is yellow", 60.0, "\033[33m"},
		{"59.9 is red", 59.9, "\033[31m"},
		{"0 is red", 0.0, "\033[31m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New()
			got := f.colorScore(tt.score)
			if !strings.Contains(got, tt.wantColor) {
				t.Errorf("colorScore(%v) = %q, want color %s", tt.score, got, tt.wantColor)
			}
		})
	}
}
