package model

import (
	"context"
	"time"
)

// SourceMetric is a metric value collected from a source, identified by metric type name.
type SourceMetric struct {
	Type  MetricTypeName
	Value float64
}

// Source defines the interface for collecting metrics from external systems.
//
// Each implementation knows how to collect a specific set of metrics
// from a source like GitHub, Jira, or Grafana.
type Source interface {
	// Name returns the source identifier (e.g., "github").
	Name() string

	// Collect gathers all supported metrics for the given repository.
	Collect(ctx context.Context, repo Repository) ([]SourceMetric, error)

	// SupportedMetrics returns the metric types this source can collect.
	SupportedMetrics() []MetricTypeName
}

// ProjectSource collects metrics scoped to a whole project rather than a single
// repository (e.g. a feature-flag service whose flags belong to a project).
type ProjectSource interface {
	Name() string
	CollectProject(ctx context.Context, project Project) ([]SourceMetric, error)
	SupportedMetrics() []MetricTypeName
}

type Dimension string

const (
	DimensionSecurity       Dimension = "security"
	DimensionDelivery       Dimension = "delivery"
	DimensionInfrastructure Dimension = "infrastructure"
	DimensionCode           Dimension = "code"
)

type MetricTypeName string

const (
	MetricTypeSecurityVulnerabilities MetricTypeName = "security_vulnerabilities"
	MetricTypeDeployFrequency         MetricTypeName = "deploy_frequency"
	MetricTypeBuildSuccessRatio       MetricTypeName = "build_success_ratio"
	MetricTypeBuildTime               MetricTypeName = "build_time"
	MetricTypeStalePRs                MetricTypeName = "stale_prs"
	MetricTypeK8sDeployments          MetricTypeName = "k8s_deployments"
	MetricTypeContainerImages         MetricTypeName = "container_images"
	MetricTypeDeployTargets           MetricTypeName = "deploy_targets"
	MetricTypeCICDComplexity          MetricTypeName = "ci_cd_complexity"
	MetricTypeDependencyCount         MetricTypeName = "dependency_count"
	MetricTypeDependencyTotal         MetricTypeName = "dependency_total"
	MetricTypeSecurityCritical        MetricTypeName = "security_critical"
	MetricTypeSecurityHigh            MetricTypeName = "security_high"
	MetricTypeSecurityMedium          MetricTypeName = "security_medium"
	MetricTypeSecurityLow             MetricTypeName = "security_low"
	MetricTypeDecisionDensity         MetricTypeName = "decision_density"
	MetricTypeFeatureFlagDebt         MetricTypeName = "feature_flag_debt"
)

type MetricType struct {
	ID        int64
	Name      MetricTypeName
	Dimension Dimension
	Unit      string
	Weight    float64 // sub-weight within its dimension; 0 is treated as 1.0
	// Methodology, surfaced by `radar report --explain`.
	RawDef   string // what the raw value counts: numerator/denominator, exclusions
	ScoreDef string // the scoring curve, ref constant, and direction
	Source   string // where the raw value comes from
}

func MetricTypes() []MetricType {
	return []MetricType{
		{Name: MetricTypeSecurityVulnerabilities, Dimension: DimensionSecurity, Unit: "weighted/svc",
			RawDef:   "severity-weighted sum of OPEN Dependabot alerts (crit 1.0/high 0.7/med 0.3/low 0.1, dev-scope ×0.3) ÷ service count",
			ScoreDef: "asymptotic: 100*25/(value+25); lower better; any open critical gates the dimension to F",
			Source:   "Dependabot alerts API"},
		{Name: MetricTypeDeployFrequency, Dimension: DimensionDelivery, Unit: "per_week",
			RawDef:   "deploys in the last 7 days (git tags matching prefix, gitops commits, or releases); -1 = no data",
			ScoreDef: "linear: value/5*100; 5/wk→100; higher better",
			Source:   "GitHub tags/releases API (or gitops commits)"},
		{Name: MetricTypeBuildSuccessRatio, Dimension: DimensionDelivery, Unit: "ratio",
			RawDef:   "successful ÷ non-skipped, non-superseded workflow runs (last 100 on branch)",
			ScoreDef: "linear: value*100; higher better",
			Source:   "GitHub Actions API"},
		{Name: MetricTypeBuildTime, Dimension: DimensionDelivery, Unit: "seconds",
			RawDef:   "mean seconds of successful+failed workflow runs (UpdatedAt-RunStartedAt)",
			ScoreDef: "linear: 100 - value/1800*100; 30min→0; lower better",
			Source:   "GitHub Actions API"},
		{Name: MetricTypeStalePRs, Dimension: DimensionDelivery, Unit: "count",
			RawDef:   "open non-draft PRs idle ≥7 days (bots included); drafts excluded",
			ScoreDef: "asymptotic: 100*45/(value+45); lower better",
			Source:   "GitHub Pulls API"},
		{Name: MetricTypeK8sDeployments, Dimension: DimensionInfrastructure, Unit: "count/svc",
			RawDef:   "yaml/yml/json manifest files under k8s|kubernetes|deploy|manifests|charts|helm dirs ÷ service count",
			ScoreDef: "asymptotic: 100*10/(value+10); lower better",
			Source:   "Git tree"},
		{Name: MetricTypeContainerImages, Dimension: DimensionInfrastructure, Unit: "count/svc",
			RawDef:   "distinct image refs from Dockerfile FROM + k8s manifest image: lines ÷ service count",
			ScoreDef: "asymptotic: 100*3/(value+3); lower better",
			Source:   "Git tree + file contents"},
		{Name: MetricTypeDeployTargets, Dimension: DimensionInfrastructure, Unit: "count",
			RawDef:   "distinct workflow environment: names + matched appspec/buildspec/imagedefinitions files (repo-level, not per-service)",
			ScoreDef: "asymptotic: 100*20/(value+20); lower better",
			Source:   "Git tree + workflow/deploy configs"},
		{Name: MetricTypeCICDComplexity, Dimension: DimensionInfrastructure, Unit: "score",
			RawDef:   "weighted sum of automation constructs (jobs×10, uses/name×2, if×3, matrix×5, reusable×8, secrets×2, env×1) + 15/other-CI file, capped 100",
			ScoreDef: "log: logNorm(value,100)*100; higher = more CI automation present",
			Source:   "Git tree + workflow contents"},
		{Name: MetricTypeDependencyCount, Dimension: DimensionCode, Unit: "count", Weight: 0.2,
			RawDef:   "deps per service: distinct third-party deps (npm/go/maven/python/nuget/cargo/ruby) ÷ distinct services (manifest dirs); vendored dirs excluded",
			ScoreDef: "asymptotic: 100*15/(value+15); lower better; weight 0.2 within Code",
			Source:   "Git tree + manifest files"},
		{Name: MetricTypeDecisionDensity, Dimension: DimensionCode, Unit: "per100loc", Weight: 0.8,
			RawDef:   "per-service p95 of per-file decision density (decision tokens per 100 non-blank lines: (1 + if/elif/for/while/case/catch/&&/||/? tokens) ÷ non-blank lines ×100), rolled up as mean of per-service p95; size-biased sample, capped ~400 file reads; vendored/generated dirs excluded; -1=no source",
			ScoreDef: "asymptotic: 100*20/(value+20); lower better; weight 0.8 within Code",
			Source:   "Git tree + sampled file contents (regex approximation, NOT AST cyclomatic; counts tokens in comments/strings)"},
		{Name: MetricTypeFeatureFlagDebt, Dimension: DimensionCode, Unit: "count", Weight: 0.2,
			RawDef:   "count of active DevCycle feature flags flagged stale (DevCycle staleness signal, else not updated in >30d); 0 when unconfigured",
			ScoreDef: "asymptotic: 100*15/(value+15); lower better; weight 0.2 within Code",
			Source:   "DevCycle Management API /v2/projects/{key}/features"},
	}
}

// DisplayMetricTypes returns metric types used for display only (not scored).
// Severity is already encoded in the scored security_vulnerabilities weighted sum;
// scoring these counts too would double-count severity.
func DisplayMetricTypes() []MetricType {
	return []MetricType{
		{Name: MetricTypeSecurityCritical, Dimension: DimensionSecurity, Unit: "count",
			RawDef: "count of OPEN critical-severity Dependabot alerts", ScoreDef: "display-only (not scored)", Source: "Dependabot alerts API"},
		{Name: MetricTypeSecurityHigh, Dimension: DimensionSecurity, Unit: "count",
			RawDef: "count of OPEN high-severity Dependabot alerts", ScoreDef: "display-only (not scored)", Source: "Dependabot alerts API"},
		{Name: MetricTypeSecurityMedium, Dimension: DimensionSecurity, Unit: "count",
			RawDef: "count of OPEN medium-severity Dependabot alerts", ScoreDef: "display-only (not scored)", Source: "Dependabot alerts API"},
		{Name: MetricTypeSecurityLow, Dimension: DimensionSecurity, Unit: "count",
			RawDef: "count of OPEN low-severity Dependabot alerts", ScoreDef: "display-only (not scored)", Source: "Dependabot alerts API"},
		{Name: MetricTypeDependencyTotal, Dimension: DimensionCode, Unit: "count",
			RawDef: "raw distinct third-party deps summed across stacks (npm/go/maven/python/nuget/cargo/ruby); vendored dirs excluded", ScoreDef: "display-only (not scored)", Source: "Git tree + manifest files"},
	}
}

type Project struct {
	ID                   int64
	Name                 string
	Description          string
	DevCycleProjectKey   string
	DevCycleClientID     string
	DevCycleClientSecret string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Repository struct {
	ID                 int64
	ProjectID          int64
	URL                string
	Branch             string
	GitopsRepoURL      string
	DeployDetection    string
	IncludePrereleases bool
	TagPrefix          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Metric struct {
	ID           int64
	RepositoryID int64
	MetricTypeID int64
	Value        float64
	CollectedAt  time.Time
}

type ProjectMetric struct {
	ID           int64
	ProjectID    int64
	MetricTypeID int64
	Value        float64
	CollectedAt  time.Time
}

type DimensionScore struct {
	ID           int64
	RepositoryID int64
	Dimension    Dimension
	Score        float64
	Weight       float64
	ComputedAt   time.Time
}

type ProjectReport struct {
	ID         int64
	ProjectID  int64
	TotalScore float64
	ComputedAt time.Time
}

type ProjectReportScore struct {
	ID              int64
	ProjectReportID int64
	Dimension       Dimension
	Score           float64
	Weight          float64
}
