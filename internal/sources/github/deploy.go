package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// Release represents a GitHub release.
type Release struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

func (s *Source) collectDeployFrequency(ctx context.Context, owner, name string) ([]sources.SourceMetric, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/releases", owner, name)
	data, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, fmt.Errorf("parsing releases: %w", err)
	}

	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	weekCount := 0

	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		published, err := time.Parse(time.RFC3339, r.PublishedAt)
		if err != nil {
			continue
		}
		if published.After(oneWeekAgo) {
			weekCount++
		}
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeDeployFrequency, Value: float64(weekCount)},
	}, nil
}
