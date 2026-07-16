package terminal

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

type Report struct {
	ProjectName        string
	ProjectDescription string
	OverallScore       float64
	Dimensions         []DimensionReport
	Metrics            []MetricReport
	CollectedAt        time.Time
	Errors             []string
	HasTrend           bool
	OverallDelta       float64
	Aggregate          bool
}

type DimensionReport struct {
	Dimension   model.Dimension
	Score       float64
	Weight      float64
	WeightSum   float64
	MetricCount int
	Breakdown   string
	Delta       float64
}

type MetricReport struct {
	Name       model.MetricTypeName
	Dimension  model.Dimension
	RawValue   float64
	Normalized float64
	Unit       string
	Weight     float64
}

type OutputFormatter interface {
	Format(report Report) string
}

func scoreGrade(score float64) string {
	switch {
	case score >= 85:
		return "A"
	case score >= 70:
		return "B"
	case score >= 50:
		return "C"
	case score >= 30:
		return "D"
	default:
		return "F"
	}
}

// criticalDimensions cap the overall grade so strong delivery can't mask a weak
// critical dimension: the overall grade can't beat the worst critical grade by
// more than one letter.
var criticalDimensions = map[model.Dimension]bool{model.DimensionSecurity: true}

var gradeOrder = []string{"F", "D", "C", "B", "A"}

func gradeIndex(g string) int {
	for i, v := range gradeOrder {
		if v == g {
			return i
		}
	}
	return 0
}

func overallGrade(overall float64, dims []DimensionReport) string {
	cap := len(gradeOrder) - 1
	for _, d := range dims {
		if !criticalDimensions[d.Dimension] || d.MetricCount == 0 {
			continue
		}
		if c := gradeIndex(scoreGrade(d.Score)) + 1; c < cap {
			cap = c
		}
	}
	if gradeIndex(scoreGrade(overall)) > cap {
		return gradeOrder[cap]
	}
	return scoreGrade(overall)
}

func isDisplayOnly(mt model.MetricType) bool {
	return strings.Contains(mt.ScoreDef, "display-only")
}

// scoredDimensionExtremes returns the lowest and highest score among dimensions
// that actually have data. Returns (+Inf, -Inf) when none do.
func scoredDimensionExtremes(dims []DimensionReport) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, d := range dims {
		if d.MetricCount == 0 {
			continue
		}
		lo = math.Min(lo, d.Score)
		hi = math.Max(hi, d.Score)
	}
	return lo, hi
}

func SecurityBreakdown(metrics map[model.MetricTypeName]float64) string {
	crit := int(metrics[model.MetricTypeSecurityCritical])
	high := int(metrics[model.MetricTypeSecurityHigh])
	med := int(metrics[model.MetricTypeSecurityMedium])
	low := int(metrics[model.MetricTypeSecurityLow])
	return fmt.Sprintf("%d critical, %d high, %d medium, %d low", crit, high, med, low)
}

type TerminalFormatter struct {
	UseColor    bool
	ShowTrend   bool
	ShowExplain bool
}

var methodology = func() map[model.MetricTypeName]model.MetricType {
	m := map[model.MetricTypeName]model.MetricType{}
	for _, mt := range append(model.MetricTypes(), model.DisplayMetricTypes()...) {
		m[mt.Name] = mt
	}
	return m
}()

func formatDelta(v float64) string {
	switch {
	case v > 0:
		return fmt.Sprintf("▲ +%.1f", v)
	case v < 0:
		return fmt.Sprintf("▼ %.1f", v)
	default:
		return "▬ 0.0"
	}
}

func New() *TerminalFormatter {
	return &TerminalFormatter{UseColor: true}
}

const boxWidth = 80

// Exported wrappers let alternate renderers (e.g. htmlreport) reuse the exact
// grade bands and value formatting without duplicating the logic.
func Grade(score float64) string { return scoreGrade(score) }

func OverallGrade(overall float64, dims []DimensionReport) string {
	return overallGrade(overall, dims)
}

func MetricDisplayName(name model.MetricTypeName) string { return formatMetricName(name) }

func UnitDisplay(unit string) string { return formatUnit(unit) }

func RawValueDisplay(value float64, unit string) string { return formatRawValue(value, unit) }

func (f *TerminalFormatter) Format(report Report) string {
	var b strings.Builder
	head := strings.Repeat("═", boxWidth)
	rule := strings.Repeat("─", boxWidth)

	fmt.Fprintf(&b, "  %s\n  ComplexityRadar Report\n  %s\n\n", head, head)
	fmt.Fprintf(&b, "  Project: %s\n", report.ProjectName)
	if report.Aggregate {
		b.WriteString("  (project rollup — all repositories)\n")
	}
	if report.ProjectDescription != "" {
		fmt.Fprintf(&b, "  %s\n", report.ProjectDescription)
	}
	b.WriteString("\n")

	trend := f.ShowTrend && report.HasTrend

	fmt.Fprintf(&b, "  %s\n", rule)
	fmt.Fprintf(&b, "  OVERALL SCORE: %s [%s]", f.colorScore(report.OverallScore), overallGrade(report.OverallScore, report.Dimensions))
	if trend {
		fmt.Fprintf(&b, "   %s vs previous", formatDelta(report.OverallDelta))
	}
	fmt.Fprintf(&b, "\n  %s\n", rule)
	b.WriteString("  Scores 0–100, higher is healthier   ·   A ≥85  B ≥70  C ≥50  D ≥30  F <30\n\n")

	b.WriteString("  Dimension Scores:\n")
	if len(report.Dimensions) == 0 {
		b.WriteString("  (no dimensions collected)\n\n")
	} else {
		dimWidths := []int{15, 7, 8, 5}
		b.WriteString(tableBorder(dimWidths, "┌", "┬", "┐"))
		b.WriteString(tableRow(dimWidths, "lrrc", "Dimension", "Score", "Weight", "Grade"))
		b.WriteString(tableBorder(dimWidths, "├", "┼", "┤"))
		for _, d := range report.Dimensions {
			weight := fmt.Sprintf("%.1f%%", d.Weight)
			row := tableRow(dimWidths, "lrrc", string(d.Dimension), f.colorScore(d.Score), weight, scoreGrade(d.Score))
			if trend {
				row = strings.TrimRight(row, "\n") + "   " + formatDelta(d.Delta) + "\n"
			}
			b.WriteString(row)
		}
		b.WriteString(tableBorder(dimWidths, "└", "┴", "┘"))
		b.WriteString("\n")
	}

	b.WriteString("  Metric Details:\n")
	b.WriteString("  Grouped by dimension; raw values are per-repository, size-scaling metrics per-service.\n")
	f.writeMetricTable(&b, report)

	if lo, hi := scoredDimensionExtremes(report.Dimensions); lo < 10 && hi > 60 {
		fmt.Fprintf(&b, "  ⚠ Suspicious score spread: one dimension scored <10 while another >60 — likely a scoring-curve bug.\n\n")
	}

	if f.ShowExplain {
		b.WriteString("  Methodology:\n")
		for _, m := range report.Metrics {
			mt := methodology[m.Name]
			fmt.Fprintf(&b, "  %s\n", formatMetricName(m.Name))
			fmt.Fprintf(&b, "    Raw:    %s\n", mt.RawDef)
			fmt.Fprintf(&b, "    Score:  %s\n", mt.ScoreDef)
			fmt.Fprintf(&b, "    Source: %s\n", mt.Source)
		}
		b.WriteString("\n")
	}

	if len(report.Errors) > 0 {
		b.WriteString("  ⚠ Errors:\n")
		for _, e := range report.Errors {
			fmt.Fprintf(&b, "    - %s\n", e)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "  Collected: %s\n", report.CollectedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "  %s\n", head)

	return b.String()
}

// writeMetricTable renders metrics grouped under a per-dimension header row so a
// reader can see which metrics drive each dimension score.
func (f *TerminalFormatter) writeMetricTable(b *strings.Builder, report Report) {
	if len(report.Metrics) == 0 {
		b.WriteString("  (no metrics collected)\n\n")
		return
	}

	dims := map[model.Dimension]DimensionReport{}
	for _, d := range report.Dimensions {
		dims[d.Dimension] = d
	}
	byDim := map[model.Dimension][]MetricReport{}
	var order []model.Dimension
	for _, m := range report.Metrics {
		if _, seen := byDim[m.Dimension]; !seen {
			order = append(order, m.Dimension)
		}
		byDim[m.Dimension] = append(byDim[m.Dimension], m)
	}

	metWidths := []int{26, 9, 6, 10, 5, 7}
	b.WriteString(tableBorder(metWidths, "┌", "┬", "┐"))
	b.WriteString(tableRow(metWidths, "lrrlrr", "Metric", "Raw", "Score", "Unit", "Wt", "Impact"))
	for _, dim := range order {
		b.WriteString(tableBorder(metWidths, "├", "┴", "┤"))
		b.WriteString(tableSpan(metWidths, f.groupHeader(dim, dims[dim])))
		b.WriteString(tableBorder(metWidths, "├", "┬", "┤"))
		var displayOnly []MetricReport
		for _, m := range byDim[dim] {
			if isDisplayOnly(methodology[m.Name]) {
				displayOnly = append(displayOnly, m)
				continue
			}
			score := f.colorScore(m.Normalized)
			weightStr := ""
			if m.Weight > 0 && m.Weight != 1.0 {
				weightStr = fmt.Sprintf("%.1f", m.Weight)
			}
			impactStr := ""
			if dims[dim].WeightSum > 0 {
				impact := m.Normalized * m.Weight / dims[dim].WeightSum
				impactStr = fmt.Sprintf("%.1f", impact)
			}
			b.WriteString(tableRow(metWidths, "lrrlrr", formatMetricName(m.Name), formatRawValue(m.RawValue, m.Unit), score, formatUnit(m.Unit), weightStr, impactStr))
		}
		if len(displayOnly) > 0 {
			b.WriteString(tableBorder(metWidths, "├", "┬", "┤"))
			b.WriteString(tableSpan(metWidths, "Raw context (not scored)"))
			b.WriteString(tableBorder(metWidths, "├", "┬", "┤"))
			for _, m := range displayOnly {
				b.WriteString(tableRow(metWidths, "lrrlrr", formatMetricName(m.Name), formatRawValue(m.RawValue, m.Unit), "—", formatUnit(m.Unit), "", ""))
			}
		}
	}
	b.WriteString(tableBorder(metWidths, "└", "┴", "┘"))
	b.WriteString("\n")
}

func (f *TerminalFormatter) groupHeader(dim model.Dimension, d DimensionReport) string {
	h := titleDimension(dim)
	if d.MetricCount > 0 {
		h += fmt.Sprintf("   %s %s", f.colorScore(d.Score), scoreGrade(d.Score))
	}
	if d.Breakdown != "" {
		h += "   " + d.Breakdown
	}
	return h
}

func titleDimension(d model.Dimension) string {
	s := string(d)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// colorScore renders a score as a whole number: the underlying formulas carry
// no sub-point precision, so a decimal would imply rigor the model lacks.
func (f *TerminalFormatter) colorScore(score float64) string {
	if !f.UseColor {
		return fmt.Sprintf("%.0f", score)
	}
	switch {
	case score >= 75:
		return fmt.Sprintf("\033[32m%.0f\033[0m", score)
	case score >= 60:
		return fmt.Sprintf("\033[33m%.0f\033[0m", score)
	case score >= 40:
		return fmt.Sprintf("\033[38;5;208m%.0f\033[0m", score)
	default:
		return fmt.Sprintf("\033[31m%.0f\033[0m", score)
	}
}

var acronymReplacements = []struct {
	word string
	repl string
}{
	{"ci cd", "CI/CD"},
	{"k8s", "K8s"},
	{"prs", "PRs"},
}

// metricDisplayNames overrides the auto-formatted label where the raw metric
// name misframes what is measured.
var metricDisplayNames = map[model.MetricTypeName]string{
	model.MetricTypeCICDComplexity: "CI/CD Maturity",
}

func formatMetricName(name model.MetricTypeName) string {
	if override, ok := metricDisplayNames[name]; ok {
		return override
	}
	s := strings.ReplaceAll(string(name), "_", " ")

	for _, a := range acronymReplacements {
		s = strings.ReplaceAll(s, a.word, a.repl)
	}

	return formatMetricNameWords(s)
}

func formatMetricNameWords(s string) string {
	var result strings.Builder
	capitalize := true
	for _, r := range s {
		if r == ' ' {
			result.WriteRune(r)
			capitalize = true
			continue
		}
		if capitalize {
			result.WriteRune(unicode.ToUpper(r))
			capitalize = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func formatUnit(unit string) string {
	switch unit {
	case "per_week":
		return "/week"
	case "seconds":
		return "sec"
	case "ratio":
		return "%"
	default:
		return unit
	}
}

func formatRawValue(value float64, unit string) string {
	if value == -1.0 {
		return "—"
	}
	switch unit {
	case "ratio":
		return fmt.Sprintf("%.2f", value*100)
	case "seconds":
		return fmt.Sprintf("%.1f", value)
	default:
		if value == math.Trunc(value) {
			return fmt.Sprintf("%.1f", value)
		}
		return fmt.Sprintf("%.2f", value)
	}
}

// tableRow renders one row; aligns has one of l/r/c per column ('c' if missing).
func tableRow(widths []int, aligns string, cols ...string) string {
	var b strings.Builder
	b.WriteString("  │")
	for i, col := range cols {
		visible := stripANSI(col)
		if len(visible) > widths[i] {
			if widths[i] > 3 {
				visible = visible[:widths[i]-1] + "…"
			} else {
				visible = visible[:widths[i]]
			}
			col = visible
		}
		padding := widths[i] - len(visible)
		if padding < 0 {
			padding = 0
		}
		leftPad, rightPad := padding/2, padding-padding/2
		if i < len(aligns) {
			switch aligns[i] {
			case 'l':
				leftPad, rightPad = 0, padding
			case 'r':
				leftPad, rightPad = padding, 0
			}
		}
		fmt.Fprintf(&b, "%s%s%s", strings.Repeat(" ", leftPad+1), col, strings.Repeat(" ", rightPad+1))
		b.WriteString("│")
	}
	b.WriteString("\n")
	return b.String()
}

// tableSpan renders a single full-width cell, used for group header rows.
func tableSpan(widths []int, content string) string {
	inner := len(widths) - 1
	for _, w := range widths {
		inner += w + 2
	}
	pad := inner - 1 - len(stripANSI(content))
	if pad < 0 {
		pad = 0
	}
	return "  │ " + content + strings.Repeat(" ", pad) + "│\n"
}

func tableBorder(widths []int, left, mid, right string) string {
	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(left)
	for i, w := range widths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			b.WriteString(mid)
		}
	}
	b.WriteString(right)
	b.WriteString("\n")
	return b.String()
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
