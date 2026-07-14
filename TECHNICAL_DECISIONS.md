# ComplexityRadar — Technical Decisions

This document captures the key technical decisions made during the design phase.

---

## 1. Project Name

**Decision:** ComplexityRadar

**Rationale:** Combines the core value proposition (complexity measurement) with the radar metaphor (scanning, mapping, detection). Clear and descriptive.

**Alternatives considered:**
- Tech Radar — conflicts with ThoughtWorks Tech Radar
- RadarQ — catchy but vague
- ProjectVitals — good but less technical

---

## 2. Tech Stack

**Decision:** Go (Golang)

**Rationale:**
- Compiles to single binary — easy distribution
- Excellent GitHub API libraries (`google/go-github`, `shurcooL/githubv4`)
- Strong YAML/TOML parsing ecosystem
- SQLite drivers available (`mattn/go-sqlite3`, `modernc.org/sqlite`)
- Fast execution — important for CI/CD integration
- Strong standard library for HTTP, JSON, CLI

**Alternatives considered:**
- Rust — faster but steeper learning curve, slower development
- Python — faster to develop but not single binary, runtime dependency
- TypeScript/Node — not single binary, heavier runtime

---

## 3. Database

**Decision:** SQLite

**Rationale:**
- Zero configuration — file-based, no server needed
- Portable — single file can be moved/backed up
- Supports historical queries for trend tracking
- Standard for CLI tools (Homebrew, npm, etc.)
- Good Go driver support

**Alternatives considered:**
- PostgreSQL — overkill for CLI tool, requires server
- JSON files — no query capability, hard for trends

---

## 4. CLI Structure

**Decision:** Three primary commands

| Command | Purpose |
|---------|---------|
| `radar collect` | Pull data from sources, store in SQLite |
| `radar report` | Calculate scores, output results |
| `radar scan` | Shorthand — runs collect + report |

**Rationale:**
- Separation of concerns (data gathering vs. presentation)
- Users can collect frequently (CI) and report on demand
- `scan` provides convenience for quick checks

---

## 5. Config Format

**Decision:** TOML

**Example:**
```toml
[project]
name = "My App"
description = "Main product"

[[repositories]]
url = "github.com/org/repo"
branch = "main"

[[repositories]]
url = "github.com/org/another-repo"

[weights]
security = 0.25
delivery = 0.30
infrastructure = 0.25
code = 0.20

[thresholds]
security_vulnerabilities_critical = 0
build_success_ratio_min = 0.95
```

**Rationale:**
- Human-readable, supports comments
- Standard for Go CLI tools
- Less error-prone than YAML (no indentation issues)

---

## 6. Extensibility Model

**Decision:** Interface-based plugins

**Rationale:**
- Clean Go idiom — define `Source` interface, implement per source
- Testable — mock sources for unit tests
- Easy to add new sources (Jira, Grafana, etc.) without changing core
- No runtime plugin loading complexity

**Interface sketch:**
```go
type Source interface {
    Name() string
    Collect(ctx context.Context, repo Repository) ([]Metric, error)
    SupportedMetrics() []MetricType
}
```

---

## 7. Scoring Model

**Decision:** Dimension-based scoring with weights

**Dimensions (groups of metrics):**
| Dimension | Metrics |
|-----------|---------|
| **Security** | Security vulnerabilities |
| **Delivery** | Deploy frequency, build success ratio, build time, stale PRs |
| **Infrastructure** | K8s deployments, container images, deploy targets, CI/CD complexity |
| **Code** | Dependencies, large-file ratio |

**Scoring approach:**
1. Normalize each metric to 0-100 scale
2. Calculate dimension score = weighted average of metrics in dimension
3. Calculate overall score = weighted average of dimension scores
4. Floor the overall grade: it can't beat the worst critical-dimension grade (security) by more than one letter, so strong delivery can't mask an insecure codebase

**Rationale:**
- Users can see which dimension is dragging score down
- Weights are configurable per project
- Transparent — users understand why score is what it is
- The grade floor keeps a weak critical dimension visible instead of averaged away

---

## 8. Output Format

**Decision:** Terminal output for v1, interface-based for future formats

**Rationale:**
- Terminal is primary use case (CLI tool)
- Interface allows adding JSON, HTML, Markdown later without refactoring
- JSON is high priority for CI/CD integration

**Future formats:**
- JSON — machine-readable, CI/CD
- HTML — visual reports, team sharing
- Markdown — GitHub/Confluence embedding

---

## 9. Repository Structure

**Decision:** Mono-repo with standard Go layout

```
complexity-radar/
├── cmd/radar/              # CLI entrypoint
├── internal/
│   ├── collector/          # Data collection orchestration
│   ├── scorer/             # Scoring engine
│   ├── sources/            # Source implementations
│   │   └── github/         # GitHub source
│   ├── store/              # SQLite storage layer
│   ├── model/              # Domain types
│   └── output/             # Report formatters
│       └── terminal/       # Terminal output
├── pkg/                    # Public API (if needed)
├── configs/                # Default config templates
├── docs/                   # Documentation
├── migrations/             # SQLite schema migrations
└── .complexity-radar.toml  # Example config
```

**Rationale:**
- Simple for a CLI tool — not a platform
- `internal/` enforces package boundaries
- Easy for contributors to navigate

---

## 10. Data Model Decisions

**Decision:** Dimension is a field on MetricType (not a separate entity)

**Rationale:**
- Simpler schema — one less table
- Dimension is a grouping label, not a first-class entity
- Easy to filter/group by dimension in queries

**MetricType example:**
```go
type MetricType struct {
    Name      string  // "security_vulnerabilities"
    Dimension string  // "security"
    Unit      string  // "count"
    Source    string  // "github"
}
```

---

## Decision Log

| Date | Decision | Status |
|------|----------|--------|
| 2026-07-11 | Project name: ComplexityRadar | Final |
| 2026-07-11 | Tech stack: Go | Final |
| 2026-07-11 | Database: SQLite | Final |
| 2026-07-11 | CLI: collect/report/scan | Final |
| 2026-07-11 | Config: TOML | Final |
| 2026-07-11 | Extensibility: Interface-based | Final |
| 2026-07-11 | Scoring: Dimension-based | Final |
| 2026-07-11 | Output: Terminal v1 | Final |
| 2026-07-11 | Repo: Mono-repo | Final |
| 2026-07-11 | Data model: Dimension as field | Final |
