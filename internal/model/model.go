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
	MetricTypeSecurityCritical        MetricTypeName = "security_critical"
	MetricTypeSecurityHigh            MetricTypeName = "security_high"
	MetricTypeSecurityMedium          MetricTypeName = "security_medium"
	MetricTypeSecurityLow             MetricTypeName = "security_low"
	MetricTypeLargeFileRatio          MetricTypeName = "large_file_ratio"
)

type MetricType struct {
	ID        int64
	Name      MetricTypeName
	Dimension Dimension
	Unit      string
}

func MetricTypes() []MetricType {
	return []MetricType{
		{Name: MetricTypeSecurityVulnerabilities, Dimension: DimensionSecurity, Unit: "weighted"},
		{Name: MetricTypeDeployFrequency, Dimension: DimensionDelivery, Unit: "per_week"},
		{Name: MetricTypeBuildSuccessRatio, Dimension: DimensionDelivery, Unit: "ratio"},
		{Name: MetricTypeBuildTime, Dimension: DimensionDelivery, Unit: "seconds"},
		{Name: MetricTypeStalePRs, Dimension: DimensionDelivery, Unit: "count"},
		{Name: MetricTypeK8sDeployments, Dimension: DimensionInfrastructure, Unit: "count"},
		{Name: MetricTypeContainerImages, Dimension: DimensionInfrastructure, Unit: "count"},
		{Name: MetricTypeDeployTargets, Dimension: DimensionInfrastructure, Unit: "count"},
		{Name: MetricTypeCICDComplexity, Dimension: DimensionInfrastructure, Unit: "score"},
		{Name: MetricTypeDependencyCount, Dimension: DimensionCode, Unit: "count"},
		{Name: MetricTypeLargeFileRatio, Dimension: DimensionCode, Unit: "ratio"},
	}
}

// DisplayMetricTypes returns metric types used for display only (not scored).
// Severity is already encoded in the scored security_vulnerabilities weighted sum;
// scoring these counts too would double-count severity.
func DisplayMetricTypes() []MetricType {
	return []MetricType{
		{Name: MetricTypeSecurityCritical, Dimension: DimensionSecurity, Unit: "count"},
		{Name: MetricTypeSecurityHigh, Dimension: DimensionSecurity, Unit: "count"},
		{Name: MetricTypeSecurityMedium, Dimension: DimensionSecurity, Unit: "count"},
		{Name: MetricTypeSecurityLow, Dimension: DimensionSecurity, Unit: "count"},
	}
}

type Project struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
