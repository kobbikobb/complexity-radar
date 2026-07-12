package terminal

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/output"
)

type TerminalFormatter struct {
	UseColor bool
}

func New() *TerminalFormatter {
	return &TerminalFormatter{UseColor: true}
}

func (f *TerminalFormatter) Format(report output.Report) string {
	var b strings.Builder

	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("  ComplexityRadar Report\n")
	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Project: %s\n", report.ProjectName))
	if report.ProjectDescription != "" {
		b.WriteString(fmt.Sprintf("  %s\n", report.ProjectDescription))
	}
	b.WriteString("\n")

	b.WriteString("───────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  OVERALL SCORE: %s [%s]\n", f.colorScore(report.OverallScore), output.ScoreGrade(report.OverallScore)))
	b.WriteString("───────────────────────────────────────────────────\n")
	b.WriteString("\n")

	b.WriteString("  Dimension Scores:\n")
	dimWidths := []int{17, 7, 8, 6}
	b.WriteString(tableBorder(dimWidths, "┌", "┬", "┐"))
	b.WriteString(tableRow(dimWidths, "Dimension", "Score", "Weight", "Grade"))
	b.WriteString(tableBorder(dimWidths, "├", "┼", "┤"))
	for _, d := range report.Dimensions {
		weight := fmt.Sprintf("%.1f%%", d.Weight)
		b.WriteString(tableRow(dimWidths, string(d.Dimension), f.colorScore(d.Score), weight, fmt.Sprintf("  %s  ", output.ScoreGrade(d.Score))))
	}
	b.WriteString(tableBorder(dimWidths, "└", "┴", "┘"))
	b.WriteString("\n")

	b.WriteString("  Metric Details:\n")
	metWidths := []int{25, 11, 7, 7}
	b.WriteString(tableBorder(metWidths, "┌", "┬", "┐"))
	b.WriteString(tableRow(metWidths, "Metric", "Raw", "Score", "Unit"))
	b.WriteString(tableBorder(metWidths, "├", "┼", "┤"))
	for _, m := range report.Metrics {
		name := formatMetricName(m.Name)
		raw := formatRawValue(m.RawValue, m.Unit)
		unit := formatUnit(m.Unit)
		b.WriteString(tableRow(metWidths, name, raw, f.colorScore(m.Normalized), fmt.Sprintf("  %s  ", unit)))
	}
	b.WriteString(tableBorder(metWidths, "└", "┴", "┘"))
	b.WriteString("\n")

	if len(report.Errors) > 0 {
		b.WriteString("  ⚠ Errors:\n")
		for _, e := range report.Errors {
			b.WriteString(fmt.Sprintf("    - %s\n", e))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("  Collected: %s\n", report.CollectedAt.UTC().Format(time.RFC3339)))
	b.WriteString("═══════════════════════════════════════════════════\n")

	return b.String()
}

func (f *TerminalFormatter) colorScore(score float64) string {
	if !f.UseColor {
		return fmt.Sprintf("%.1f", score)
	}
	switch {
	case score >= 80:
		return fmt.Sprintf("\033[32m%.1f\033[0m", score)
	case score >= 60:
		return fmt.Sprintf("\033[33m%.1f\033[0m", score)
	default:
		return fmt.Sprintf("\033[31m%.1f\033[0m", score)
	}
}

func formatMetricName(name model.MetricTypeName) string {
	s := strings.ReplaceAll(string(name), "_", " ")
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
		padding := widths[i] - len(stripANSI(col))
		if padding < 0 {
			padding = 0
		}
		leftPad := padding / 2
		rightPad := padding - leftPad
		b.WriteString(fmt.Sprintf("%s%s%s", strings.Repeat(" ", leftPad+1), col, strings.Repeat(" ", rightPad+1)))
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
