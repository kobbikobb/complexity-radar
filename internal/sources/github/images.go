package github

import (
	"context"
	"regexp"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// imageRefPattern matches container image references in Dockerfiles and manifests.
// Handles both tagged (nginx:1.21) and untagged (nginx) images.
var imageRefPattern = regexp.MustCompile(`image:\s*["']?([a-zA-Z0-9._/-]+(?::[a-zA-Z0-9._-]+)?)["']?`)

// dockerfileFromPattern matches FROM statements in Dockerfiles.
// Handles both tagged (nginx:1.21) and untagged (scratch) images.
var dockerfileFromPattern = regexp.MustCompile(`^FROM\s+([a-zA-Z0-9._/-]+(?::[a-zA-Z0-9._-]+)?)`)

func collectContainerImages(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree, services int) []model.SourceMetric {
	images := make(map[string]bool)

	// Check Dockerfile
	collectImagesFromDockerfile(ctx, client, owner, name, branch, images)

	// Check K8s manifests from the pre-fetched tree
	collectImagesFromManifests(ctx, client, owner, name, branch, tree, images)

	if services < 1 {
		services = 1
	}
	return []model.SourceMetric{
		{Type: model.MetricTypeContainerImages, Value: float64(len(images)) / float64(services)},
	}
}

func collectImagesFromDockerfile(ctx context.Context, client APIClient, owner, name, branch string, images map[string]bool) {
	content, err := client.GetFileContent(ctx, owner, name, "Dockerfile", branch)
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

func collectImagesFromManifests(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree, images map[string]bool) {
	for _, entry := range tree.Tree {
		if isK8sManifestDir(entry.Path) && isK8sManifestFile(entry.Path) {
			content, err := client.GetFileContent(ctx, owner, name, entry.Path, branch)
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
}
