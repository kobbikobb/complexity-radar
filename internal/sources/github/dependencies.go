package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// manifestParser matches dependency manifests by file name and extracts
// the bare dependency names for a given stack.
type manifestParser struct {
	stack string
	match func(fileName string) bool
	parse func(content string) []string
}

func exact(name string) func(string) bool {
	return func(f string) bool { return f == name }
}

var dependencyManifests = []manifestParser{
	{"npm", exact("package.json"), parsePackageJSON},
	{"go", exact("go.mod"), parseGoMod},
	{"maven", exact("pom.xml"), parsePomXML},
	{"python", func(f string) bool { return strings.HasPrefix(f, "requirements") && strings.HasSuffix(f, ".txt") }, parseRequirementsTxt},
	{"python", exact("pyproject.toml"), parsePyprojectToml},
	{"nuget", func(f string) bool { return strings.HasSuffix(f, ".csproj") }, parseCsproj},
	{"cargo", exact("Cargo.toml"), parseCargoToml},
	{"ruby", exact("Gemfile"), parseGemfile},
}

func collectDependencyCount(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree) ([]model.SourceMetric, error) {
	sets := map[string]map[string]struct{}{}
	for _, entry := range tree.Tree {
		if isVendored(entry.Path) {
			continue
		}
		fileName := entry.Path
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 {
			fileName = fileName[idx+1:]
		}
		mp, ok := matchManifest(fileName)
		if !ok {
			continue
		}
		content, err := client.GetFileContent(ctx, owner, name, entry.Path, branch)
		if err != nil {
			continue
		}
		set := sets[mp.stack]
		if set == nil {
			set = map[string]struct{}{}
			sets[mp.stack] = set
		}
		for _, dep := range mp.parse(content) {
			if dep != "" {
				set[dep] = struct{}{}
			}
		}
	}

	total := 0
	stacks := make([]string, 0, len(sets))
	for stack := range sets {
		stacks = append(stacks, stack)
		total += len(sets[stack])
	}
	sort.Strings(stacks)
	parts := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		parts = append(parts, fmt.Sprintf("%s=%d", stack, len(sets[stack])))
	}
	if total > 0 {
		log.Printf("distinct dependencies by stack: %s (total=%d)", strings.Join(parts, " "), total)
	}

	return []model.SourceMetric{
		{Type: model.MetricTypeDependencyCount, Value: float64(total)},
	}, nil
}

func matchManifest(fileName string) (manifestParser, bool) {
	for _, mp := range dependencyManifests {
		if mp.match(fileName) {
			return mp, true
		}
	}
	return manifestParser{}, false
}

func isVendored(path string) bool {
	for _, seg := range strings.Split(path, "/") {
		switch seg {
		case "node_modules", "vendor", ".venv":
			return true
		}
	}
	return false
}

func parsePackageJSON(content string) []string {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil
	}
	names := make([]string, 0, len(pkg.Dependencies)+len(pkg.DevDependencies))
	for n := range pkg.Dependencies {
		names = append(names, n)
	}
	for n := range pkg.DevDependencies {
		names = append(names, n)
	}
	return names
}

func parseGoMod(content string) []string {
	var names []string
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
			names = append(names, strings.Fields(line)[0])
			continue
		}
		// Single-line require: require github.com/foo/bar v1.0.0
		if !inRequire && strings.HasPrefix(line, "require ") && !strings.HasSuffix(line, "(") {
			if fields := strings.Fields(line); len(fields) >= 2 {
				names = append(names, fields[1])
			}
		}
	}
	return names
}

var (
	pomDependencyRe = regexp.MustCompile(`(?s)<dependency>(.*?)</dependency>`)
	pomGroupRe      = regexp.MustCompile(`<groupId>(.*?)</groupId>`)
	pomArtifactRe   = regexp.MustCompile(`<artifactId>(.*?)</artifactId>`)
)

func parsePomXML(content string) []string {
	var names []string
	for _, block := range pomDependencyRe.FindAllStringSubmatch(content, -1) {
		var name string
		if g := pomGroupRe.FindStringSubmatch(block[1]); len(g) > 1 {
			name = strings.TrimSpace(g[1])
		}
		if a := pomArtifactRe.FindStringSubmatch(block[1]); len(a) > 1 {
			if name != "" {
				name += ":"
			}
			name += strings.TrimSpace(a[1])
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func parseRequirementsTxt(content string) []string {
	var names []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if name := pyPackageName(line); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func parsePyprojectToml(content string) []string {
	var names []string
	section := ""
	inProjectDeps := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			section = line
			inProjectDeps = false
			continue
		}
		switch section {
		case "[project]":
			if strings.HasPrefix(line, "dependencies") && strings.Contains(line, "[") {
				inProjectDeps = true
				line = line[strings.Index(line, "[")+1:]
			}
			if inProjectDeps {
				if idx := strings.Index(line, "]"); idx >= 0 {
					line = line[:idx]
					inProjectDeps = false
				}
				for _, item := range strings.Split(line, ",") {
					item = strings.Trim(strings.TrimSpace(item), `"'`)
					if name := pyPackageName(item); name != "" {
						names = append(names, name)
					}
				}
			}
		case "[tool.poetry.dependencies]", "[tool.poetry.dev-dependencies]":
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key := strings.ToLower(strings.Trim(strings.TrimSpace(strings.SplitN(line, "=", 2)[0]), `"'`))
			if key != "" && key != "python" {
				names = append(names, key)
			}
		}
	}
	return names
}

// pyPackageName strips version specifiers, extras and markers to a bare name.
func pyPackageName(spec string) string {
	name := strings.FieldsFunc(spec, func(r rune) bool {
		return strings.ContainsRune(" <>=!~;[(", r)
	})
	if len(name) == 0 {
		return ""
	}
	return strings.ToLower(name[0])
}

var csprojPackageRe = regexp.MustCompile(`<PackageReference\s+[^>]*Include\s*=\s*"([^"]+)"`)

func parseCsproj(content string) []string {
	var names []string
	for _, m := range csprojPackageRe.FindAllStringSubmatch(content, -1) {
		names = append(names, m[1])
	}
	return names
}

func parseCargoToml(content string) []string {
	var names []string
	inDeps := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "[dependencies]" || line == "[dev-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") && inDeps {
			inDeps = false
		}
		if inDeps && line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
			names = append(names, strings.TrimSpace(strings.SplitN(line, "=", 2)[0]))
		}
	}
	return names
}

func parseGemfile(content string) []string {
	var names []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gem ") || strings.HasPrefix(line, "gem\t") {
			name := strings.Trim(strings.SplitN(strings.TrimSpace(line[3:]), ",", 2)[0], ` '"`)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}
