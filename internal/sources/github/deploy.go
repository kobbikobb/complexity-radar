package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
)

// Release represents a GitHub release.
type Release struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

const (
	noDataValue   = -1.0
	maxTagCommits = 50
)

// Tag represents a GitHub tag.
type Tag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func (s *Source) collectDeployFrequency(ctx context.Context, owner, name, gitopsRepoURL, method string, includePrereleases bool, tagPrefix string) ([]model.SourceMetric, error) {
	if gitopsRepoURL != "" {
		metrics, err := s.collectGitopsDeployFrequency(ctx, gitopsRepoURL)
		if err == nil {
			return metrics, nil
		}
		log.Printf("warning: gitops deploy frequency failed (%v), falling back to configured method", err)
	}

	var metrics []model.SourceMetric
	var err error
	switch method {
	case config.DeployDetectionTags:
		metrics, err = s.collectTagDeployFrequency(ctx, owner, name, tagPrefix)
	default:
		metrics, err = s.collectReleaseDeployFrequency(ctx, owner, name, includePrereleases, tagPrefix)
	}
	if err != nil {
		return []model.SourceMetric{
			{Type: model.MetricTypeDeployFrequency, Value: noDataValue},
		}, nil
	}
	return metrics, nil
}

func (s *Source) collectTagDeployFrequency(ctx context.Context, owner, name, tagPrefix string) ([]model.SourceMetric, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/tags", owner, name)
	data, err := s.client.GetPaginated(ctx, endpoint, map[string]string{"per_page": "100"}, 5)
	if err != nil {
		return nil, err
	}

	var tags []Tag
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, fmt.Errorf("parsing tags: %w", err)
	}

	var matched []Tag
	for _, t := range tags {
		if tagPrefix == "" || strings.HasPrefix(t.Name, tagPrefix) {
			matched = append(matched, t)
		}
	}
	if len(matched) > maxTagCommits {
		log.Printf("warning: %d tags matched, capping commit lookups at %d", len(matched), maxTagCommits)
		matched = matched[:maxTagCommits]
	}

	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	count := 0
	for _, t := range matched {
		commitData, err := s.client.Get(ctx, fmt.Sprintf("/repos/%s/%s/commits/%s", owner, name, t.Commit.SHA))
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
		date, err := time.Parse(time.RFC3339, commit.Commit.Committer.Date)
		if err != nil {
			continue
		}
		if date.After(oneWeekAgo) {
			count++
		}
	}

	return []model.SourceMetric{
		{Type: model.MetricTypeDeployFrequency, Value: float64(count)},
	}, nil
}

func (s *Source) collectGitopsDeployFrequency(ctx context.Context, gitopsRepoURL string) ([]model.SourceMetric, error) {
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

	return []model.SourceMetric{
		{Type: model.MetricTypeDeployFrequency, Value: float64(len(commits))},
	}, nil
}

func (s *Source) collectReleaseDeployFrequency(ctx context.Context, owner, name string, includePrereleases bool, tagPrefix string) ([]model.SourceMetric, error) {
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
		if r.Draft {
			continue
		}
		if r.Prerelease && !includePrereleases {
			continue
		}
		if tagPrefix != "" && !strings.HasPrefix(r.TagName, tagPrefix) {
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

	return []model.SourceMetric{
		{Type: model.MetricTypeDeployFrequency, Value: float64(weekCount)},
	}, nil
}
