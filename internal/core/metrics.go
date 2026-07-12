// Package core defines the domain types and interfaces for ComplexityRadar.
//
// This package contains the fundamental building blocks used throughout the
// application: metrics, repositories, reports, and the interfaces that sources
// and formatters must implement.
package core

import "fmt"

// MetricType describes a specific metric that can be collected from a repository.
type MetricType struct {
	Name      string `json:"name"`
	Dimension string `json:"dimension"`
	Unit      string `json:"unit"`
	Source    string `json:"source"`
}

// Equal checks whether two MetricType values are identical in all fields.
func (m MetricType) Equal(other MetricType) bool {
	return m.Name == other.Name &&
		m.Dimension == other.Dimension &&
		m.Unit == other.Unit &&
		m.Source == other.Source
}

// Repository represents a Git repository to analyze.
type Repository struct {
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

// Validate ensures the repository has a non-empty URL and branch.
func (r Repository) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("repository URL must not be empty")
	}
	if r.Branch == "" {
		return fmt.Errorf("repository branch must not be empty")
	}
	return nil
}

// Metric represents a single collected measurement for a repository.
type Metric struct {
	Type  MetricType `json:"type"`
	Value float64    `json:"value"`
}

// DimensionScore holds the aggregated score, weight, and name for a single
// dimension within a project report.
type DimensionScore struct {
	Dimension string  `json:"dimension"`
	Score     float64 `json:"score"`
	Weight    float64 `json:"weight"`
}

// Report holds the complete analysis results for a set of repositories.
type Report struct {
	Projects []ProjectReport `json:"projects"`
}

// ProjectByName looks up a project report by its repository name.
// Returns the report and true when found, or nil and false otherwise.
func (r Report) ProjectByName(name string) (*ProjectReport, bool) {
	for i := range r.Projects {
		if r.Projects[i].Repository.URL == name {
			return &r.Projects[i], true
		}
	}
	return nil, false
}

// ProjectReport contains the metrics and computed scores for a single repository.
type ProjectReport struct {
	Repository      Repository       `json:"repository"`
	Metrics         []Metric         `json:"metrics"`
	DimensionScores []DimensionScore `json:"dimension_scores"`
	Score           float64          `json:"score"`
}

// MetricsByDimension groups the project's metrics by their type dimension.
// Returns a map keyed by dimension name with the matching metrics as values.
func (p ProjectReport) MetricsByDimension() map[string][]Metric {
	out := make(map[string][]Metric)
	for _, m := range p.Metrics {
		d := m.Type.Dimension
		out[d] = append(out[d], m)
	}
	return out
}

// ScoreForDimension returns the score for the named dimension.
// The second return value reports whether a matching dimension was found.
func (p *ProjectReport) ScoreForDimension(dimension string) (float64, bool) {
	for _, ds := range p.DimensionScores {
		if ds.Dimension == dimension {
			return ds.Score, true
		}
	}
	return 0, false
}
