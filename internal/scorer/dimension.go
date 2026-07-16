package scorer

import (
	"math"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

type DimensionResult struct {
	Dimension   model.Dimension
	Score       float64
	MetricCount int
}

// securityCriticalCap gates the security dimension into the F band (grade
// threshold is 30) when any open critical vulnerability exists: a critical is a
// failing condition on its own, but its severity is diluted inside the weighted
// security_vulnerabilities sum, so averaging alone lets criticals sit in D.
const securityCriticalCap = 29.0

func ScoreDimensions(metrics map[model.MetricTypeName]float64) []DimensionResult {
	dimensions := []model.Dimension{
		model.DimensionSecurity,
		model.DimensionDelivery,
		model.DimensionInfrastructure,
		model.DimensionCode,
	}

	type meta struct {
		dim    model.Dimension
		weight float64
	}
	metricMeta := make(map[model.MetricTypeName]meta)
	for _, mt := range model.MetricTypes() {
		w := mt.Weight
		if w <= 0 {
			w = 1.0
		}
		metricMeta[mt.Name] = meta{dim: mt.Dimension, weight: w}
	}

	type acc struct {
		weighted, weight float64
		count            int
	}
	grouped := make(map[model.Dimension]*acc)
	for _, d := range dimensions {
		grouped[d] = &acc{}
	}

	for name, value := range metrics {
		m, ok := metricMeta[name]
		if !ok {
			continue
		}
		normalized := NormalizeMetric(name, value)
		if math.IsNaN(normalized) {
			continue
		}
		a := grouped[m.dim]
		a.weighted += normalized * m.weight
		a.weight += m.weight
		a.count++
	}

	results := make([]DimensionResult, len(dimensions))
	for i, d := range dimensions {
		a := grouped[d]
		score := 0.0
		if a.weight > 0 {
			score = a.weighted / a.weight
		}
		// > 0 (not >= 1): the project rollup averages the per-repo critical count,
		// so one critical across many repos arrives as a fraction but must still gate.
		if d == model.DimensionSecurity && a.count > 0 && metrics[model.MetricTypeSecurityCritical] > 0 {
			score = math.Min(score, securityCriticalCap)
		}
		results[i] = DimensionResult{
			Dimension:   d,
			Score:       score,
			MetricCount: a.count,
		}
	}

	return results
}
