package scorer

import (
	"math"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

const (
	refSecurityVulnerabilities = 20.0
	refStalePRs                = 20.0
	refBuildTime               = 1800.0
	refK8sDeployments          = 50.0
	refContainerImages         = 50.0
	refDeployTargets           = 20.0
	refDependencyCount         = 200.0
	refDeployFrequency         = 14.0
)

func NormalizeMetric(metricType model.MetricTypeName, value float64) float64 {
	switch metricType {
	case model.MetricTypeSecurityVulnerabilities:
		return clamp(100 - (value/refSecurityVulnerabilities)*100)
	case model.MetricTypeStalePRs:
		return clamp(100 - (value/refStalePRs)*100)
	case model.MetricTypeBuildTime:
		return clamp(100 - (value/refBuildTime)*100)
	case model.MetricTypeK8sDeployments:
		return clamp(100 - (value/refK8sDeployments)*100)
	case model.MetricTypeContainerImages:
		return clamp(100 - (value/refContainerImages)*100)
	case model.MetricTypeDeployTargets:
		return clamp(100 - (value/refDeployTargets)*100)
	case model.MetricTypeDependencyCount:
		return clamp(100 - (value/refDependencyCount)*100)
	case model.MetricTypeDeployFrequency:
		return clamp((value / refDeployFrequency) * 100)
	case model.MetricTypeBuildSuccessRatio:
		return clamp(value * 100)
	case model.MetricTypeCICDComplexity:
		return clamp(value)
	default:
		return 0
	}
}

func clamp(v float64) float64 {
	return math.Min(100, math.Max(0, v))
}
