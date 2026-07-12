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

func ScoreColor(score float64) string {
	switch {
	case score >= 80:
		return fmt.Sprintf("\033[32m%.1f\033[0m", score)
	case score >= 60:
		return fmt.Sprintf("\033[33m%.1f\033[0m", score)
	default:
		return fmt.Sprintf("\033[31m%.1f\033[0m", score)
	}
}

func ScoreGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}
