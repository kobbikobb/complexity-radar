package github

import (
	"github.com/kobbikobb/complexity-radar/internal/model"
)

func collectCodeComplexity(tree *GitTree, languages map[string]int64) []model.SourceMetric {
	var totalBytes int64
	for _, b := range languages {
		totalBytes += b
	}

	var fileCount int64
	for _, entry := range tree.Tree {
		if entry.Type == "blob" {
			fileCount++
		}
	}

	if fileCount == 0 {
		return []model.SourceMetric{
			{Type: model.MetricTypeCodeComplexity, Value: 0},
		}
	}

	avgSize := float64(totalBytes) / float64(fileCount)
	return []model.SourceMetric{
		{Type: model.MetricTypeCodeComplexity, Value: avgSize},
	}
}
