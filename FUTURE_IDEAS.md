# Future Ideas

## Metrics to Add

### Delivery
- **Lead time for changes** - Commit → deploy duration (completes DORA alongside MTTR + change failure rate)
  - Source: commit dates vs deploy/tag dates (deploy detection already exists)
- **Test coverage + flaky rate** - Coverage from CI artifacts; flaky = retry markers in workflow runs
- **Mean Time to Resolve (MTTR)** - Time from incident/bug creation to resolution
  - Source: GitHub issues with labels `incident`, `bug`, `critical`
  - Scoring: <24h = excellent, 7 days = baseline
- **Change Failure Rate** - % of deploys causing incidents
  - Requires incident tracking + deploy correlation

### Code Health
- **Real code complexity** - Cyclomatic / cognitive complexity + nesting depth, replacing large-file ratio as the scored signal
  - Requires file contents + a parser (e.g. tree-sitter for multi-language)
- **Churn × complexity hotspots** - Files that are both complex and change often (Tornhill signal) — where pain concentrates
  - Needs per-file git history; likely a git clone + `git log --numstat` walk
- **Inter-service coupling** - Dependency-graph fan-in/fan-out; architectural complexity lives in the edges
- **Dead Dependency Count** - Packages in go.mod with no imports
  - Source: `go mod graph` + import analysis
  - Scoring: 0 dead deps = perfect
- **Stale Feature Branch Count** - Branches with no activity >30 days
  - Source: GitHub branches API
  - Scoring: 0 stale = perfect
- **Code Duplication** - Percentage of duplicated code
  - Requires static analysis tool integration

### Usage/Adoption (requires external data)
- **Unused Features** - Features with no recent usage
  - Needs: APM, feature flags, or usage analytics
- **API Endpoint Usage** - Endpoints with low/no traffic
  - Needs: Grafana, Datadog, or similar
- **Feature Flag Status** - Flags always on or always off
  - Needs: Feature flag system integration

### Security
- **Dependency Vulnerability Trends** - Vulns over time
  - Track if getting better/worse
- **License Compliance** - Non-permissive licenses in deps
  - Source: `go-licenses` or similar

### Infrastructure
- **Kubernetes Resource Utilization** - CPU/memory usage vs allocated
  - Needs: Prometheus/Kubernetes metrics
- **Container Image Size** - Track bloat over time
  - Source: Docker registry API

## Improvements

### Scoring
- **Normalize per service/team** - Count metrics relative to repo size
- **Historical trending** - Compare scores over time
- **Custom thresholds** - Let users define what's "good" for their context

### Output
- **Markdown report** - For README/PR comments
- **JSON output** - For CI integration
- **Grafana dashboard** - Visualize trends
- **Slack/Teams notifications** - Alert on score degradation

### Collection
- **Multi-source support** - GitLab, Bitbucket, Jira
- **Parallel collection** - Fetch from multiple repos concurrently
- **Incremental updates** - Only fetch new data since last run
- **Cache layer** - Avoid re-fetching unchanged data

### CLI
- **radar diff** - Compare two scores
- **radar history** - Show score trends
- **radar export** - Export to various formats
- **radar compare** - Compare across repos/teams

## Experiments
- **AI-powered insights** - Use LLM to explain scores and suggest improvements
- **Predictive scoring** - Forecast future complexity based on trends
- **Team-level scoring** - Aggregate across multiple repos per team
