// Package htmlreport renders complexity reports as a single self-contained
// HTML file (all CSS inline, no external resources) safe to email and share.
package htmlreport

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/terminal"
)

type pageData struct {
	Project   reportVM
	Repos     []reportVM
	Generated string
}

type reportVM struct {
	Name        string
	Description string
	Aggregate   bool
	Overall     float64
	Grade       string
	Band        string
	Collected   string
	Dimensions  []dimVM
	Groups      []groupVM
	Errors      []string
}

type dimVM struct {
	Name        string
	Score       float64
	Weight      float64
	Grade       string
	Band        string
	Breakdown   string
	MetricCount int
}

type groupVM struct {
	Dimension string
	Metrics   []metricVM
	Details   []metricVM
}

type metricVM struct {
	Name        string
	Raw         string
	Unit        string
	Score       string
	Band        string
	DisplayOnly bool
	ScoreDef    string
	Tooltip     string
	Weight      string
	Impact      string
}

var methodology = func() map[model.MetricTypeName]model.MetricType {
	m := map[model.MetricTypeName]model.MetricType{}
	for _, mt := range append(model.MetricTypes(), model.DisplayMetricTypes()...) {
		m[mt.Name] = mt
	}
	return m
}()

var metricOrder = func() map[model.MetricTypeName]int {
	m := map[model.MetricTypeName]int{}
	for i, mt := range append(model.MetricTypes(), model.DisplayMetricTypes()...) {
		m[mt.Name] = i
	}
	return m
}()

func band(score float64) string {
	switch {
	case score >= 75:
		return "good"
	case score >= 40:
		return "warn"
	default:
		return "bad"
	}
}

func isDisplayOnly(mt model.MetricType) bool {
	return strings.Contains(mt.ScoreDef, "display-only")
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func toMetricVM(m terminal.MetricReport, weightSum float64) metricVM {
	mt := methodology[m.Name]
	mv := metricVM{
		Name:        terminal.MetricDisplayName(m.Name),
		Raw:         terminal.RawValueDisplay(m.RawValue, m.Unit),
		Unit:        terminal.UnitDisplay(m.Unit),
		ScoreDef:    mt.ScoreDef,
		Tooltip:     "Raw: " + mt.RawDef + "\nSource: " + mt.Source,
		DisplayOnly: isDisplayOnly(mt),
	}
	if mv.DisplayOnly {
		mv.Score = "—"
		mv.Band = "empty"
	} else {
		mv.Score = terminal.RawValueDisplay(m.Normalized, "score")
		mv.Band = band(m.Normalized)
	}
	if m.Weight > 0 && m.Weight != 1.0 {
		mv.Weight = fmt.Sprintf("%.1f", m.Weight)
	}
	if !mv.DisplayOnly && weightSum > 0 {
		impact := m.Normalized * m.Weight / weightSum
		mv.Impact = fmt.Sprintf("%.1f", impact)
	}
	return mv
}

func toReportVM(r terminal.Report) reportVM {
	vm := reportVM{
		Name:        r.ProjectName,
		Description: r.ProjectDescription,
		Aggregate:   r.Aggregate,
		Overall:     r.OverallScore,
		Grade:       terminal.OverallGrade(r.OverallScore, r.Dimensions),
		Band:        band(r.OverallScore),
		Collected:   r.CollectedAt.UTC().Format(time.RFC3339),
		Errors:      r.Errors,
	}

	for _, d := range r.Dimensions {
		dv := dimVM{
			Name:        titleCase(string(d.Dimension)),
			Score:       d.Score,
			Weight:      d.Weight,
			Grade:       terminal.Grade(d.Score),
			Band:        band(d.Score),
			Breakdown:   d.Breakdown,
			MetricCount: d.MetricCount,
		}
		if d.MetricCount == 0 {
			dv.Band = "empty"
		}
		vm.Dimensions = append(vm.Dimensions, dv)
	}

	byDim := map[model.Dimension][]terminal.MetricReport{}
	for _, m := range r.Metrics {
		byDim[m.Dimension] = append(byDim[m.Dimension], m)
	}

	for _, d := range r.Dimensions {
		metrics := byDim[d.Dimension]
		if len(metrics) == 0 {
			continue
		}
		sort.SliceStable(metrics, func(i, j int) bool {
			return metricOrder[metrics[i].Name] < metricOrder[metrics[j].Name]
		})
		g := groupVM{Dimension: titleCase(string(d.Dimension))}
		for _, m := range metrics {
			mv := toMetricVM(m, d.WeightSum)
			if mv.DisplayOnly {
				g.Details = append(g.Details, mv)
			} else {
				g.Metrics = append(g.Metrics, mv)
			}
		}
		vm.Groups = append(vm.Groups, g)
	}

	return vm
}

// Render produces a self-contained HTML document. project is the aggregate
// rollup; repos are the per-repository reports.
func Render(project terminal.Report, repos []terminal.Report) (string, error) {
	data := pageData{
		Project:   toReportVM(project),
		Generated: time.Now().UTC().Format(time.RFC3339),
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

var tmpl = template.Must(template.New("page").Parse(pageTemplate))
