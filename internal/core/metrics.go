// Package core defines the domain types and interfaces for ComplexityRadar.
//
// This package contains the fundamental building blocks used throughout the
// application: metrics, repositories, reports, and the interfaces that sources
// and formatters must implement.
package core

// MetricType describes a specific metric that can be collected from a repository.
type MetricType struct {
	Name      string `json:"name"`
	Dimension string `json:"dimension"`
	Unit      string `json:"unit"`
	Source    string `json:"source"`
}

// Repository represents a Git repository to analyze.
type Repository struct {
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

// Metric represents a single collected measurement for a repository.
type Metric struct {
	Type  MetricType `json:"type"`
	Value float64    `json:"value"`
}

// Report holds the complete analysis results for a set of repositories.
type Report struct {
	Projects []ProjectReport `json:"projects"`
}

// ProjectReport contains the metrics and computed scores for a single repository.
type ProjectReport struct {
	Repository Repository `json:"repository"`
	Metrics    []Metric   `json:"metrics"`
	Score      float64    `json:"score"`
}
