package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/scorer"
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
	if !strings.Contains(out, "OVERALL SCORE: 0") {
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
	report.OverallScore = 85.0
	out := f.Format(report)

	if !strings.Contains(out, "OVERALL SCORE: 85 [A]") {
		t.Errorf("expected 85 [A] in output, got:\n%s", out)
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

// TestScoringSanityGatePlatformFixture pins the real platform raw values,
// normalized per-service (~100 services), with no open criticals. It guards two
// things: the asymptotic curve never floors a healthy repo to F, and per-service
// normalization stops the monorepo's size from dragging security/infra down.
func TestScoringSanityGatePlatformFixture(t *testing.T) {
	// Arrange: size-scaling metrics are per-service (absolute ÷ ~100 services).
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0.959,
		model.MetricTypeStalePRs:                46,
		model.MetricTypeBuildTime:               365.8,
		model.MetricTypeDeployFrequency:         5,
		model.MetricTypeBuildSuccessRatio:       1.0,
		model.MetricTypeDecisionDensity:         18.4,
		model.MetricTypeDependencyCount:         5.8,
		model.MetricTypeK8sDeployments:          0.79,
		model.MetricTypeContainerImages:         0.05,
		model.MetricTypeDeployTargets:           11,
		model.MetricTypeCICDComplexity:          100,
	}

	// Act
	result := scorer.ScoreWithDefaults(metrics)
	dims := make([]DimensionReport, len(result.Dimensions))
	byDim := map[model.Dimension]float64{}
	for i, d := range result.Dimensions {
		dims[i] = DimensionReport{Dimension: d.Dimension, Score: d.Score, MetricCount: d.MetricCount}
		byDim[d.Dimension] = d.Score
	}
	grade := overallGrade(result.Overall, dims)

	// Assert
	if result.Overall < 75 || result.Overall > 87 {
		t.Errorf("overall = %.1f, want [75,87]", result.Overall)
	}
	if grade != "B" {
		t.Errorf("overall grade = %q, want B", grade)
	}
	bands := []struct {
		dim    model.Dimension
		lo, hi float64
	}{
		{model.DimensionSecurity, 90, 100},
		{model.DimensionDelivery, 78, 90},
		{model.DimensionInfrastructure, 85, 97},
		{model.DimensionCode, 55, 65},
	}
	for _, b := range bands {
		if s := byDim[b.dim]; s < b.lo || s > b.hi {
			t.Errorf("%s dimension = %.1f, want [%v,%v]", b.dim, s, b.lo, b.hi)
		}
	}
}

// TestSecurityCriticalGatesPlatformGrade takes the same platform fixture plus
// the open criticals the real run carried, and asserts the critical gate drops
// security to F and the overall grade below C.
func TestSecurityCriticalGatesPlatformGrade(t *testing.T) {
	// Arrange: same per-service fixture, plus the 4 open criticals the real run
	// carried. Without the gate, per-service normalization would score security
	// as healthy (~84); the gate must override that.
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0.959,
		model.MetricTypeSecurityCritical:        4,
		model.MetricTypeStalePRs:                46,
		model.MetricTypeBuildTime:               365.8,
		model.MetricTypeDeployFrequency:         5,
		model.MetricTypeBuildSuccessRatio:       1.0,
		model.MetricTypeDecisionDensity:         18.4,
		model.MetricTypeDependencyCount:         5.8,
		model.MetricTypeK8sDeployments:          0.79,
		model.MetricTypeContainerImages:         0.05,
		model.MetricTypeDeployTargets:           11,
		model.MetricTypeCICDComplexity:          100,
	}

	// Act
	result := scorer.ScoreWithDefaults(metrics)
	dims := make([]DimensionReport, len(result.Dimensions))
	byDim := map[model.Dimension]float64{}
	for i, d := range result.Dimensions {
		dims[i] = DimensionReport{Dimension: d.Dimension, Score: d.Score, MetricCount: d.MetricCount}
		byDim[d.Dimension] = d.Score
	}
	grade := overallGrade(result.Overall, dims)

	// Assert
	if s := byDim[model.DimensionSecurity]; s >= 40 {
		t.Errorf("security = %.1f, want < 40 (F, gated by open critical)", s)
	}
	if grade == "C" || grade == "B" || grade == "A" {
		t.Errorf("overall grade = %q, want D or F (critical must cap the rollup)", grade)
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

func TestFormatShouldShowMethodologyWhenExplainEnabled(t *testing.T) {
	f := New()
	f.UseColor = false
	f.ShowExplain = true

	out := f.Format(sampleReport())

	bt := methodology[model.MetricTypeBuildTime]
	if !strings.Contains(out, "Methodology:") {
		t.Errorf("expected methodology section, got:\n%s", out)
	}
	if !strings.Contains(out, bt.RawDef) || !strings.Contains(out, bt.ScoreDef) || !strings.Contains(out, bt.Source) {
		t.Errorf("expected build_time raw/score/source in output, got:\n%s", out)
	}
}

func TestFormatShouldHideMethodologyWhenExplainDisabled(t *testing.T) {
	f := New()
	f.UseColor = false
	f.ShowExplain = false

	out := f.Format(sampleReport())

	if strings.Contains(out, "Methodology:") {
		t.Errorf("methodology section shown without --explain, got:\n%s", out)
	}
	if strings.Contains(out, methodology[model.MetricTypeBuildTime].RawDef) {
		t.Errorf("raw definition shown without --explain, got:\n%s", out)
	}
}

func TestFormatShouldKeepMetricRowsNarrowByDefault(t *testing.T) {
	f := New()
	f.UseColor = false

	out := f.Format(sampleReport())

	if strings.Contains(out, methodology[model.MetricTypeBuildTime].ScoreDef) {
		t.Errorf("scoring definition should stay out of narrow rows without --explain, got:\n%s", out)
	}
}

func TestFormatShouldGroupMetricsByDimension(t *testing.T) {
	// Arrange
	f := New()
	f.UseColor = false

	// Act
	out := f.Format(sampleReport())

	// Assert
	if !strings.Contains(out, "Delivery   65 C") {
		t.Errorf("expected delivery group header driving its metrics, got:\n%s", out)
	}
	if strings.Index(out, "Deploy Frequency") < strings.Index(out, "Delivery   65 C") {
		t.Errorf("expected delivery metrics under their group header, got:\n%s", out)
	}
}

func TestFormatShouldIncludeScoreLegend(t *testing.T) {
	f := New()
	f.UseColor = false

	out := f.Format(sampleReport())

	if !strings.Contains(out, "higher is healthier") {
		t.Error("missing score direction legend")
	}
	if !strings.Contains(out, "A ≥85") {
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
	if !strings.Contains(out, "85") {
		t.Errorf("expected 85, got %q", out)
	}
}

func TestColorScoreYellow(t *testing.T) {
	f := New()
	out := f.colorScore(65.0)
	if !strings.Contains(out, "\033[33m") {
		t.Errorf("expected yellow ANSI, got %q", out)
	}
	if !strings.Contains(out, "65") {
		t.Errorf("expected 65, got %q", out)
	}
}

func TestColorScoreRed(t *testing.T) {
	f := New()
	out := f.colorScore(40.0)
	if !strings.Contains(out, "\033[31m") {
		t.Errorf("expected red ANSI, got %q", out)
	}
	if !strings.Contains(out, "40") {
		t.Errorf("expected 40, got %q", out)
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
		{model.MetricTypeCICDComplexity, "CI/CD Maturity"},
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
		{85.0, "A"},
		{80.0, "B"},
		{70.0, "B"},
		{65.0, "C"},
		{50.0, "C"},
		{45.0, "D"},
		{30.0, "D"},
		{29.0, "F"},
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

	// overall score with grade B (75)
	if !strings.Contains(out, "OVERALL SCORE: 75 [B]") {
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
