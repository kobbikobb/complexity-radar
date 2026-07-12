package scorer

import "github.com/kobbikobb/complexity-radar/internal/model"

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

	metricTypeMap := make(map[model.MetricTypeName]model.Dimension)
	for _, mt := range model.MetricTypes() {
		metricTypeMap[mt.Name] = mt.Dimension
	}

	grouped := make(map[model.Dimension][]float64)
	for _, d := range dimensions {
		grouped[d] = nil
	}

	for name, value := range metrics {
		dim, ok := metricTypeMap[name]
		if !ok {
			continue
		}
		normalized := NormalizeMetric(name, value)
		grouped[dim] = append(grouped[dim], normalized)
	}

	results := make([]DimensionResult, len(dimensions))
	for i, d := range dimensions {
		values := grouped[d]
		score := 0.0
		if len(values) > 0 {
			sum := 0.0
			for _, v := range values {
				sum += v
			}
			score = sum / float64(len(values))
		}
		results[i] = DimensionResult{
			Dimension:   d,
			Score:       score,
			MetricCount: len(values),
		}
	}

	return results
}
