package github

import (
	"github.com/kobbikobb/complexity-radar/internal/model"
)

const avgBytesPerLine = 50.0

func collectCodeLOC(languages map[string]int64) []model.SourceMetric {
	var totalBytes int64
	for _, b := range languages {
		totalBytes += b
	}
	loc := float64(totalBytes) / avgBytesPerLine
	return []model.SourceMetric{
		{Type: model.MetricTypeCodeLOC, Value: loc},
	}
}
