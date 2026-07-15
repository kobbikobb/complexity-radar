package scorer

import (
	"math"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

const (
	refSecurityVulnerabilities = 50.0
	refStalePRs                = 30.0
	refBuildTime               = 1800.0
	refK8sDeployments          = 200.0
	refContainerImages         = 200.0
	refDeployTargets           = 50.0
	refDependencyCount         = 20.0  // provisional; deps-per-service ratio is small (was 1500 for the raw total), needs calibration against a real run
	refDeployFrequency         = 5.0   // weekday deploy = full marks, not CD cadence
	refCICDComplexity          = 100.0 // CI/CD automation maturity (higher is better); ref matches cicd.go's raw cap of 100
	refSecurityCritical        = 5.0
	refSecurityHigh            = 10.0
	refSecurityMedium          = 20.0
	refSecurityLow             = 30.0
	refDecisionDensity         = 15.0 // provisional; density is small (~5-20 tokens per 100 non-blank lines), needs calibration against a real run
)

func NormalizeMetric(metricType model.MetricTypeName, value float64) float64 {
	switch metricType {
	case model.MetricTypeSecurityVulnerabilities:
		return clamp(100 - logNormalize(value, refSecurityVulnerabilities)*100)
	case model.MetricTypeStalePRs:
		return clamp(100 - logNormalize(value, refStalePRs)*100)
	case model.MetricTypeBuildTime:
		return clamp(100 - (value/refBuildTime)*100)
	case model.MetricTypeK8sDeployments:
		return clamp(100 - logNormalize(value, refK8sDeployments)*100)
	case model.MetricTypeContainerImages:
		return clamp(100 - logNormalize(value, refContainerImages)*100)
	case model.MetricTypeDeployTargets:
		return clamp(100 - logNormalize(value, refDeployTargets)*100)
	case model.MetricTypeDependencyCount:
		return clamp(100 - logNormalize(value, refDependencyCount)*100)
	case model.MetricTypeDecisionDensity:
		if value < 0 {
			return math.NaN()
		}
		return clamp(100 - logNormalize(value, refDecisionDensity)*100)
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
		return clamp(100 - logNormalize(value, refSecurityCritical)*100)
	case model.MetricTypeSecurityHigh:
		return clamp(100 - logNormalize(value, refSecurityHigh)*100)
	case model.MetricTypeSecurityMedium:
		return clamp(100 - logNormalize(value, refSecurityMedium)*100)
	case model.MetricTypeSecurityLow:
		return clamp(100 - logNormalize(value, refSecurityLow)*100)
	default:
		return 0
	}
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
