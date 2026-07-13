# ComplexityRadar — Domain Glossary

## Project
A software product or system being measured. A Project contains one or more Repositories and is the primary unit of reporting.

## Repository
A single git repository that belongs to a Project. Metrics are collected per Repository and aggregated into the Project's report.

## Metric
A single raw measurement value collected from a Source for a specific MetricType. Examples: "42 security vulnerabilities", "0.98 build success ratio". Metrics are stored per Repository per collection run.

## MetricType
The definition or schema of a measurable attribute — its name, the Dimension it belongs to, and its unit of measure. MetricTypes are seeded at startup and shared across all Projects.

## Dimension
A thematic grouping of related MetricTypes. ComplexityRadar defines four fixed Dimensions: Security, Delivery, Infrastructure, and Code. Dimensions are used to compute sub-scores and help users understand which area drives overall complexity.

- **Security** — vulnerabilities (e.g., Dependabot alerts)
- **Delivery** — CI/CD health (e.g., build time, deploy frequency, stale PRs)
- **Infrastructure** — platform footprint (e.g., K8s deployments, container images, CI/CD pipeline stages)
- **Code** — code-level attributes (e.g., dependency count)

## Source
An external system that provides Metric data. Each Source implementation knows how to collect a specific set of MetricTypes from a given system (e.g., GitHub, Jira, Grafana). Sources implement the `Source` interface and return `SourceMetric` values.

> **Note:** `TECHNICAL_DECISIONS.md` and `ARCHITECTURE.md` contain earlier interface sketches that are now stale (e.g., `SupportedMetrics() []MetricType` vs the actual `[]model.MetricTypeName`). This glossary reflects the current code.

## SourceMetric
A transient Metric value as returned by a Source during collection, before it is stored in the database. Distinguished from the persisted `Metric` entity by being identified by `MetricTypeName` (not a DB ID).

## Score
A normalized value in the range 0–100 representing the complexity of a Dimension or a Project, where higher scores indicate better health (lower complexity). Scores are derived from Metrics via normalization and weighting.

## DimensionScore
A Score computed for a single Dimension (e.g., Security = 72.5). Stored per Repository per collection run.

## OverallScore
The weighted average of all DimensionScores for a Project or Repository, producing a single 0–100 complexity rating. This is a conceptual term; in code the overall score lives in `ScoreResult.Overall` (scorer), `RepositoryResult.OverallScore` (collector), and `ProjectReport.TotalScore` (model).

## ProjectReport
A snapshot of a Project's overall score (`TotalScore`) and per-Dimension Scores at a specific point in time. Used for tracking complexity over time.

## ProjectReportScore
A per-Dimension score entry within a ProjectReport. Stores the Dimension, its Score, and the Weight used when computing the report's TotalScore.

## ThresholdsConfig
Configuration for alerting thresholds (e.g., maximum critical vulnerabilities, minimum build success ratio). Stored in the TOML config and used to flag repositories that exceed defined limits.

## Collection
The process of executing one or more Sources against a set of Repositories to gather Metrics, store them, and compute Scores. Each Collection produces a `CollectionResult` containing per-Repository results.
