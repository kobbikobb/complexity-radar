# Agents

## Before committing

Always run before committing:

```bash
go test ./... && golangci-lint run ./...
```

If either fails, fix the issue before committing. Never commit code that breaks tests or lint.

## When creating new packages

- Put new internal packages under `internal/`
- Create test files alongside source files
- Use in-memory SQLite (`:memory:`) for store-dependent tests
- Use mock interfaces (not real implementations) for external dependencies
- Run `go test ./...` and `golangci-lint run ./...` before pushing

## Code style

- No comments unless asked
- Follow existing patterns in the codebase
- Use `t.Cleanup` instead of `defer` in tests for resource cleanup
- Use table-driven tests where practical
