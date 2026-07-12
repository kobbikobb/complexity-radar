# Contributing to ComplexityRadar

Thank you for your interest in contributing!

## Prerequisites

- Go 1.25+
- `gh` CLI (authenticated)
- `golangci-lint` (for linting)
- `goreleaser` (for releases, optional)

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/kobbikobb/complexity-radar.git
   cd complexity-radar
   ```

2. Build the project:
   ```bash
   make build
   ```

3. Run tests:
   ```bash
   make test
   ```

4. Run linter:
   ```bash
   make lint
   ```

## Making Changes

1. Create a branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```

2. Make your changes

3. Run tests and lint:
   ```bash
   make test
   make lint
   ```

4. Commit your changes with a clear message

5. Push and create a pull request

## Pull Request Guidelines

- Keep PRs focused on a single change
- Include tests for new functionality
- Update documentation if needed
- Ensure CI passes

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `make lint` before committing

## Questions?

Open an issue on GitHub.
