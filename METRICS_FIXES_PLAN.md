# Metrics Fixes Plan

Findings from an independent audit against the Lucinity platform monorepo. Each section is self-contained and can be handed to a single prompt. Rule for all sections: **fix collection/scoring logic, never hand-edit scores.** Add/adjust tests (`should` + AAA). Confirm `make test` green.

---

## 1. Stale PRs collector counts zero (should be ~46)

**Root cause:** `internal/sources/github/stale.go`
- `:71` `if isBotPR(pr) || pr.Draft { continue }` drops all bot PRs; the real stale backlog is almost entirely dependabot/Snyk/patches-bot.
- `:64` threshold is `now-14d`; ground truth wants a 7d floor.
- `:28,:48-50` `maxStalePRpages=10` + `sort=updated&direction=desc` puts the stalest PRs on the *last* pages — they get truncated on large repos.

**Ground truth (GitHub API):** 46 open PRs idle ≥7d, 39 ≥14d, 24 ≥30d, 13 ≥60d (oldest 69d).

**Fix:**
- Count bot PRs as stale (they are un-actioned dep/security debt). Document the decision in a one-line comment.
- Sort `direction=asc` so the stale tail is page 1.
- Report a histogram: `≥7d / ≥14d / ≥30d / ≥60d`. Score off the ≥7d count (or a weighted blend), not a single 14d number.
- Keep the draft exclusion.

**Acceptance:** collector returns ~46 for the platform repo; stale-PR score lands in F/D tier, not 100.

---

## 2. Dependency Count averages per-manifest (26.12, should be ~576)

**Root cause:** `internal/sources/github/dependencies.go`
- `:61` `Value: total / float64(manifestCount)` — averages deps-per-manifest instead of unioning distinct deps. The `.12` decimal is the tell.
- `:13-23` `dependencyFiles` has no `pyproject.toml` (PEP 621) and no `*.csproj` (`PackageReference`) → all NuGet + modern Python deps invisible.
- No dedup across manifests; no `node_modules` exclusion.

**Ground truth:** ~405 npm + 79 Python + 92 NuGet ≈ 576 distinct third-party deps across ~1551 manifests.

**Fix:**
- Collect into `map[stack]map[name]bool`; value = sum of distinct names.
- Add `pyproject.toml` and `*.csproj` parsers. Match `*.csproj` and `requirements*.txt` by suffix, not exact filename.
- Exclude any path under `node_modules/`, `vendor/`, `.venv/`.
- Emit a per-stack breakdown alongside the total.

**Acceptance:** integer count ≈ 576 with per-stack split; dependency score drops accordingly.

---

## 3. Averaging trio → distinct counts (deploy targets 0.09, k8s 9.88)

**Root cause:** same averaging bug as #2.
- `internal/sources/github/deploy_targets.go:34` `len(targets)/workflowCount` → fractional, then scores ~98 (looks perfect, means nothing).
- `internal/sources/github/k8s.go:43` `count/len(subdirs)` → the `.88` tell.

**Fix:** return the distinct count directly (`len(targets)`, manifest `count`). Drop the divisor. Re-check the `ref*` constants in `normalize.go` still make sense against un-averaged magnitudes.

**Acceptance:** both metrics are integers; scores reflect real scale.

---

## 4. Security dimension is under-measured and rides one number

**Root cause:** `internal/sources/github/security.go`
- `:45` `s.client.Get` (single page, no `per_page`, no `state=open` in query) — client-side filters ~first 30 alerts only. On >30 alerts the score is a sample.
- `internal/model/model.go:82-89` `security_critical/high/medium/low` are `DisplayMetricTypes` — **not scored**. 6 highs + 11 mediums move nothing; only the weighted-sum `security_vulnerabilities` drives the score.
- `security_vulnerabilities` is a *weighted sum* but displayed with unit `count` (misleading).

**Fix:**
- Switch to `GetPaginated` with `state=open&per_page=100`.
- Decide: either fold critical/high into the scored set, or keep the single weighted-sum but justify it in the methodology (see #8). Don't leave severity counts collected-but-ignored silently.
- Fix the unit label (`weighted` not `count`).

**Acceptance:** alert count matches API ground truth on a >30-alert repo; severity breakdown either scores or is explicitly documented as display-only.

---

## 5. CI/CD Complexity caps at ~74 and is inverted

**Root cause:** `internal/sources/github/cicd.go` caps raw at 100, but `normalize.go:53-54` uses `refCICDComplexity=500` and no `100 -` prefix → max achievable score `log(101)/log(501)≈0.742`, and *more* CI complexity yields a *higher* score.

**Fix:** decide the intent. If CI maturity/presence is good → keep direction, set `ref` so raw 100 maps to ~100. If CI *complexity* is bad → invert to `100 - logNormalize(...)`. Align `ref` with the actual raw cap either way.

**Acceptance:** a maxed-out raw reaches the intended score end; direction documented.

---

## 6. "Code Complexity" is a mislabeled file-size ratio

**Root cause:** `internal/sources/github/large_file_ratio.go:10-29` computes `filesOver20KB / totalFiles` — a file-*size* ratio, not cyclomatic complexity. `5f894ce` renamed the label but not the substance. A repo-wide ratio hides every hotspot.

**Fix (pick one):**
- **Cheap:** rename display to "Large-file ratio" so it's honest; keep as a code-dimension signal.
- **Real:** per-service/per-stack **p95 cyclomatic complexity** (lizard/gocyclo per language). Aggregate = mean of per-service p95, reported as rollup only; expose per-service in the detailed view.

**Acceptance:** metric name matches what it measures; if p95 route taken, hotspots are visible per service.

---

## 7. Report replays dead DB metric types (code_loc score 0)

**Root cause:** `internal/report/builder.go:55-140` `BuildFromDB` reads metric names/units from DB rows (`GetMetricTypeByID`) and trusts them. Stale `.complexity-radar.db` still carries `code_loc` (id 16) and `code_complexity` (id 11) from before `5f894ce`; `code_loc` hits `normalize.go` `default: return 0` and drags the display.

**Fix:** in `BuildFromDB`, skip DB metric types not present in `MetricTypes()+DisplayMetricTypes()`. Optionally a migration to prune orphaned metric types. If LOC is ever wanted back, normalize per-service or per-deploy-target — never absolute LOC.

**Acceptance:** stale/unknown metrics don't appear in the report; no flat-0 penalty.

---

## 8. Show methodology per metric (`--explain`)

**Root cause:** every row prints `Raw → Score` with a hidden curve (`386s→78.5`, `5.44%→94.6`). Un-auditable.

**Fix:** extend `model.MetricType` with `RawDef` (numerator/denominator/exclusions), `ScoreDef` (the scoring fn + ref constant), and `Source` (data source). Render under a `--explain` flag in `internal/terminal`. Keep the default table compact.

**Acceptance:** `radar report --explain` prints raw definition, scoring function, and source for each metric.

---

## Suggested PR grouping

- **PR 1 (bug):** #1 + #2 — both live, both `github/` collection logic, both wildly wrong.
- **PR 2 (bug):** #3 + #7 — averaging + stale-DB cleanup.
- **PR 3 (bug):** #4 + #5 — security completeness + CI/CD curve.
- **PR 4 (enhancement):** #8 methodology, then #6 complexity decision.
