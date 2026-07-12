# ComplexityRadar — Development Roadmap

Each item below is a separate PR. Work proceeds top-to-bottom.

---

## PR #1: Project Infrastructure

**Goal:** Build pipeline, test pipeline, release pipeline.

| # | Task | Description |
|---|------|-------------|
| 1.1 | Go module setup | go.mod, go.sum, linter config (.golangci.yml) |
| 1.2 | Makefile | `make build`, `make test`, `make lint`, `make run` |
| 1.3 | CI pipeline | GitHub Actions: lint, test, build on PR |
| 1.4 | Release pipeline | GitHub Actions: goreleaser on tag |
| 1.5 | Contributing guide | CONTRIBUTING.md with dev setup instructions |

### Done When

- [ ] `make build` produces binary
- [ ] `make test` runs test suite
- [ ] `make lint` passes
- [ ] CI runs on PRs
- [ ] Tagged releases produce binaries

---

## PR #2: Config Parser

**Goal:** Read and validate TOML configuration.

| # | Task | Description |
|---|------|-------------|
| 2.1 | Config struct | Define Project, Repository, Weights types |
| 2.2 | TOML parsing | Load config file using `pelletier/go-toml` |
| 2.3 | Validation | Required fields, weight sums, URL format |
| 2.4 | Default values | Sensible defaults when not specified |
| 2.5 | Unit tests | Parse valid/invalid configs |

### Done When

- [ ] Config file is parsed into Go structs
- [ ] Validation errors are clear and actionable
- [ ] Defaults are applied correctly
- [ ] Tests pass

---

## PR #3: Domain Model & Storage

**Goal:** SQLite schema, domain types, storage layer.

| # | Task | Description |
|---|------|-------------|
| 3.1 | Domain types | Project, Repository, Metric, MetricType, Score |
| 3.2 | SQLite schema | Tables for projects, repositories, metrics, scores |
| 3.3 | Migrations | Schema versioning with migration files |
| 3.4 | Storage layer | CRUD operations for all entities |
| 3.5 | Unit tests | Store and retrieve all entity types |

### Done When

- [ ] Database is created and migrated on first run
- [ ] Can store and retrieve projects, repos, metrics, scores
- [ ] Tests pass with in-memory SQLite

---

## PR #4: GitHub Source — API Metrics

**Goal:** Collect metrics from GitHub API using `gh` CLI.

| # | Task | Description |
|---|------|-------------|
| 4.1 | Source interface | Define `Source` interface in Go |
| 4.2 | GitHub client | Shell out to `gh api` for API calls |
| 4.3 | Security vulns | Fetch Dependabot alerts |
| 4.4 | Build success ratio | Fetch Actions workflow runs, calculate success rate |
| 4.5 | Deploy frequency | Fetch releases or deploy workflow runs |
| 4.6 | Build time | Fetch Actions workflow run durations |
| 4.7 | Stale PRs | Fetch PRs, filter by last activity date |
| 4.8 | Unit tests | Mock `gh` responses for each metric |

### Done When

- [ ] Source interface is defined and documented
- [ ] All 5 API metrics are collected
- [ ] Tests pass with mocked `gh` output
- [ ] Error handling is robust (rate limits, auth, missing data)

---

## PR #5: GitHub Source — File Parsing

**Goal:** Collect metrics from repository file contents.

| # | Task | Description |
|---|------|-------------|
| 5.1 | File fetcher | Use `gh api` to read file contents |
| 5.2 | Dependency count | Parse package.json, go.mod, pom.xml, etc. |
| 5.3 | K8s deployments | Parse YAML manifests for Deployment resources |
| 5.4 | Container images | Parse manifests and Dockerfiles for image refs |
| 5.5 | Deploy targets | Parse workflow files for deploy environments |
| 5.6 | CI/CD complexity | Count workflow files and steps |
| 5.7 | Unit tests | Mock file contents for each parser |

### Done When

- [ ] All 5 file-based metrics are collected
- [ ] Parsers handle missing files gracefully
- [ ] Tests pass with mocked file contents

---

## PR #6: Scoring Engine

**Goal:** Calculate dimension and overall scores.

| # | Task | Description |
|---|------|-------------|
| 6.1 | Normalization | Map raw metrics to 0-100 scale |
| 6.2 | Dimension scoring | Group metrics by dimension, calculate weighted average |
| 6.3 | Overall scoring | Aggregate dimension scores with weights |
| 6.4 | Configurable weights | Load weights from config file |
| 6.5 | Unit tests | Score calculation with known inputs/outputs |

### Done When

- [ ] Metrics are normalized correctly
- [ ] Dimension scores are calculated
- [ ] Overall score is calculated
- [ ] Weights are configurable
- [ ] Tests pass

---

## PR #7: Terminal Output

**Goal:** Human-readable terminal report.

| # | Task | Description |
|---|------|-------------|
| 7.1 | Output interface | Define `OutputFormatter` interface |
| 7.2 | Terminal formatter | Colored output with tables |
| 7.3 | Score display | Overall score with dimension breakdown |
| 7.4 | Metric details | Show raw values and normalized scores |
| 7.5 | Error display | Clear error messages for missing data |

### Done When

- [ ] `radar report` prints formatted output
- [ ] Colors and tables render correctly
- [ ] Scores are easy to understand
- [ ] Missing data is handled gracefully

---

## PR #8: End-to-End Integration

**Goal:** `radar scan` works on a real project.

| # | Task | Description |
|---|------|-------------|
| 8.1 | Wire up components | Connect config, source, store, scorer, output |
| 8.2 | `radar collect` | Full collection flow |
| 8.3 | `radar report` | Full reporting flow |
| 8.4 | `radar scan` | Collect + report in one command |
| 8.5 | Integration test | Run on a real GitHub repo |

### Done When

- [ ] `radar scan` runs end-to-end without errors
- [ ] Config is loaded correctly
- [ ] Metrics are collected and stored
- [ ] Scores are calculated and displayed
- [ ] Integration test passes

---

## PR #9: Trend Tracking

**Goal:** Show how scores change over time.

| # | Task | Description |
|---|------|-------------|
| 9.1 | Collection history | Store multiple collections with timestamps |
| 9.2 | Trend query | Fetch score history for a project |
| 9.3 | Trend display | Show score changes in terminal |
| 9.4 | `--history` flag | Add history flag to report command |

### Done When

- [ ] Multiple collections are stored with timestamps
- [ ] `radar report --history` shows trend
- [ ] Trend is displayed clearly in terminal

---

## PR #10: JSON Output

**Goal:** Machine-readable output for CI/CD.

| # | Task | Description |
|---|------|-------------|
| 10.1 | JSON formatter | Implement `OutputFormatter` for JSON |
| 10.2 | `--output json` flag | Add output flag to report command |
| 10.3 | Schema definition | Document JSON output schema |

### Done When

- [ ] `radar report --output json` produces valid JSON
- [ ] JSON includes all scores and metrics
- [ ] Schema is documented

---

## PR #11: Multi-Project Support

**Goal:** Measure and compare multiple projects.

| # | Task | Description |
|---|------|-------------|
| 11.1 | Project filtering | `--project` flag to filter by project |
| 11.2 | Comparison view | Show scores for multiple projects side by side |
| 11.3 | Config changes | Support multiple projects in config |

### Done When

- [ ] Multiple projects can be configured
- [ ] `radar report --project "My App"` filters correctly
- [ ] Comparison view works

---

## PR #12: Configurable Thresholds

**Goal:** Custom warning/critical thresholds.

| # | Task | Description |
|---|------|-------------|
| 12.1 | Threshold config | Add thresholds section to config |
| 12.2 | Threshold evaluation | Check metrics against thresholds |
| 12.3 | Warning display | Show warnings in terminal output |
| 12.4 | Exit codes | Non-zero exit when thresholds exceeded |

### Done When

- [ ] Thresholds are configurable per metric
- [ ] Warnings are displayed when exceeded
- [ ] Exit code reflects threshold status

---

## Future Ideas (Out of Scope)

- HTML reports with visual charts
- Markdown output for GitHub/Confluence
- Homebrew formula for easy installation
- Jira source integration
- Grafana source integration
- Custom metric definitions
- Web UI / dashboard
- SaaS / hosted version
- Real-time monitoring
- Team management / permissions

---

## Dependencies

- `gh` CLI must be installed and authenticated
- Go 1.25+ for building from source
- SQLite (embedded via Go driver, no install needed)
