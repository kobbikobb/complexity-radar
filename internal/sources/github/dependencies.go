package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/sources"
)

// dependencyFiles lists files to check for dependencies, in priority order.
var dependencyFiles = []struct {
	path  string
	parse func(string) (float64, error)
}{
	{"package.json", parsePackageJSON},
	{"go.mod", parseGoMod},
	{"pom.xml", parsePomXML},
	{"requirements.txt", parseRequirementsTxt},
	{"Cargo.toml", parseCargoToml},
	{"Gemfile", parseGemfile},
}

func (s *Source) collectDependencyCount(ctx context.Context, owner, name, branch string) ([]sources.SourceMetric, error) {
	for _, df := range dependencyFiles {
		content, err := s.client.GetFileContent(ctx, owner, name, df.path, branch)
		if err != nil {
			continue // File not found, try next
		}
		count, err := df.parse(content)
		if err != nil {
			continue // Parse error, try next
		}
		return []sources.SourceMetric{
			{Type: model.MetricTypeDependencyCount, Value: count},
		}, nil
	}

	// No dependency file found
	return []sources.SourceMetric{
		{Type: model.MetricTypeDependencyCount, Value: 0},
	}, nil
}

func parsePackageJSON(content string) (float64, error) {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return 0, fmt.Errorf("parsing package.json: %w", err)
	}
	return float64(len(pkg.Dependencies) + len(pkg.DevDependencies)), nil
}

func parseGoMod(content string) (float64, error) {
	count := 0
	inRequire := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			count++
		}
		// Single-line require: require github.com/foo/bar v1.0.0
		if strings.HasPrefix(line, "require ") && !strings.HasSuffix(line, "(") {
			count++
		}
	}
	return float64(count), nil
}

func parsePomXML(content string) (float64, error) {
	count := 0
	// Simple XML parsing: count <dependency> elements
	decoder := strings.NewReader(content)
	buf := make([]byte, len(content))
	_, _ = decoder.Read(buf)

	contentStr := string(buf)
	count = strings.Count(contentStr, "<dependency>")
	return float64(count), nil
}

func parseRequirementsTxt(content string) (float64, error) {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		count++
	}
	return float64(count), nil
}

func parseCargoToml(content string) (float64, error) {
	count := 0
	inDeps := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "[dependencies]" || line == "[dev-dependencies]" {
			inDeps = true
			count++ // Count the section header as a marker
			continue
		}
		if strings.HasPrefix(line, "[") && inDeps {
			inDeps = false
		}
		if inDeps && line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
			count++
		}
	}
	return float64(count), nil
}

func parseGemfile(content string) (float64, error) {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gem ") || strings.HasPrefix(line, "gem\t") {
			count++
		}
	}
	return float64(count), nil
}
