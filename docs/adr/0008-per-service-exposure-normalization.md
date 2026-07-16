# ADR 0008: Per-Service Exposure Normalization and Score Display Honesty

**Status:** Accepted
**Date:** 2026-07-16
**Deciders:** Project team

## Context

ADR 0005 de-sized `dependency_count` and `decision_density` to per-service rates
so a monorepo isn't penalized for its size ("health over size"), but left three
scored metrics on absolute counts: `security_vulnerabilities` (severity-weighted
sum of open alerts), `k8s_deployments` (manifest files), and `container_images`
(distinct image refs). On the platform monorepo these accrue in proportion to the
number of services, so a big repo loses to many small repos at the same per-service
health — the exact size penalty ADR 0005 set out to remove. A review also flagged
score display precision (`41.9`) as implying rigor the formulas don't have, and
the `ci_cd_complexity` label as backwards framing (high complexity read as good).

## Decision

1. **Per-service normalization.** `security_vulnerabilities`, `k8s_deployments`,
   and `container_images` are divided by the repo's service count — distinct
   directories holding a recognized dependency manifest (the same definition
   `dependency_count` already uses, extracted as `countServices`). Refs are
   recalibrated to per-service targets: security 5, k8s 5, container_images 3.
2. **`deploy_targets` stays absolute.** Deploy environments (prod/staging/…) are
   repo-level, not per-service; dividing them by service count would understate a
   genuinely large environment sprawl. Left on its absolute ref (20).
3. **Scores render as whole numbers.** The formulas carry no sub-point precision,
   so the report rounds displayed scores; a decimal would fake rigor.
4. **`ci_cd_complexity` displays as "CI/CD Maturity".** The metric measures how
   much CI automation is present (higher = more), so the label now matches the
   direction. The stored metric-type name is unchanged.

## Details

The per-service refs are **provisional**: they were derived to preserve the real
platform's observed dimension scores under the new denominator (platform ≈ 100
services, from `dependency_count`'s 599 deps ÷ 5.8 per-service), not calibrated
against a peer-repo distribution. They should be recalibrated from percentiles
across peer repositories once that data exists — the review's broader "magic
constants" point, deferred here. This narrows ADR 0005's ref table for the three
metrics; the asymptotic curve and the "health over size" principle are unchanged.

The critical-vulnerability gate (ADR 0007) composes with this: per-service
normalization can score security as healthy, but any open critical still gates the
dimension to F.

## Consequences

- A monorepo and many small repos with equal per-service health now score equally;
  size no longer drives security/infra.
- Every affected score changes; the platform sanity fixture and normalization
  tests are updated to per-service raw values.
- Refs are provisional and flagged as such; a peer-percentile recalibration is the
  intended follow-up.

## Options Considered

- **Normalize per-KLOC instead of per-service** — deferred. Needs a line-count
  collector; per-service reuses an existing denominator and matches ADR 0005.
- **Normalize `deploy_targets` per-service too** — rejected as semantically wrong
  (environments are repo-level).
- **Shrink medium/low severity weights so criticals dominate** — dropped as
  redundant: the ADR 0007 gate already makes any open critical fail the grade.
