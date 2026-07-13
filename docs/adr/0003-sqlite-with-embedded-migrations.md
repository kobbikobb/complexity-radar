# ADR 0003: SQLite with Embedded Migrations

**Status:** Accepted  
**Date:** 2026-07-11  
**Deciders:** Project team

## Context

ComplexityRadar is a CLI tool that must persist historical metric data for trend tracking. It cannot assume a database server is available. It must be portable — single file, zero setup.

## Decision

Use SQLite via `modernc.org/sqlite` (pure Go, no CGO) with embedded `.sql` migration files shipped inside the binary via `embed.FS`.

## Consequences

- Zero database setup — the binary creates and migrates its own database
- Single file is portable and easy to back up
- Prevents drift between schema and code (migrations are compiled in)
- SQLite can handle the expected data volume (hundreds of metrics per scan, per repo)
- No CGO dependency simplifies cross-compilation
