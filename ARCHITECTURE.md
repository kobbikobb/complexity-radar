# Architecture

High-level architecture of ComplexityRadar.

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI (cobra)                             │
│   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   │
│   │  radar   │   │  radar   │   │  radar   │   │  radar   │   │
│   │ collect  │   │ report   │   │   scan   │   │ version  │   │
│   └────┬─────┘   └────┬─────┘   └────┬─────┘   └──────────┘   │
│        │              │              │                          │
├────────┼──────────────┼──────────────┼──────────────────────────┤
│        │         Internal Packages   │                          │
│        │              │              │                          │
│   ┌────▼─────────────────────────────▼──────┐                  │
│   │              Core Package               │                  │
│   │  ┌─────────────┐  ┌─────────────────┐  │                  │
│   │  │   Types     │  │   Interfaces    │  │                  │
│   │  │ - Metric    │  │ - Source        │  │                  │
│   │  │ - Repository│  │ - OutputFormatter│ │                  │
│   │  │ - Report    │  │                 │  │                  │
│   │  └─────────────┘  └─────────────────┘  │                  │
│   └────────────────────────────────────────┘                  │
│                                                                │
│   ┌──────────────────────────┐   ┌─────────────┐              │
│   │        Sources           │   │    Output   │              │
│   │  ┌───────────────┐      │   │  ┌────────┐ │              │
│   │  │    GitHub     │      │   │  │Terminal│ │              │
│   │  │  (v1 source)  │      │   │  └────────┘ │              │
│   │  └───────────────┘      │   │  (more...)  │              │
│   │  (extensible)           │   └─────────────┘              │
│   └──────────────────────────┘                               │
│                                                                │
├────────────────────────────────────────────────────────────────┤
│                     Storage (SQLite)                           │
│   ┌──────────┐   ┌──────────┐   ┌──────────┐                  │
│   │ Projects │   │ Metrics  │   │  Scores  │                  │
│   └──────────┘   └──────────┘   └──────────┘                  │
└────────────────────────────────────────────────────────────────┘
```

## Data Flow

```
                    ┌─────────────┐
                    │   Config    │
                    │  (.toml)    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  Collector  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
       ┌──────▼──────┐    │     ┌──────▼──────┐
       │   GitHub    │    │     │  (future)   │
       │   Source    │    │     │   Jira      │
       └──────┬──────┘    │     └──────┬──────┘
              │            │            │
              └────────────┼────────────┘
                           │
                    ┌──────▼──────┐
                    │   SQLite    │
                    │  (metrics)  │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Scorer    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Output    │
                    │  (terminal) │
                    └─────────────┘
```

## Key Interfaces

Interfaces are defined in `internal/core/interfaces.go`:

### Source

```go
type Source interface {
    Name() string
    Collect(ctx context.Context, repo Repository) ([]Metric, error)
    SupportedMetrics() []MetricType
}
```

### OutputFormatter

```go
type OutputFormatter interface {
    Format(report Report) string
}
```

## Directory Structure

```
complexity-radar/
├── cmd/radar/              # CLI entrypoint
│   ├── main.go             # Entry point
│   ├── collect.go          # Collect command
│   ├── report.go           # Report command
│   └── scan.go             # Scan command (collect + report)
├── internal/
│   ├── core/               # Core domain types and interfaces
│   │   ├── metrics.go      # MetricType, Repository, Metric, Report types
│   │   ├── interfaces.go   # Source, OutputFormatter interfaces
│   │   └── doc.go          # Package documentation
│   ├── sources/            # Source implementations
│   └── output/             # Report formatters
├── configs/                # Default config templates
├── docs/                   # Documentation
└── migrations/             # SQLite schema migrations
```
