package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// k8sManifestPaths lists paths to check for Kubernetes manifests.
var k8sManifestPaths = []string{
	"k8s/",
	"kubernetes/",
	"deploy/",
	"manifests/",
	"charts/",
	"helm/",
}

func (s *Source) collectK8sDeployments(ctx context.Context, owner, name, branch string) ([]sources.SourceMetric, error) {
	count := 0

	for _, prefix := range k8sManifestPaths {
		// Try to list files in the directory using the git tree API
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
				count++
			}
		}
		break // Only check first matching prefix
	}

	return []sources.SourceMetric{
		{Type: model.MetricTypeK8sDeployments, Value: float64(count)},
	}, nil
}

func isK8sManifest(path string) bool {
	extensions := []string{".yaml", ".yml", ".json"}
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
