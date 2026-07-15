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
		// Log scale: 0 → 100, ref → ~0, half-ref → ~50
		{"zero security_vulns", model.MetricTypeSecurityVulnerabilities, 0, 100, 100},
		{"low security_vulns", model.MetricTypeSecurityVulnerabilities, 5, 50, 90},
		{"high security_vulns", model.MetricTypeSecurityVulnerabilities, 30, 0, 30},

		{"zero stale_prs", model.MetricTypeStalePRs, 0, 100, 100},
		{"low stale_prs", model.MetricTypeStalePRs, 5, 40, 80},
		{"high stale_prs", model.MetricTypeStalePRs, 20, 0, 30},

		{"zero build_time", model.MetricTypeBuildTime, 0, 100, 100},
		{"half refMax build_time", model.MetricTypeBuildTime, 900, 45, 55},
		{"at refMax build_time", model.MetricTypeBuildTime, 1800, 0, 5},

		{"zero k8s_deployments", model.MetricTypeK8sDeployments, 0, 100, 100},
		{"low k8s_deployments", model.MetricTypeK8sDeployments, 10, 50, 80},
		{"high k8s_deployments", model.MetricTypeK8sDeployments, 100, 0, 40},

		{"zero container_images", model.MetricTypeContainerImages, 0, 100, 100},
		{"low container_images", model.MetricTypeContainerImages, 10, 50, 80},

		{"zero deploy_targets", model.MetricTypeDeployTargets, 0, 100, 100},
		{"low deploy_targets", model.MetricTypeDeployTargets, 5, 50, 90},

		{"zero dependency_count", model.MetricTypeDependencyCount, 0, 100, 100},
		{"low dependency_count", model.MetricTypeDependencyCount, 50, 30, 70},
		{"high dependency_count", model.MetricTypeDependencyCount, 300, 0, 30},
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

func TestNormalizeMetricCyclomaticP95(t *testing.T) {
	low := NormalizeMetric(model.MetricTypeCyclomaticP95, 10)
	high := NormalizeMetric(model.MetricTypeCyclomaticP95, 200)

	if low <= high {
		t.Errorf("expected higher complexity to score lower: low(10)=%v high(200)=%v", low, high)
	}
	if got := NormalizeMetric(model.MetricTypeCyclomaticP95, 0); !approxEqual(got, 100) {
		t.Errorf("value 0 = %v, want 100", got)
	}
	if got := NormalizeMetric(model.MetricTypeCyclomaticP95, -1.0); !math.IsNaN(got) {
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
		model.MetricTypeCyclomaticP95:   0, // 100
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
		t.Errorf("code metric count = %d, want 2 (dependency_count + cyclomatic_p95)", code.MetricCount)
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
		model.MetricTypeK8sDeployments:          50,  // infra: ~26 (log scale)
		model.MetricTypeDependencyCount:         200, // code: ~15 (log scale)
	}
	weights := map[model.Dimension]float64{
		model.DimensionSecurity:       0.25,
		model.DimensionDelivery:       0.25,
		model.DimensionInfrastructure: 0.25,
		model.DimensionCode:           0.25,
	}
	result := Score(metrics, weights)
	// security=100*0.25, delivery=0, infra=~26*0.25, code=~15*0.25
	// overall should be > 25 but < 50
	if result.Overall < 20 || result.Overall > 50 {
		t.Errorf("overall = %v, want between 20 and 50", result.Overall)
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
	// With log scale, scores will differ from linear
	// security=100, delivery=50, infra=~63, code=~70
	// overall should be reasonable
	if result.Overall < 50 || result.Overall > 80 {
		t.Errorf("overall = %v, want between 50 and 80", result.Overall)
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
		model.MetricTypeSecurityVulnerabilities: 50,  // security: low (weighted vulns)
		model.MetricTypeBuildSuccessRatio:       1.0, // delivery: 100
		model.MetricTypeK8sDeployments:          200, // infra: low (log scale)
		model.MetricTypeDependencyCount:         500, // code: low (log scale)
	}
	cfg := config.WeightsConfig{
		Security:       0.25,
		Delivery:       0.25,
		Infrastructure: 0.25,
		Code:           0.25,
	}
	result := ScoreWithConfig(metrics, cfg)
	// delivery contributes: 100 * 0.25 = 25, others contribute some via log scale
	if result.Overall < 20 || result.Overall > 40 {
		t.Errorf("overall = %v, want between 20 and 40", result.Overall)
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
	if result.Overall > 20 {
		t.Errorf("worst case overall = %v, want <= 20", result.Overall)
	}
}
