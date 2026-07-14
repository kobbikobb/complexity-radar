package output

import (
	"fmt"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

type Report struct {
	ProjectName        string
	ProjectDescription string
	OverallScore       float64
	Dimensions         []DimensionReport
	Metrics            []MetricReport
	CollectedAt        time.Time
	Errors             []string
}

type DimensionReport struct {
	Dimension   model.Dimension
	Score       float64
	Weight      float64
	MetricCount int
	Breakdown   string
}

type MetricReport struct {
	Name       model.MetricTypeName
	Dimension  model.Dimension
	RawValue   float64
	Normalized float64
	Unit       string
}

type OutputFormatter interface {
	Format(report Report) string
}

func ScoreGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}

func SecurityBreakdown(metrics map[model.MetricTypeName]float64) string {
	crit := int(metrics[model.MetricTypeSecurityCritical])
	high := int(metrics[model.MetricTypeSecurityHigh])
	med := int(metrics[model.MetricTypeSecurityMedium])
	low := int(metrics[model.MetricTypeSecurityLow])
	return fmt.Sprintf("%d critical, %d high, %d medium, %d low", crit, high, med, low)
}
