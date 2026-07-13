# ADR 0002: Interface-Based Source Extensibility

**Status:** Accepted  
**Date:** 2026-07-11  
**Deciders:** Project team

## Context

ComplexityRadar must collect metrics from multiple external systems (GitHub, Jira, Grafana, etc.). New sources will be added over time. The design must allow adding sources without changing core logic.

## Decision

Define the `Source` interface in the `sources` package and implement it per source. The collector orchestrates Source calls in a loop — it never knows the concrete Source type.

## Consequences

- Adding a new source means writing one package implementing `Source` — no core changes
- Sources are trivially mockable for testing
- No runtime plugin loading overhead (all sources are compiled in)
- Each source declares which MetricTypes it supports, enabling validation at collection time
