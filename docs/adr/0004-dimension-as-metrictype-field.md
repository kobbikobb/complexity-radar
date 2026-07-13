# ADR 0004: Dimension as a Field on MetricType

**Status:** Accepted  
**Date:** 2026-07-11  
**Deciders:** Project team

## Context

Each MetricType belongs to a Dimension. This relationship can be modeled either as a separate `dimensions` table with a foreign key, or as a string field on the `metric_types` table.

## Decision

Store Dimension as a string field directly on `metric_types`, not as a separate table.

## Consequences

- Simpler schema — one less table, no join required
- Dimension is a fixed enum, not user-extensible, so a separate table adds no benefit
- Easy to filter or group by dimension in queries
- Adding a new MetricType is a single INSERT

## Alternatives Considered

- **Dimensions table** with FK from metric_types — normalized but adds a join for every query with no real benefit since dimensions are fixed
