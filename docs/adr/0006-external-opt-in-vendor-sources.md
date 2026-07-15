# ADR 0006: External Opt-In Vendor Sources

**Status:** Accepted  
**Date:** 2026-07-15  
**Deciders:** Project team

## Context

Until now every metric came from GitHub, per repo, reachable with no extra setup.
Some metrics need an external vendor that only some users have, requires
credentials, and is scoped to a whole project rather than a repo — the first
being DevCycle feature-flag debt, where flags belong to a DevCycle project, not a
git repo. We need to add such sources without forcing setup on non-users and
without changing the collector each time (ADR 0002), and to surface a
project-scoped metric even though every metric so far has been repo-scoped.

## Decision

- **Two source axes.** `model.Source` stays repo-scoped (`Collect(repo)`); a new
  `model.ProjectSource` is project-scoped (`CollectProject(project)`).
  `sources.Default()` composes repo sources, `sources.DefaultProject()` composes
  project sources — each the single place its kind of vendor is registered.
- **Self-skipping, opt-in.** A vendor source is always registered but returns
  `(nil, nil)` when unconfigured, so opt-in is a graceful no-op, never an error.
- **Project-scoped config and credentials in the DB.** The DevCycle project key,
  client ID, and client secret are `projects` columns set by the init wizard. The
  local SQLite DB (`*.db`, gitignored) is the single place a user configures a
  project, so credentials live there too rather than in the environment. Tradeoff:
  the client secret is stored plaintext at rest; the DB file is gitignored and
  should be kept out of shared/backed-up locations.
- **Project-level scoring via a rollup report.** Project metrics are stored in a
  `project_metrics` table. The report layer builds a project rollup: each repo
  metric is averaged across repos (skipping no-data sentinels) and project metrics
  are added directly, then the combined bag is scored by the existing scorer, so
  `feature_flag_debt` folds into the project's Code dimension and overall score.
- **A configured vendor's failure is recorded, not fatal.** Project-source errors
  are collected into the result's `ProjectErrors` and shown; they don't abort the
  run.

## Consequences

- New repo vendors implement `Source` + one line in `Default()`; new project
  vendors implement `ProjectSource` + one line in `DefaultProject()`.
- Non-users see no behavior change and need no credentials.
- Vendor secrets never touch the database.
- Radar gains its first project-level scored surface (the rollup report), shown
  above the per-repo reports.
- Averaging repo metrics for the rollup is a deliberate simplification; revisit if
  a more principled cross-repo aggregation is wanted.
