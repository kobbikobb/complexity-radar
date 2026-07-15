# ADR 0006: External Opt-In Vendor Sources

**Status:** Accepted  
**Date:** 2026-07-15  
**Deciders:** Project team

## Context

Until now every metric came from GitHub, reachable for any configured repo with no
extra setup. Some metrics need an external vendor that only some users have and that
requires credentials — the first being DevCycle for feature-flag debt. We need a way
to add such sources without forcing setup on users who don't use the vendor, and
without changing the collector each time (see ADR 0002).

## Decision

- **Composite source.** `sources.Default()` returns a `Multi` that merges all sources
  (`github`, `devcycle`, …). The runner and collector still receive a single
  `model.Source`; `Multi` is the one place new vendors are registered.
- **Self-skipping, opt-in.** A vendor source is always in the composite but returns
  `(nil, nil)` from `Collect` when it isn't configured for a repo. Opt-in is a
  graceful no-op, never an error, so registration stays static.
- **Per-repo, non-secret config in the DB; secrets in env.** The DevCycle project key
  is a `repositories` column set by the init wizard. Credentials
  (`DEVCYCLE_CLIENT_ID` / `DEVCYCLE_CLIENT_SECRET`) come from the environment, never
  the DB — matching the existing `gh`-CLI auth idiom.
- **A configured vendor's failure fails that repo's collection**, same as a GitHub
  failure. There is no partial-success machinery.

## Consequences

- New vendors follow the DevCycle pattern: a package implementing `Source` that
  self-skips when unconfigured, plus one line in `sources.Default()`.
- Non-users see no behavior change and need no credentials.
- Vendor secrets never touch the database.
- A vendor outage can fail collection for repos that opted in; acceptable for now,
  revisit if partial-success reporting is wanted.
