package github

import (
	"github.com/kobbikobb/complexity-radar/internal/model"
)

// largeFileBytes is ~400 lines at 50 bytes/line, the point where a file reads as a god-file.
const largeFileBytes = 20000

func collectLargeFileRatio(tree *GitTree) []model.SourceMetric {
	var totalBlobs, largeBlobs int64
	for _, entry := range tree.Tree {
		if entry.Type != "blob" {
			continue
		}
		totalBlobs++
		if entry.Size > largeFileBytes {
			largeBlobs++
		}
	}

	if totalBlobs == 0 {
		return []model.SourceMetric{
			{Type: model.MetricTypeLargeFileRatio, Value: noDataValue},
		}
	}

	return []model.SourceMetric{
		{Type: model.MetricTypeLargeFileRatio, Value: float64(largeBlobs) / float64(totalBlobs)},
	}
}
