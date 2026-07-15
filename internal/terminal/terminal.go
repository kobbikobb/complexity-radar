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
}

type DimensionReport struct {
	Dimension   model.Dimension
	Score       float64
	Weight      float64
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
}

type OutputFormatter interface {
	Format(report Report) string
}

func scoreGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
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

func (f *TerminalFormatter) Format(report Report) string {
	var b strings.Builder

	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("  ComplexityRadar Report\n")
	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Project: %s\n", report.ProjectName)
	if report.ProjectDescription != "" {
		fmt.Fprintf(&b, "  %s\n", report.ProjectDescription)
	}
	b.WriteString("\n")

	trend := f.ShowTrend && report.HasTrend

	b.WriteString("───────────────────────────────────────────────────\n")
	fmt.Fprintf(&b, "  OVERALL SCORE: %s [%s]", f.colorScore(report.OverallScore), overallGrade(report.OverallScore, report.Dimensions))
	if trend {
		fmt.Fprintf(&b, "   %s vs previous", formatDelta(report.OverallDelta))
	}
	b.WriteString("\n")
	b.WriteString("───────────────────────────────────────────────────\n")
	b.WriteString("  Scores 0–100, higher is healthier.\n")
	b.WriteString("  A ≥90   B ≥75   C ≥60   D ≥40   F <40\n")
	b.WriteString("\n")

	b.WriteString("  Dimension Scores:\n")
	dimWidths := []int{17, 7, 8, 6}
	b.WriteString(tableBorder(dimWidths, "┌", "┬", "┐"))
	b.WriteString(tableRow(dimWidths, "Dimension", "Score", "Weight", "Grade"))
	b.WriteString(tableBorder(dimWidths, "├", "┼", "┤"))
	for _, d := range report.Dimensions {
		weight := fmt.Sprintf("%.1f%%", d.Weight)
		row := tableRow(dimWidths, string(d.Dimension), f.colorScore(d.Score), weight, fmt.Sprintf("  %s  ", scoreGrade(d.Score)))
		suffix := ""
		if trend {
			suffix += "  " + formatDelta(d.Delta)
		}
		if d.Breakdown != "" {
			suffix += "  — " + d.Breakdown
		}
		if suffix != "" {
			row = strings.TrimRight(row, "\n") + suffix + "\n"
		}
		b.WriteString(row)
	}
	b.WriteString(tableBorder(dimWidths, "└", "┴", "┘"))
	b.WriteString("\n")

	b.WriteString("  Metric Details:\n")
	b.WriteString("  Raw values are per-repository totals.\n")
	metWidths := []int{25, 11, 7, 7}
	b.WriteString(tableBorder(metWidths, "┌", "┬", "┐"))
	b.WriteString(tableRow(metWidths, "Metric", "Raw", "Score", "Unit"))
	b.WriteString(tableBorder(metWidths, "├", "┼", "┤"))
	for _, m := range report.Metrics {
		name := formatMetricName(m.Name)
		raw := formatRawValue(m.RawValue, m.Unit)
		unit := formatUnit(m.Unit)
		b.WriteString(tableRow(metWidths, name, raw, f.colorScore(m.Normalized), unit))
	}
	b.WriteString(tableBorder(metWidths, "└", "┴", "┘"))
	b.WriteString("\n")

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
	b.WriteString("═══════════════════════════════════════════════════\n")

	return b.String()
}

func (f *TerminalFormatter) colorScore(score float64) string {
	if !f.UseColor {
		return fmt.Sprintf("%.1f", score)
	}
	switch {
	case score >= 75:
		return fmt.Sprintf("\033[32m%.1f\033[0m", score)
	case score >= 60:
		return fmt.Sprintf("\033[33m%.1f\033[0m", score)
	default:
		return fmt.Sprintf("\033[31m%.1f\033[0m", score)
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

func formatMetricName(name model.MetricTypeName) string {
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

func tableRow(widths []int, cols ...string) string {
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
		leftPad := padding / 2
		rightPad := padding - leftPad
		fmt.Fprintf(&b, "%s%s%s", strings.Repeat(" ", leftPad+1), col, strings.Repeat(" ", rightPad+1))
		b.WriteString("│")
	}
	b.WriteString("\n")
	return b.String()
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
