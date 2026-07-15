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
		results[i] = DimensionResult{
			Dimension:   d,
			Score:       score,
			MetricCount: a.count,
		}
	}

	return results
}
