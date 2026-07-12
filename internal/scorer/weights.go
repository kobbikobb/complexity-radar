package scorer

import (
	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
)

func WeightsFromConfig(cfg config.WeightsConfig) map[model.Dimension]float64 {
	return map[model.Dimension]float64{
		model.DimensionSecurity:       cfg.Security,
		model.DimensionDelivery:       cfg.Delivery,
		model.DimensionInfrastructure: cfg.Infrastructure,
		model.DimensionCode:           cfg.Code,
	}
}

func ScoreWithConfig(metrics map[model.MetricTypeName]float64, cfg config.WeightsConfig) ScoreResult {
	return Score(metrics, WeightsFromConfig(cfg))
}

func ScoreWithDefaults(metrics map[model.MetricTypeName]float64) ScoreResult {
	return ScoreWithConfig(metrics, config.DefaultWeights())
}
