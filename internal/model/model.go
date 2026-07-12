package model

import "time"

type Dimension string

const (
	DimensionSecurity      Dimension = "security"
	DimensionDelivery      Dimension = "delivery"
	DimensionInfrastructure Dimension = "infrastructure"
	DimensionCode          Dimension = "code"
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
)

type MetricType struct {
	ID        int64
	Name      MetricTypeName
	Dimension Dimension
	Unit      string
}

func MetricTypes() []MetricType {
	return []MetricType{
		{Name: MetricTypeSecurityVulnerabilities, Dimension: DimensionSecurity, Unit: "count"},
		{Name: MetricTypeDeployFrequency, Dimension: DimensionDelivery, Unit: "per_week"},
		{Name: MetricTypeBuildSuccessRatio, Dimension: DimensionDelivery, Unit: "ratio"},
		{Name: MetricTypeBuildTime, Dimension: DimensionDelivery, Unit: "seconds"},
		{Name: MetricTypeStalePRs, Dimension: DimensionDelivery, Unit: "count"},
		{Name: MetricTypeK8sDeployments, Dimension: DimensionInfrastructure, Unit: "count"},
		{Name: MetricTypeContainerImages, Dimension: DimensionInfrastructure, Unit: "count"},
		{Name: MetricTypeDeployTargets, Dimension: DimensionInfrastructure, Unit: "count"},
		{Name: MetricTypeCICDComplexity, Dimension: DimensionInfrastructure, Unit: "score"},
		{Name: MetricTypeDependencyCount, Dimension: DimensionCode, Unit: "count"},
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
	ID        int64
	ProjectID int64
	URL       string
	Branch    string
	CreatedAt time.Time
	UpdatedAt time.Time
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
	ID          int64
	ProjectID   int64
	TotalScore  float64
	ComputedAt  time.Time
}

type ProjectReportScore struct {
	ID              int64
	ProjectReportID int64
	Dimension       Dimension
	Score           float64
	Weight          float64
}
