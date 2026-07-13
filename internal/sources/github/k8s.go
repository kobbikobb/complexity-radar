package github

import (
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// k8sManifestPrefixes lists path prefixes to check for Kubernetes manifests.
var k8sManifestPrefixes = []string{
	"k8s/",
	"kubernetes/",
	"deploy/",
	"manifests/",
	"charts/",
	"helm/",
}

func collectK8sDeployments(tree *GitTree) []model.SourceMetric {
	count := 0
	subdirs := map[string]bool{}
	for _, entry := range tree.Tree {
		if isK8sManifestDir(entry.Path) && isK8sManifestFile(entry.Path) {
			count++
			for _, prefix := range k8sManifestPrefixes {
				if strings.HasPrefix(entry.Path, prefix) {
					rest := strings.TrimPrefix(entry.Path, prefix)
					if idx := strings.Index(rest, "/"); idx >= 0 {
						subdirs[prefix+rest[:idx]] = true
					} else {
						subdirs[prefix] = true
					}
					break
				}
			}
		}
	}
	divisor := len(subdirs)
	if divisor == 0 {
		divisor = 1
	}
	return []model.SourceMetric{
		{Type: model.MetricTypeK8sDeployments, Value: float64(count) / float64(divisor)},
	}
}

func isK8sManifestDir(path string) bool {
	for _, prefix := range k8sManifestPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func isK8sManifestFile(path string) bool {
	extensions := []string{".yaml", ".yml", ".json"}
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
