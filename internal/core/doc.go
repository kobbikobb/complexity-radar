// Package core defines the central domain types and interfaces for
// ComplexityRadar. It serves as the shared vocabulary between all internal
// packages (collector, scorer, sources, output) and ensures that each layer
// depends only on abstractions rather than concrete implementations.
//
// The key concepts are:
//
//   - MetricType – describes what a measurement represents (e.g. "file count").
//   - Repository – a Git repository URL and branch to analyse.
//   - Metric     – a single collected data point for a repository.
//   - Report     – the aggregated analysis results across repositories.
//
// All source implementations must satisfy the Source interface, and all output
// renderers must satisfy the OutputFormatter interface.
package core
