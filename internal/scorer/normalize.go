package scorer

import (
	"math"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

const (
	refSecurityVulnerabilities = 70.0
	refStalePRs                = 45.0
	refBuildTime               = 1800.0
	refK8sDeployments          = 100.0
	refContainerImages         = 40.0
	refDeployTargets           = 20.0
	refDependencyCount         = 8.0   // deps-per-service ratio
	refDeployFrequency         = 5.0   // weekday deploy = full marks, not CD cadence
	refCICDComplexity          = 100.0 // CI/CD automation maturity (higher is better); ref matches cicd.go's raw cap of 100
	refSecurityCritical        = 5.0
	refSecurityHigh            = 40.0
	refSecurityMedium          = 60.0
	refSecurityLow             = 40.0
	refDecisionDensity         = 20.0
)

func NormalizeMetric(metricType model.MetricTypeName, value float64) float64 {
	switch metricType {
	case model.MetricTypeSecurityVulnerabilities:
		return asymptotic(value, refSecurityVulnerabilities)
	case model.MetricTypeStalePRs:
		return asymptotic(value, refStalePRs)
	case model.MetricTypeBuildTime:
		return clamp(100 - (value/refBuildTime)*100)
	case model.MetricTypeK8sDeployments:
		return asymptotic(value, refK8sDeployments)
	case model.MetricTypeContainerImages:
		return asymptotic(value, refContainerImages)
	case model.MetricTypeDeployTargets:
		return asymptotic(value, refDeployTargets)
	case model.MetricTypeDependencyCount:
		return asymptotic(value, refDependencyCount)
	case model.MetricTypeDecisionDensity:
		return asymptotic(value, refDecisionDensity)
	case model.MetricTypeDeployFrequency:
		if value < 0 {
			return math.NaN()
		}
		return clamp((value / refDeployFrequency) * 100)
	case model.MetricTypeBuildSuccessRatio:
		return clamp(value * 100)
	case model.MetricTypeCICDComplexity:
		return clamp(logNormalize(value, refCICDComplexity) * 100)
	case model.MetricTypeSecurityCritical:
		return asymptotic(value, refSecurityCritical)
	case model.MetricTypeSecurityHigh:
		return asymptotic(value, refSecurityHigh)
	case model.MetricTypeSecurityMedium:
		return asymptotic(value, refSecurityMedium)
	case model.MetricTypeSecurityLow:
		return asymptotic(value, refSecurityLow)
	default:
		return 0
	}
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
