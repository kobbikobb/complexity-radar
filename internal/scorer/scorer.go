package scorer

import "github.com/kobbikobb/complexity-radar/internal/model"

type ScoreResult struct {
	Overall    float64
	Dimensions []DimensionResult
}

func Score(metrics map[model.MetricTypeName]float64, weights map[model.Dimension]float64) ScoreResult {
	dimensions := ScoreDimensions(metrics)

	totalWeight := 0.0
	for _, d := range dimensions {
		if w, ok := weights[d.Dimension]; ok {
			totalWeight += w
		}
	}

	if totalWeight == 0 {
		return ScoreResult{Dimensions: dimensions}
	}

	needsNormalize := totalWeight != 1.0
	overall := 0.0
	for _, d := range dimensions {
		w := weights[d.Dimension]
		if needsNormalize {
			w /= totalWeight
		}
		overall += d.Score * w
	}

	return ScoreResult{
		Overall:    overall,
		Dimensions: dimensions,
	}
}
