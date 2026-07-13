# ADR 0001: Dimension-Based Scoring Model

**Status:** Accepted  
**Date:** 2026-07-11  
**Deciders:** Project team

## Context

ComplexityRadar needs to measure technical complexity across multiple dimensions. The scoring model must be:

- **Understandable** — users should see why a score is what it is
- **Configurable** — different projects may weight areas differently
- **Transparent** — the calculation should be inspectable

## Decision

Score using four fixed Dimensions (Security, Delivery, Infrastructure, Code), each containing related MetricTypes. The overall score is the weighted average of dimension scores.

## Details

1. Each MetricType belongs to exactly one Dimension
2. Each raw Metric value is normalized to 0–100 per-Dimension rules
3. Each Dimension score is the arithmetic mean of its normalized Metrics
4. The Overall score is the weighted average of Dimension scores
5. Weights are user-configurable via TOML config and default to equal distribution

## Consequences

- Users can see which dimension is dragging score down
- Adding a new MetricType automatically feeds into the correct Dimension
- Weights are simple to explain and configure
- Weight non-normalization (sum != 1.0) is handled transparently
