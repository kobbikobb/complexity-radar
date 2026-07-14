package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

const noDataValue = -1.0

func (s *Source) collectDeployFrequency(ctx context.Context, owner, name, gitopsRepoURL, tagPattern string) ([]sources.SourceMetric, error) {
	if gitopsRepoURL != "" {
		metrics, err := s.collectGitopsDeployFrequency(ctx, gitopsRepoURL)
		if err == nil {
			return metrics, nil
		}
		log.Printf("warning: gitops deploy frequency failed (%v), falling back to tags/releases", err)
	}

	if tagPattern != "" {
		metrics, err := s.collectTagDeployFrequency(ctx, owner, name, tagPattern)
		if err == nil {
			return metrics, nil
		}
		log.Printf("warning: tag deploy frequency failed (%v), falling back to releases", err)
	}

	metrics, err := s.collectReleaseDeployFrequency(ctx, owner, name)
	if err != nil {
		return []sources.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: noDataValue},
		}, nil
	}
	return metrics, nil
}

func (s *Source) collectGitopsDeployFrequency(ctx context.Context, gitopsRepoURL string) ([]sources.SourceMetric, error) {
	gitopsOwner, gitopsName, err := parseRepoURL(gitopsRepoURL)
	if err != nil {
		return nil, fmt.Errorf("parsing gitops repo URL: %w", err)
	}

	since := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	endpoint := fmt.Sprintf("/repos/%s/%s/commits", gitopsOwner, gitopsName)
	data, err := s.client.GetWithParams(ctx, endpoint, map[string]string{
		"since": since,
	})
	if err != nil {
		return nil, err
	}

	var commits []json.RawMessage
	if err := json.Unmarshal(data, &commits); err != nil {
		return nil, fmt.Errorf("parsing gitops commits: %w", err)
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeDeployFrequency, Value: float64(len(commits))},
	}, nil
}

func (s *Source) collectTagDeployFrequency(ctx context.Context, owner, name, tagPattern string) ([]sources.SourceMetric, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/tags", owner, name)
	data, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, fmt.Errorf("parsing tags: %w", err)
	}

	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	weekCount := 0

	for _, tag := range tags {
		if !strings.HasPrefix(tag.Name, tagPattern) {
			continue
		}

		commitEndpoint := fmt.Sprintf("/repos/%s/%s/commits/%s", owner, name, tag.Commit.SHA)
		commitData, err := s.client.Get(ctx, commitEndpoint)
		if err != nil {
			continue
		}

		var commit struct {
			Commit struct {
				Committer struct {
					Date string `json:"date"`
				} `json:"committer"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(commitData, &commit); err != nil {
			continue
		}

		commitDate, err := time.Parse(time.RFC3339, commit.Commit.Committer.Date)
		if err != nil {
			continue
		}

		if commitDate.After(oneWeekAgo) {
			weekCount++
		}
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeDeployFrequency, Value: float64(weekCount)},
	}, nil
}

func (s *Source) collectReleaseDeployFrequency(ctx context.Context, owner, name string) ([]sources.SourceMetric, error) {
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
