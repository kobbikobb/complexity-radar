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
		name     string
		metric   model.MetricTypeName
		value    float64
		minScore float64
		maxScore float64
	}{
		// Asymptotic: 0 → 100, ref → 50, never floors to 0
		{"zero security_vulns", model.MetricTypeSecurityVulnerabilities, 0, 100, 100},
		{"low security_vulns", model.MetricTypeSecurityVulnerabilities, 0.5, 88, 93},
		{"at ref security_vulns", model.MetricTypeSecurityVulnerabilities, 5, 48, 52},

		{"zero stale_prs", model.MetricTypeStalePRs, 0, 100, 100},
		{"low stale_prs", model.MetricTypeStalePRs, 5, 85, 92},
		{"at ref stale_prs", model.MetricTypeStalePRs, 45, 48, 52},

		{"zero build_time", model.MetricTypeBuildTime, 0, 100, 100},
		{"half refMax build_time", model.MetricTypeBuildTime, 900, 45, 55},
		{"at refMax build_time", model.MetricTypeBuildTime, 1800, 0, 5},

		{"zero k8s_deployments", model.MetricTypeK8sDeployments, 0, 100, 100},
		{"low k8s_deployments", model.MetricTypeK8sDeployments, 0.5, 88, 93},
		{"at ref k8s_deployments", model.MetricTypeK8sDeployments, 5, 48, 52},

		{"zero container_images", model.MetricTypeContainerImages, 0, 100, 100},
		{"low container_images", model.MetricTypeContainerImages, 0.5, 82, 88},

		{"zero deploy_targets", model.MetricTypeDeployTargets, 0, 100, 100},
		{"low deploy_targets", model.MetricTypeDeployTargets, 5, 75, 82},

		{"zero dependency_count", model.MetricTypeDependencyCount, 0, 100, 100},
		{"low dependency_count", model.MetricTypeDependencyCount, 5, 58, 65},
		{"at ref dependency_count", model.MetricTypeDependencyCount, 8, 48, 52},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMetric(tt.metric, tt.value)
			if got < tt.minScore || got > tt.maxScore {
				t.Errorf("NormalizeMetric(%q, %v) = %v, want between %v and %v", tt.metric, tt.value, got, tt.minScore, tt.maxScore)
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
		{"half refMax deploy_frequency", model.MetricTypeDeployFrequency, 2.5, 50},
		{"at refMax deploy_frequency", model.MetricTypeDeployFrequency, 5, 100},
		{"over refMax deploy_frequency clamps to 100", model.MetricTypeDeployFrequency, 10, 100},

		{"zero ci_cd_complexity", model.MetricTypeCICDComplexity, 0, 0},
		{"at raw cap ci_cd_complexity", model.MetricTypeCICDComplexity, 100, 100},
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
		min   float64
		max   float64
	}{
		{"low complexity", 10, 30, 70},
		{"at raw cap reaches full score", 100, 95, 100},
		{"over raw cap clamps to 100", 300, 95, 100},
		{"negative clamps to 0", -10, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMetric(model.MetricTypeCICDComplexity, tt.value)
			if got < tt.min || got > tt.max {
				t.Errorf("NormalizeMetric(ci_cd_complexity, %v) = %v, want between %v and %v", tt.value, got, tt.min, tt.max)
			}
		})
	}
}

func TestNormalizeMetricDecisionDensity(t *testing.T) {
	low := NormalizeMetric(model.MetricTypeDecisionDensity, 10)
	high := NormalizeMetric(model.MetricTypeDecisionDensity, 200)

	if low <= high {
		t.Errorf("expected higher density to score lower: low(10)=%v high(200)=%v", low, high)
	}
	if got := NormalizeMetric(model.MetricTypeDecisionDensity, 0); !approxEqual(got, 100) {
		t.Errorf("value 0 = %v, want 100", got)
	}
	if got := NormalizeMetric(model.MetricTypeDecisionDensity, -1.0); !math.IsNaN(got) {
		t.Errorf("no-data = %v, want NaN", got)
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

func TestScoreDimensionsCode(t *testing.T) {
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeDependencyCount: 0, // 100
		model.MetricTypeDecisionDensity: 0, // 100
	}
	results := ScoreDimensions(metrics)

	var code *DimensionResult
	for i := range results {
		if results[i].Dimension == model.DimensionCode {
			code = &results[i]
		}
	}
	if code == nil {
		t.Fatal("code dimension not found")
	}
	if !approxEqual(code.Score, 100) {
		t.Errorf("code score = %v, want 100", code.Score)
	}
	if code.MetricCount != 2 {
		t.Errorf("code metric count = %d, want 2 (dependency_count + decision_density)", code.MetricCount)
	}
}

func TestScoreDimensionsCriticalVulnGatesSecurityToF(t *testing.T) {
	// Arrange: many lesser vulns average to a passing weighted sum, but a
	// critical is present.
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 40, // asymptotic → ~64 (would be D/C)
		model.MetricTypeSecurityCritical:        4,
	}

	// Act
	results := ScoreDimensions(metrics)

	// Assert
	var security *DimensionResult
	for i := range results {
		if results[i].Dimension == model.DimensionSecurity {
			security = &results[i]
		}
	}
	if security == nil {
		t.Fatal("security dimension not found")
	}
	if security.Score >= 40 {
		t.Errorf("security score = %v, want < 40 (gated to F by open critical)", security.Score)
	}
}

func TestScoreDimensionsFractionalCriticalGates(t *testing.T) {
	// Arrange: project rollup averages the per-repo critical count, so one
	// critical across ten repos arrives as 0.1.
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,
		model.MetricTypeSecurityCritical:        0.1,
	}

	// Act
	results := ScoreDimensions(metrics)

	// Assert
	for i := range results {
		if results[i].Dimension == model.DimensionSecurity && results[i].Score >= 40 {
			t.Errorf("security score = %v, want < 40 (fractional critical must gate)", results[i].Score)
		}
	}
}

func TestScoreDimensionsNoCriticalDoesNotGate(t *testing.T) {
	// Arrange
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 2, // per-service; normalizes to ~71
		model.MetricTypeSecurityCritical:        0,
	}

	// Act
	results := ScoreDimensions(metrics)

	// Assert
	var security *DimensionResult
	for i := range results {
		if results[i].Dimension == model.DimensionSecurity {
			security = &results[i]
		}
	}
	if security.Score < 40 {
		t.Errorf("security score = %v, want ungated (>= 40) with zero criticals", security.Score)
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
	// security=100, delivery=0, infra=50, code≈4
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // 100
		model.MetricTypeBuildSuccessRatio:       0,   // delivery: 0
		model.MetricTypeK8sDeployments:          5,   // infra: 50 (asymptotic, ref 5)
		model.MetricTypeDependencyCount:         200, // code: ~4 (well above ref 8)
	}
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0.25,
		model.DimensionDelivery:       0.25,
		model.DimensionInfrastructure: 0.25,
		model.DimensionCode:           0.25,
	}
	result := Score(metrics, weights)
	// (100 + 0 + 50 + ~4) * 0.25 ≈ 39
	if result.Overall < 30 || result.Overall > 55 {
		t.Errorf("overall = %v, want between 30 and 55", result.Overall)
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
		model.MetricTypeK8sDeployments:          2,
		model.MetricTypeDependencyCount:         2,
	}
	result := ScoreWithDefaults(metrics)
	// security=100, delivery=50, infra=~71 (k8s 2/svc, ref 5), code=~80 (deps 2, ref 8)
	if result.Overall < 50 || result.Overall > 85 {
		t.Errorf("overall = %v, want between 50 and 85", result.Overall)
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
	// Delivery best → 100; security/infra mid, code low
	metrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 5,   // security: 50 (per-service, ref 5)
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          5,   // infra: 50 (per-service, ref 5)
		model.MetricTypeDependencyCount:         500, // code: ~2 (asymptotic, never floors)
	}
	cfg := config.WeightsConfig{
		Security:       0.25,
		Delivery:       0.25,
		Infrastructure: 0.25,
		Code:           0.25,
	}
	result := ScoreWithConfig(metrics, cfg)
	// (50 + 100 + 50 + ~2) * 0.25 ≈ 50
	if result.Overall < 40 || result.Overall > 55 {
		t.Errorf("overall = %v, want between 40 and 55", result.Overall)
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
	bestMetrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 0,   // 100
		model.MetricTypeStalePRs:                0,   // 100
		model.MetricTypeBuildTime:               0,   // 100
		model.MetricTypeK8sDeployments:          0,   // 100
		model.MetricTypeContainerImages:         0,   // 100
		model.MetricTypeDeployTargets:           0,   // 100
		model.MetricTypeDependencyCount:         0,   // 100
		model.MetricTypeBuildSuccessRatio:       1.0, // 100
		model.MetricTypeDeployFrequency:         5,   // 100
		model.MetricTypeCICDComplexity:          0,   // 0 (raw, not health)
	}
	result := ScoreWithDefaults(bestMetrics)
	// With CICD=0 contributing 0, overall will be less than 100
	if result.Overall < 70 {
		t.Errorf("best case overall = %v, want >= 70", result.Overall)
	}

	// Worst values: all lower-is-better at high ref, higher-is-better at 0
	worstMetrics := map[model.MetricTypeName]float64{
		model.MetricTypeSecurityVulnerabilities: 50,   // low (weighted)
		model.MetricTypeStalePRs:                30,   // low
		model.MetricTypeBuildTime:               1800, // 0
		model.MetricTypeK8sDeployments:          200,  // low
		model.MetricTypeContainerImages:         200,  // low
		model.MetricTypeDeployTargets:           50,   // low
		model.MetricTypeDependencyCount:         500,  // low
		model.MetricTypeBuildSuccessRatio:       0,    // 0
		model.MetricTypeDeployFrequency:         0,    // 0
		model.MetricTypeCICDComplexity:          500,  // high
	}
	result = ScoreWithDefaults(worstMetrics)
	// asymptotic curve never floors, so worst case lands low but not near 0
	if result.Overall > 40 {
		t.Errorf("worst case overall = %v, want <= 40", result.Overall)
	}
}
