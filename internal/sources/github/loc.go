package github

import (
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

const avgBytesPerLine = 50.0

func collectCodeLOC(languages map[string]int64) []sources.SourceMetric {
	var totalBytes int64
	for _, b := range languages {
		totalBytes += b
	}
	loc := float64(totalBytes) / avgBytesPerLine
	return []sources.SourceMetric{
		{Type: model.MetricTypeCodeLOC, Value: loc},
	}
}
