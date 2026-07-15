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
