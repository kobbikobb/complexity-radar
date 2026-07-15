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

func collectCyclomaticP95(ctx context.Context, client APIClient, owner, name, branch string, tree *GitTree) []model.SourceMetric {
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
		if e.Type != "blob" || isVendored(e.Path) || !sourceExtensions[strings.ToLower(path.Ext(e.Path))] {
			continue
		}
		key := serviceKey(e.Path, manifestDirs)
		services[key] = append(services[key], e)
	}

	if len(services) == 0 {
		return []model.SourceMetric{{Type: model.MetricTypeCyclomaticP95, Value: noDataValue}}
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
			values[k] = append(values[k], float64(complexity(content)))
		}
		if !anyLeft {
			break
		}
	}

	var p95s []float64
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := values[k]
		if len(vals) == 0 {
			continue
		}
		sort.Float64s(vals)
		p := percentile(vals, 0.95)
		p95s = append(p95s, p)
		label := k
		if label == "" || label == "." {
			label = "(root)"
		}
		parts = append(parts, fmt.Sprintf("%s=%.0f", label, p))
	}

	if len(p95s) == 0 {
		return []model.SourceMetric{{Type: model.MetricTypeCyclomaticP95, Value: noDataValue}}
	}

	sum := 0.0
	for _, p := range p95s {
		sum += p
	}
	mean := sum / float64(len(p95s))
	log.Printf("cyclomatic p95 by service: %s (mean=%.1f)", strings.Join(parts, " "), mean)

	return []model.SourceMetric{{Type: model.MetricTypeCyclomaticP95, Value: mean}}
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
