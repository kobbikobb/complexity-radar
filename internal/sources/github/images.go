package github

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// imageRefPattern matches container image references in Dockerfiles and manifests.
var imageRefPattern = regexp.MustCompile(`image:\s*["']?([a-zA-Z0-9._/-]+:[a-zA-Z0-9._-]+)["']?`)

// dockerfileFromPattern matches FROM statements in Dockerfiles.
var dockerfileFromPattern = regexp.MustCompile(`^FROM\s+([a-zA-Z0-9._/-]+:[a-zA-Z0-9._-]+)`)

func (s *Source) collectContainerImages(ctx context.Context, owner, name, branch string) ([]sources.SourceMetric, error) {
	images := make(map[string]bool)

	// Check Dockerfile
	s.collectImagesFromDockerfile(ctx, owner, name, branch, images)

	// Check K8s manifests
	s.collectImagesFromManifests(ctx, owner, name, branch, images)

	return []sources.SourceMetric{
		{Type: model.MetricTypeContainerImages, Value: float64(len(images))},
	}, nil
}

func (s *Source) collectImagesFromDockerfile(ctx context.Context, owner, name, branch string, images map[string]bool) {
	content, err := s.client.GetFileContent(ctx, owner, name, "Dockerfile", branch)
	if err != nil {
		return
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		matches := dockerfileFromPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			images[matches[1]] = true
		}
	}
}

func (s *Source) collectImagesFromManifests(ctx context.Context, owner, name, branch string, images map[string]bool) {
	// Check common manifest paths
	manifestPaths := []string{
		"k8s/", "kubernetes/", "deploy/", "manifests/", "charts/", "helm/",
	}

	for _, prefix := range manifestPaths {
		endpoint := fmt.Sprintf("/repos/%s/%s/git/trees/%s", owner, name, branch)
		data, err := s.client.GetWithParams(ctx, endpoint, map[string]string{"recursive": "1"})
		if err != nil {
			continue
		}

		var tree struct {
			Tree []struct {
				Path string `json:"path"`
			} `json:"tree"`
		}
		if err := json.Unmarshal(data, &tree); err != nil {
			continue
		}

		for _, item := range tree.Tree {
			if strings.HasPrefix(item.Path, prefix) && isK8sManifest(item.Path) {
				content, err := s.client.GetFileContent(ctx, owner, name, item.Path, branch)
				if err != nil {
					continue
				}
				for _, match := range imageRefPattern.FindAllStringSubmatch(content, -1) {
					if len(match) > 1 {
						images[match[1]] = true
					}
				}
			}
		}
		break
	}
}
