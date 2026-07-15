package model

import "testing"

func TestEveryMetricTypeShouldHaveMethodology(t *testing.T) {
	// Arrange
	all := append(MetricTypes(), DisplayMetricTypes()...)

	// Act & Assert
	for _, mt := range all {
		if mt.RawDef == "" || mt.ScoreDef == "" || mt.Source == "" {
			t.Errorf("metric %q missing methodology: RawDef=%q ScoreDef=%q Source=%q", mt.Name, mt.RawDef, mt.ScoreDef, mt.Source)
		}
	}
}

func has(types []MetricType, name MetricTypeName) bool {
	for _, mt := range types {
		if mt.Name == name {
			return true
		}
	}
	return false
}

func TestDependencyTotalIsDisplayOnly(t *testing.T) {
	// Act & Assert
	if has(MetricTypes(), MetricTypeDependencyTotal) {
		t.Error("dependency_total should not be a scored metric type")
	}
	if !has(DisplayMetricTypes(), MetricTypeDependencyTotal) {
		t.Error("dependency_total should be a display metric type")
	}
}
