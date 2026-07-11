# ComplexityRadar — Development Roadmap

This document outlines the phased development plan for ComplexityRadar.

---

## Phase 1: Foundation (v0.1)

**Goal:** Working CLI that collects from GitHub and produces a report.

### Deliverables

| # | Deliverable | Description |
|---|-------------|-------------|
| 1.1 | Config parser | Read and validate TOML configuration files |
| 1.2 | SQLite schema | Database migrations, storage layer |
| 1.3 | GitHub source | Collect all 10 metrics using `gh` CLI |
| 1.4 | Scoring engine | Normalize metrics, calculate dimension and overall scores |
| 1.5 | Terminal output | Human-readable report with colors and tables |
| 1.6 | End-to-end flow | `radar scan` works end-to-end on a real project |

### Metrics (Phase 1)

| Metric | Source | How |
|--------|--------|-----|
| Security vulnerabilities | `gh api` | Dependabot alerts |
| Build success ratio | `gh api` | Actions workflow runs |
| Deploy frequency | `gh api` | Releases or deploy workflows |
| Build time | `gh api` | Actions workflow run duration |
| Stale PRs | `gh api` | PRs with no activity > 14 days |
| Dependency count | File parsing | package.json, go.mod, etc. |
| K8s deployments | File parsing | Manifests in repo |
| Container images | File parsing | Manifests and Dockerfiles |
| Deploy targets | File parsing | Workflow files |
| CI/CD complexity | File parsing | Number of workflow files |

### Done When

- [ ] `radar scan` runs without errors
- [ ] Config file is parsed and validated
- [ ] All 10 metrics are collected from a GitHub repo
- [ ] Scores are calculated with dimension grouping
- [ ] Terminal report is printed with clear formatting
- [ ] Data is stored in SQLite for historical tracking

---

## Phase 2: Reporting & History (v0.2)

**Goal:** Trend tracking and improved reporting.

### Deliverables

| # | Deliverable | Description |
|---|-------------|-------------|
| 2.1 | Trend report | Show how scores change over time |
| 2.2 | JSON output | Machine-readable output for CI/CD |
| 2.3 | Multi-project support | Measure and compare multiple projects |
| 2.4 | Score history | Query past scores and visualize trends |
| 2.5 | Configurable thresholds | Custom warning/critical thresholds |

### Done When

- [ ] `radar report --history` shows score trend
- [ ] `radar report --output json` produces valid JSON
- [ ] Multiple projects can be configured and measured
- [ ] Historical scores are queryable

---

## Phase 3: Extensibility (v0.3)

**Goal:** Plugin architecture for new sources.

### Deliverables

| # | Deliverable | Description |
|---|-------------|-------------|
| 3.1 | Source interface | Stable, documented interface for sources |
| 3.2 | Jira source | Collect bug/issue metrics from Jira |
| 3.3 | Custom metrics | Allow users to define custom metric types |
| 3.4 | Source documentation | Guide for writing new sources |

### Done When

- [ ] Source interface is documented and stable
- [ ] Jira source works end-to-end
- [ ] Users can add custom metrics via config
- [ ] Documentation exists for writing new sources

---

## Phase 4: Polish & Distribution (v1.0)

**Goal:** Production-ready release.

### Deliverables

| # | Deliverable | Description |
|---|-------------|-------------|
| 4.1 | HTML reports | Visual reports for sharing |
| 4.2 | Markdown output | GitHub/Confluence embedding |
| 4.3 | CI/CD integration | GitHub Actions template |
| 4.4 | Homebrew formula | Easy installation |
| 4.5 | Comprehensive docs | User guide, examples, FAQ |

### Done When

- [ ] `radar report --output html` produces shareable reports
- [ ] GitHub Actions template exists
- [ ] `brew install complexity-radar` works
- [ ] Documentation covers all use cases

---

## Milestones

| Milestone | Target | Scope |
|-----------|--------|-------|
| **v0.1** | Working prototype | Phase 1 — core collect + report |
| **v0.2** | History & trends | Phase 2 — reporting improvements |
| **v0.3** | Extensible | Phase 3 — plugin architecture |
| **v1.0** | Production ready | Phase 4 — polish & distribution |

---

## Dependencies

- `gh` CLI must be installed and authenticated
- Go 1.21+ for building from source
- SQLite (embedded via Go driver, no install needed)

---

## Out of Scope (for now)

- Web UI / dashboard
- SaaS / hosted version
- Real-time monitoring
- Team management / permissions
- Authentication beyond `gh` CLI
