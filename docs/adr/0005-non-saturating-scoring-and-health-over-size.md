# ADR 0005: Non-Saturating Scoring and Health Over Size

**Status:** Accepted  
**Date:** 2026-07-15  
**Deciders:** Project team

## Context

On a real platform run the overall score swung from 70/C to 34/F between iterations, and neither number was true. Two structural flaws drove the swings:

1. **Saturating curve** — lower-is-better metrics used `100 - logNorm(value, ref)*100`. Once a raw value passed its `ref`, the term saturated and the metric floored to 0. A metric that was merely "above target" scored identically to one that was catastrophically bad, so scores collapsed to F for ordinary repositories.
2. **Absolute counts penalize size** — metrics like dependency count used raw totals, so a larger platform was punished for being large rather than for being unhealthy. Size, not health, dominated the score.

The scoring must reflect health on a continuous scale and stop flooring healthy-but-imperfect repositories to 0.

## Decision

1. **Asymptotic curve for lower-is-better metrics.** Replace the log curve with `score = 100 * ref / (value + ref)`. It equals 100 at 0, 50 at `ref`, and approaches but never reaches 0 as `value` grows — no floor. Refs are calibrated against a real platform run.
2. **Decision density over raw size.** Code health is measured by decision density (a per-100-LOC intensity), not by absolute counts.
3. **Dependency count: scored but dampened, per-service.** Dependency count stays in the score as a per-service ratio (not a raw total), but with a small sub-weight so it is a minor signal, not a size penalty.
4. **Per-metric sub-weights.** `MetricType` gains a `Weight`; dimension scores are a weighted mean. Within Code, decision density weighs 0.8 and dependency count 0.2. Equal weights reduce to a plain mean, so single-metric and uniform dimensions are unchanged.
5. **Security dedup by `ghsa_id`** (already merged upstream) removes double-counted alerts feeding the security metric.
6. **Security dimension stays single-metric.** Severity is already encoded in the weighted `security_vulnerabilities` sum; the severity counts remain display-only.
7. **Sanity gate + inline methodology.** A regression test pins real platform raw values and asserts the outcome stays ≈ C/D (not F), and the report prints each metric's scoring function inline by default plus a warn line when the dimension spread looks curve-broken.

## Details

Calibrated refs (lower-is-better): security_vulnerabilities 70, stale_prs 45, decision_density 20, dependency_count 8 (per-service), k8s_deployments 100, container_images 40, deploy_targets 20. Severity display counts use asymptotic refs (crit 5, high 40, med 60, low 40) purely for sane display. build_time, deploy_frequency, build_success_ratio, and ci_cd_complexity keep their existing linear / log forms.

## Consequences

- Healthy-but-imperfect repositories no longer floor to 0; scores move smoothly with health.
- Larger platforms are no longer penalized simply for size.
- Every score changes with this release; the regression sanity gate guards the calibration.
- Methodology is visible in the default report, so a score is inspectable without `--explain`.
- `logNormalize` is retained only for ci_cd_complexity (higher-is-better automation maturity).

## Options Considered

- **Keep log curve, enlarge refs** — rejected. It only delays saturation; any value far enough past `ref` still floors to 0.
- **Demote dependency count to display-only, or score vulnerable-% instead** — rejected/deferred. Dropping it loses a real signal; a vulnerable-dependency ratio needs collector work not yet available. Dampening via sub-weight captures most of the benefit now.
- **Fold severity counts into the score** — rejected. Severity is already encoded in the weighted security sum; scoring the counts too would double-count severity.
