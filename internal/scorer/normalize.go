package scorer

import (
	"math"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

var normalizers = map[model.MetricTypeName]func(float64) float64{
	model.MetricTypeSecurityVulnerabilities: func(v float64) float64 { return asymptotic(v, 25) },
	model.MetricTypeStalePRs:                func(v float64) float64 { return asymptotic(v, 45) },
	model.MetricTypeBuildTime:               func(v float64) float64 { return clamp(100 - (v/1800)*100) },
	model.MetricTypeK8sDeployments:          func(v float64) float64 { return asymptotic(v, 10) },
	model.MetricTypeContainerImages:         func(v float64) float64 { return asymptotic(v, 3) },
	model.MetricTypeDeployTargets:           func(v float64) float64 { return asymptotic(v, 20) },
	model.MetricTypeDependencyCount:         func(v float64) float64 { return asymptotic(v, 15) },
	model.MetricTypeDecisionDensity:         func(v float64) float64 { return asymptotic(v, 20) },
	model.MetricTypeFeatureFlagDebt:         func(v float64) float64 { return asymptotic(v, 15) },
	model.MetricTypeDeployFrequency: func(v float64) float64 {
		if v < 0 {
			return math.NaN()
		}
		return clamp((v / 5) * 100)
	},
	model.MetricTypeBuildSuccessRatio: func(v float64) float64 { return clamp(v * 100) },
	model.MetricTypeCICDComplexity:    func(v float64) float64 { return clamp(logNormalize(v, 100) * 100) },
	model.MetricTypeSecurityCritical:  func(v float64) float64 { return asymptotic(v, 5) },
	model.MetricTypeSecurityHigh:      func(v float64) float64 { return asymptotic(v, 40) },
	model.MetricTypeSecurityMedium:    func(v float64) float64 { return asymptotic(v, 60) },
	model.MetricTypeSecurityLow:       func(v float64) float64 { return asymptotic(v, 40) },
}

func NormalizeMetric(metricType model.MetricTypeName, value float64) float64 {
	if fn, ok := normalizers[metricType]; ok {
		return fn(value)
	}
	return 0
}

// asymptotic scores a lower-is-better metric as 100*ref/(value+ref): 100 at 0,
// 50 at ref, approaching but never reaching 0 as value grows. A negative value
// (no-data sentinel) yields NaN so it is skipped rather than scored.
func asymptotic(value, ref float64) float64 {
	if value < 0 {
		return math.NaN()
	}
	if ref <= 0 {
		return 0
	}
	return 100 * ref / (value + ref)
}

// logNormalize normalizes a value using log scale to reduce impact of large counts.
// Returns 0-1 where 1 means value >= ref.
func logNormalize(value, ref float64) float64 {
	if value <= 0 {
		return 0
	}
	if ref <= 0 {
		return 0
	}
	// log(1 + value) / log(1 + ref) gives smooth scaling
	return math.Log(1+value) / math.Log(1+ref)
}

func clamp(v float64) float64 {
	return math.Min(100, math.Max(0, v))
}
