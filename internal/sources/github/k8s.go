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

func collectK8sDeployments(tree *GitTree, services int) []model.SourceMetric {
	count := 0
	for _, entry := range tree.Tree {
		if isK8sManifestDir(entry.Path) && isK8sManifestFile(entry.Path) {
			count++
		}
	}
	if services < 1 {
		services = 1
	}
	return []model.SourceMetric{
		{Type: model.MetricTypeK8sDeployments, Value: float64(count) / float64(services)},
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
