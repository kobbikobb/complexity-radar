package github

import (
	"context"
	"fmt"
	"log"
	"math"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// maxComplexityFileReads bounds API cost: at most this many file contents are fetched per repo.
const maxComplexityFileReads = 400

var sourceExtensions = map[string]bool{
	".py": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true,
	".cs": true, ".go": true, ".java": true, ".rb": true, ".kt": true, ".scala": true, ".rs": true,
}

// controlFlowRe matches decision-point tokens; `else if` is covered by `if` and `??` counts once (like `||`).
// It does not strip comments/strings, so tokens there are counted too (documented limitation).
var controlFlowRe = regexp.MustCompile(`\b(?:if|elif|for|while|case|when|catch|except)\b|&&|\|\||\?\?|\?`)

// complexity approximates a file's decision-point count: 1 + control-flow tokens.
func complexity(fileText string) int {
	return 1 + len(controlFlowRe.FindAllStringIndex(fileText, -1))
}

func nonBlankLineCount(fileText string) int {
	n := 0
	for _, line := range strings.Split(fileText, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// percentile returns the q-quantile (0..1) of an ascending-sorted slice via linear interpolation.
func percentile(sorted []float64, q float64) float64 {
	switch len(sorted) {
	case 0:
		return 0
	case 1:
		return sorted[0]
	}
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func collectDecisionDensity(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree) []model.SourceMetric {
	manifestDirs := map[string]bool{}
	for _, e := range tree.Tree {
		if e.Type == "blob" {
			if _, ok := matchManifest(path.Base(e.Path)); ok {
				manifestDirs[path.Dir(e.Path)] = true
			}
		}
	}

	services := map[string][]GitTreeEntry{}
	for _, e := range tree.Tree {
		if e.Type != "blob" || isVendored(e.Path) || isGenerated(e.Path) || !sourceExtensions[strings.ToLower(path.Ext(e.Path))] {
			continue
		}
		key := serviceKey(e.Path, manifestDirs)
		services[key] = append(services[key], e)
	}

	if len(services) == 0 {
		return []model.SourceMetric{{Type: model.MetricTypeDecisionDensity, Value: noDataValue}}
	}

	keys := make([]string, 0, len(services))
	for k := range services {
		keys = append(keys, k)
		// Size-biased sample: largest files first (hotspot-biased), path ASC for determinism.
		sort.Slice(services[k], func(i, j int) bool {
			a, b := services[k][i], services[k][j]
			if a.Size != b.Size {
				return a.Size > b.Size
			}
			return a.Path < b.Path
		})
	}
	sort.Strings(keys)

	values := map[string][]float64{}
	reads := 0
	for round := 0; reads < maxComplexityFileReads; round++ {
		anyLeft := false
		for _, k := range keys {
			if round >= len(services[k]) {
				continue
			}
			anyLeft = true
			if reads >= maxComplexityFileReads {
				break
			}
			reads++
			content, err := client.GetFileContent(ctx, owner, name, services[k][round].Path, branch)
			if err != nil {
				continue
			}
			// Minified/bundled files pack everything onto one huge line; their token count isn't meaningful complexity.
			if longestLineLen(content) > 5000 {
				continue
			}
			density := float64(complexity(content)) / float64(max(1, nonBlankLineCount(content))) * 100
			values[k] = append(values[k], density)
		}
		if !anyLeft {
			break
		}
	}

	type servicePercentile struct {
		label string
		p95   float64
	}
	var services95 []servicePercentile
	for _, k := range keys {
		vals := values[k]
		if len(vals) == 0 {
			continue
		}
		sort.Float64s(vals)
		label := k
		if label == "" || label == "." {
			label = "(root)"
		}
		services95 = append(services95, servicePercentile{label, percentile(vals, 0.95)})
	}

	if len(services95) == 0 {
		return []model.SourceMetric{{Type: model.MetricTypeDecisionDensity, Value: noDataValue}}
	}

	sum := 0.0
	for _, s := range services95 {
		sum += s.p95
	}
	mean := sum / float64(len(services95))

	sort.Slice(services95, func(i, j int) bool { return services95[i].p95 > services95[j].p95 })
	top := services95
	if len(top) > 10 {
		top = top[:10]
	}
	parts := make([]string, len(top))
	for i, s := range top {
		parts[i] = fmt.Sprintf("%s=%.0f", s.label, s.p95)
	}
	log.Printf("decision density: mean=%.1f across %d services; worst: %s", mean, len(services95), strings.Join(parts, " "))

	return []model.SourceMetric{{Type: model.MetricTypeDecisionDensity, Value: mean}}
}

// isGenerated reports whether a path is a generated or build artifact whose token count isn't hand-written complexity.
func isGenerated(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		switch seg {
		case "dist", "build", "out", "bin", "obj", "generated", "__generated__", ".next", "coverage", ".turbo", "target":
			return true
		}
	}
	base := strings.ToLower(path.Base(p))
	switch {
	case strings.HasSuffix(base, ".min.js"),
		strings.HasSuffix(base, ".bundle.js"),
		strings.Contains(base, ".generated."),
		strings.HasSuffix(base, ".pb.go"),
		strings.HasSuffix(base, "_pb2.py"),
		strings.HasSuffix(base, ".g.cs"),
		strings.HasSuffix(base, ".designer.cs"):
		return true
	}
	return false
}

func longestLineLen(s string) int {
	longest := 0
	for _, line := range strings.Split(s, "\n") {
		if len(line) > longest {
			longest = len(line)
		}
	}
	return longest
}

// serviceKey returns the nearest-ancestor manifest directory, or "" for the repo-root bucket.
func serviceKey(filePath string, manifestDirs map[string]bool) string {
	for dir := path.Dir(filePath); ; dir = path.Dir(dir) {
		if manifestDirs[dir] {
			return dir
		}
		if dir == "." || dir == "/" || dir == "" {
			return ""
		}
	}
}
