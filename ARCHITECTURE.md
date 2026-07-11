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
│   ┌────▼─────┐   ┌────▼─────┐       │                          │
│   │Collector │   │ Scorer   │       │                          │
│   │          │   │          │       │                          │
│   │ - reads  │   │ - norma- │       │                          │
│   │   config │   │   lize   │       │                          │
│   │ - calls  │   │ - weight │       │                          │
│   │   sources│   │ - score  │       │                          │
│   └────┬─────┘   └──────────┘       │                          │
│        │                            │                          │
│   ┌────▼─────────────────┐   ┌──────▼──────┐                  │
│   │     Sources          │   │   Output     │                  │
│   │  ┌───────────────┐   │   │  ┌────────┐ │                  │
│   │  │    GitHub     │   │   │  │Terminal│ │                  │
│   │  │  (v1 source)  │   │   │  └────────┘ │                  │
│   │  └───────────────┘   │   │  (more...)  │                  │
│   │  (extensible)        │   └─────────────┘                  │
│   └──────────────────────┘                                    │
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
│   ├── root.go             # Root command + subcommands
│   ├── collect.go          # Collect command
│   ├── report.go           # Report command
│   └── scan.go             # Scan command (collect + report)
├── internal/
│   ├── collector/          # Data collection orchestration
│   ├── scorer/             # Scoring engine
│   ├── sources/            # Source implementations
│   │   └── github/         # GitHub source (v1)
│   ├── store/              # SQLite storage layer
│   ├── model/              # Domain types
│   └── output/             # Report formatters
│       └── terminal/       # Terminal output (v1)
├── pkg/                    # Public API (if needed)
├── configs/                # Default config templates
├── docs/                   # Documentation
└── migrations/             # SQLite schema migrations
```
