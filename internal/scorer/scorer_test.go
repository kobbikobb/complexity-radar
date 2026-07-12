package scorer

import (
	"math"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
)

const epsilon = 0.001

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// --- Normalization Tests ---

func TestNormalizeMetricLowerIsBetter(t *testing.T) {
	tests := []struct {
		name   string
		metric model.MetricTypeName
		value  float64
		want   float64
	}{
		{"zero security_vulns", model.MetricTypeSecurityVulnerabilities, 0, 100},
		{"half refMax security_vulns", model.MetricTypeSecurityVulnerabilities, 10, 50},
		{"at refMax security_vulns", model.MetricTypeSecurityVulnerabilities, 20, 0},
		{"over refMax security_vulns clamps to 0", model.MetricTypeSecurityVulnerabilities, 40, 0},

		{"zero stale_prs", model.MetricTypeStalePRs, 0, 100},
		{"half refMax stale_prs", model.MetricTypeStalePRs, 10, 50},
		{"at refMax stale_prs", model.MetricTypeStalePRs, 20, 0},

		{"zero build_time", model.MetricTypeBuildTime, 0, 100},
		{"half refMax build_time", model.MetricTypeBuildTime, 900, 50},
		{"at refMax build_time", model.MetricTypeBuildTime, 1800, 0},

		{"zero k8s_deployments", model.MetricTypeK8sDeployments, 0, 100},
		{"at refMax k8s_deployments", model.MetricTypeK8sDeployments, 50, 0},

		{"zero container_images", model.MetricTypeContainerImages, 0, 100},
		{"at refMax container_images", model.MetricTypeContainerImages, 50, 0},

		{"zero deploy_targets", model.MetricTypeDeployTargets, 0, 100},
		{"at refMax deploy_targets", model.MetricTypeDeployTargets, 20, 0},

		{"zero dependency_count", model.MetricTypeDependencyCount, 0, 100},
		{"at refMax dependency_count", model.MetricTypeDependencyCount, 200, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMetric(tt.metric, tt.value)
			if !approxEqual(got, tt.want) {
				t.Errorf("NormalizeMetric(%q, %v) = %v, want %v", tt.metric, tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeMetricHigherIsBetter(t *testing.T) {
	tests := []struct {
		name   string
		metric model.MetricTypeName
		value  float64
		want   float64
	}{
		{"zero build_success_ratio", model.MetricTypeBuildSuccessRatio, 0, 0},
		{"half build_success_ratio", model.MetricTypeBuildSuccessRatio, 0.5, 50},
		{"full build_success_ratio", model.MetricTypeBuildSuccessRatio, 1.0, 100},

		{"zero deploy_frequency", model.MetricTypeDeployFrequency, 0, 0},
		{"half refMax deploy_frequency", model.MetricTypeDeployFrequency, 7, 50},
		{"at refMax deploy_frequency", model.MetricTypeDeployFrequency, 14, 100},
		{"over refMax deploy_frequency clamps to 100", model.MetricTypeDeployFrequency, 28, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMetric(tt.metric, tt.value)
			if !approxEqual(got, tt.want) {
				t.Errorf("NormalizeMetric(%q, %v) = %v, want %v", tt.metric, tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeMetricClamped(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{"midpoint", 50, 50},
		{"over 100 clamps to 100", 150, 100},
		{"negative clamps to 0", -10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMetric(model.MetricTypeCICDComplexity, tt.value)
			if !approxEqual(got, tt.want) {
				t.Errorf("NormalizeMetric(ci_cd_complexity, %v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeMetricUnknown(t *testing.T) {
	got := NormalizeMetric("unknown_metric", 42)
	if got != 0 {
		t.Errorf("NormalizeMetric(unknown, 42) = %v, want 0", got)
	}
}

// --- Dimension Scoring Tests ---

func TestScoreDimensionsEmpty(t *testing.T) {
	results := ScoreDimensions(nil)
	if len(results) != 4 {
		t.Fatalf("got %d dimensions, want 4", len(results))
	}
	for _, r := range results {
		if r.Score != 0 {
			t.Errorf("dimension %q score = %v, want 0", r.Dimension, r.Score)
		}
		if r.MetricCount != 0 {
			t.Errorf("dimension %q metric count = %d, want 0", r.Dimension, r.MetricCount)
		}
	}
}

func TestScoreDimensionsSingleMetric(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0, // normalizes to 100
	}
	results := ScoreDimensions(metrics)

	var securityResult *DimensionResult
	for i := range results {
		if results[i].Dimension == model.DimensionSecurity {
			securityResult = &results[i]
		}
	}
	if securityResult == nil {
		t.Fatal("security dimension not found")
	}
	if !approxEqual(securityResult.Score, 100) {
		t.Errorf("security score = %v, want 100", securityResult.Score)
	}
	if securityResult.MetricCount != 1 {
		t.Errorf("security metric count = %d, want 1", securityResult.MetricCount)
	}
}

func TestScoreDimensionsMultipleMetrics(t *testing.T) {
	// delivery has deploy_frequency, build_success_ratio, build_time, stale_prs
	// Use two delivery metrics: build_success_ratio=1.0→100, stale_prs=0→100 → avg=100
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeBuildSuccessRatio: 1.0,
		model.MetricTypeStalePRs:          0,
	}
	results := ScoreDimensions(metrics)

	var deliveryResult *DimensionResult
	for i := range results {
		if results[i].Dimension == model.DimensionDelivery {
			deliveryResult = &results[i]
		}
	}
	if deliveryResult == nil {
		t.Fatal("delivery dimension not found")
	}
	if !approxEqual(deliveryResult.Score, 100) {
		t.Errorf("delivery score = %v, want 100 (avg of 100, 100)", deliveryResult.Score)
	}
	if deliveryResult.MetricCount != 2 {
		t.Errorf("delivery metric count = %d, want 2", deliveryResult.MetricCount)
	}
}

func TestScoreDimensionsMultipleMetricsUneven(t *testing.T) {
	// delivery: build_success_ratio=0.5→50, stale_prs=0→100 → avg=75
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeBuildSuccessRatio: 0.5,
		model.MetricTypeStalePRs:          0,
	}
	results := ScoreDimensions(metrics)

	var deliveryResult *DimensionResult
	for i := range results {
		if results[i].Dimension == model.DimensionDelivery {
			deliveryResult = &results[i]
		}
	}
	if !approxEqual(deliveryResult.Score, 75) {
		t.Errorf("delivery score = %v, want 75", deliveryResult.Score)
	}
}

func TestScoreDimensionsAllDimensions(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // security: 100
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          0,   // infrastructure: 100
		model.MetricTypeDependencyCount:         0,   // code: 100
	}
	results := ScoreDimensions(metrics)

	if len(results) != 4 {
		t.Fatalf("got %d dimensions, want 4", len(results))
	}

	for _, r := range results {
		if !approxEqual(r.Score, 100) {
			t.Errorf("dimension %q score = %v, want 100", r.Dimension, r.Score)
		}
		if r.MetricCount != 1 {
			t.Errorf("dimension %q metric count = %d, want 1", r.Dimension, r.MetricCount)
		}
	}
}

// --- Overall Scoring Tests ---

func TestScoreEqualWeights(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // security: 100
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          0,   // infrastructure: 100
		model.MetricTypeDependencyCount:         0,   // code: 100
	}
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0.25,
		model.DimensionDelivery:       0.25,
		model.DimensionInfrastructure: 0.25,
		model.DimensionCode:           0.25,
	}
	result := Score(metrics, weights)
	if !approxEqual(result.Overall, 100) {
		t.Errorf("overall = %v, want 100", result.Overall)
	}
}

func TestScoreUnequalWeights(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // security: 100
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          0,   // infrastructure: 100
		model.MetricTypeDependencyCount:         0,   // code: 100
	}
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0.50,
		model.DimensionDelivery:       0.30,
		model.DimensionInfrastructure: 0.10,
		model.DimensionCode:           0.10,
	}
	result := Score(metrics, weights)
	if !approxEqual(result.Overall, 100) {
		t.Errorf("overall = %v, want 100 (all dimensions score 100)", result.Overall)
	}
}

func TestScoreUnequalDimensionScores(t *testing.T) {
	// security=100, delivery=0, infra=0, code=0
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // 100
		model.MetricTypeBuildSuccessRatio:       0,   // delivery: 0
		model.MetricTypeK8sDeployments:          50,  // infra: 0
		model.MetricTypeDependencyCount:         200, // code: 0
	}
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0.25,
		model.DimensionDelivery:       0.25,
		model.DimensionInfrastructure: 0.25,
		model.DimensionCode:           0.25,
	}
	result := Score(metrics, weights)
	// only security contributes: 100 * 0.25 / 1.0 = 25
	if !approxEqual(result.Overall, 25) {
		t.Errorf("overall = %v, want 25", result.Overall)
	}
}

func TestScoreZeroWeights(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,
	}
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0,
		model.DimensionDelivery:       0,
		model.DimensionInfrastructure: 0,
		model.DimensionCode:           0,
	}
	result := Score(metrics, weights)
	if result.Overall != 0 {
		t.Errorf("overall = %v, want 0", result.Overall)
	}
}

func TestScoreWithDefaults(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,
		model.MetricTypeBuildSuccessRatio:       0.5,
		model.MetricTypeK8sDeployments:          25,
		model.MetricTypeDependencyCount:         100,
	}
	result := ScoreWithDefaults(metrics)
	// security=100, delivery=50, infra=50, code=50
	// overall = (100+50+50+50)/4 = 62.5
	if !approxEqual(result.Overall, 62.5) {
		t.Errorf("overall = %v, want 62.5", result.Overall)
	}
}

// --- Config Weights Tests ---

func TestWeightsFromConfig(t *testing.T) {
	cfg := config.WeightsConfig{
		Security:       0.40,
		Delivery:       0.30,
		Infrastructure: 0.20,
		Code:           0.10,
	}
	w := WeightsFromConfig(cfg)

	if !approxEqual(w[model.DimensionSecurity], 0.40) {
		t.Errorf("security weight = %v, want 0.40", w[model.DimensionSecurity])
	}
	if !approxEqual(w[model.DimensionDelivery], 0.30) {
		t.Errorf("delivery weight = %v, want 0.30", w[model.DimensionDelivery])
	}
	if !approxEqual(w[model.DimensionInfrastructure], 0.20) {
		t.Errorf("infrastructure weight = %v, want 0.20", w[model.DimensionInfrastructure])
	}
	if !approxEqual(w[model.DimensionCode], 0.10) {
		t.Errorf("code weight = %v, want 0.10", w[model.DimensionCode])
	}
}

func TestScoreWithConfig(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // security: 100
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          0,   // infrastructure: 100
		model.MetricTypeDependencyCount:         0,   // code: 100
	}
	cfg := config.WeightsConfig{
		Security:       0.40,
		Delivery:       0.30,
		Infrastructure: 0.20,
		Code:           0.10,
	}
	result := ScoreWithConfig(metrics, cfg)
	if !approxEqual(result.Overall, 100) {
		t.Errorf("overall = %v, want 100", result.Overall)
	}
}

func TestScoreWithConfigUnequalScores(t *testing.T) {
	// All delivery metrics at best → delivery=100, others at worst → 0
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 20,  // security: 0
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          50,  // infra: 0
		model.MetricTypeDependencyCount:         200, // code: 0
	}
	cfg := config.WeightsConfig{
		Security:       0.25,
		Delivery:       0.25,
		Infrastructure: 0.25,
		Code:           0.25,
	}
	result := ScoreWithConfig(metrics, cfg)
	// delivery contributes: 100 * 0.25 = 25
	if !approxEqual(result.Overall, 25) {
		t.Errorf("overall = %v, want 25", result.Overall)
	}
}

// --- Edge Cases ---

func TestScoreNilMetrics(t *testing.T) {
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0.25,
		model.DimensionDelivery:       0.25,
		model.DimensionInfrastructure: 0.25,
		model.DimensionCode:           0.25,
	}
	result := Score(nil, weights)
	if result.Overall != 0 {
		t.Errorf("overall = %v, want 0 for nil metrics", result.Overall)
	}
	if len(result.Dimensions) != 4 {
		t.Errorf("got %d dimensions, want 4", len(result.Dimensions))
	}
}

func TestScoreBoundaryValues(t *testing.T) {
	// Best values: all lower-is-better at 0, higher-is-better at max
	// ci_cd_complexity=0 → score 0 (raw value, not inverted)
	bestMetrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // 100
		model.MetricTypeStalePRs:                0,   // 100
		model.MetricTypeBuildTime:               0,   // 100
		model.MetricTypeK8sDeployments:          0,   // 100
		model.MetricTypeContainerImages:         0,   // 100
		model.MetricTypeDeployTargets:           0,   // 100
		model.MetricTypeDependencyCount:         0,   // 100
		model.MetricTypeBuildSuccessRatio:       1.0, // 100
		model.MetricTypeDeployFrequency:         14,  // 100
		model.MetricTypeCICDComplexity:          0,   // 0 (raw clamp)
	}
	result := ScoreWithDefaults(bestMetrics)
	// security=100, delivery=100, infra=(100+100+100+0)/4=75, code=100
	// overall = (100+100+75+100)/4 = 93.75
	if !approxEqual(result.Overall, 93.75) {
		t.Errorf("best case overall = %v, want 93.75", result.Overall)
	}

	// Worst values: all lower-is-better at refMax, higher-is-better at 0
	worstMetrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 20,   // 0
		model.MetricTypeStalePRs:                20,   // 0
		model.MetricTypeBuildTime:               1800, // 0
		model.MetricTypeK8sDeployments:          50,   // 0
		model.MetricTypeContainerImages:         50,   // 0
		model.MetricTypeDeployTargets:           20,   // 0
		model.MetricTypeDependencyCount:         200,  // 0
		model.MetricTypeBuildSuccessRatio:       0,    // 0
		model.MetricTypeDeployFrequency:         0,    // 0
		model.MetricTypeCICDComplexity:          100,  // 100 (raw clamp)
	}
	result = ScoreWithDefaults(worstMetrics)
	// security=0, delivery=0, infra=(0+0+0+100)/4=25, code=0
	// overall = (0+0+25+0)/4 = 6.25
	if !approxEqual(result.Overall, 6.25) {
		t.Errorf("worst case overall = %v, want 6.25", result.Overall)
	}
}
