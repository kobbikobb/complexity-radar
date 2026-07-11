# ComplexityRadar

**Technical complexity scoring for software projects.**

ComplexityRadar helps you understand and measure the complexity of your software projects. It pulls data from multiple sources, calculates weighted complexity scores, and tracks how your scores evolve over time.

## Vision

Software projects grow complex in many dimensions — code, infrastructure, delivery, and security. ComplexityRadar gives you a single, actionable view of that complexity.

- **Pick what to measure** — configure which repositories and metrics matter to your project
- **Pull from multiple sources** — GitHub APIs, workflow files, Kubernetes manifests, dependency files
- **Get a score** — weighted, dimension-based scoring across Security, Delivery, Infrastructure, and Code
- **Track over time** — store historical results to see how complexity evolves

## What We Measure

ComplexityRadar measures technical complexity across four dimensions:

### Security
- Security vulnerabilities (Dependabot alerts)

### Delivery
- Deploy frequency
- Build success ratio
- Build time
- Stale pull requests

### Infrastructure
- Kubernetes deployments
- Container image count
- Deploy targets
- CI/CD pipeline complexity

### Code
- Dependency count

## How It Works

```bash
# Pull data from GitHub
radar collect

# Generate a complexity report
radar report

# Or do both at once
radar scan
```

## Configuration

ComplexityRadar is configured via a TOML file:

```toml
[project]
name = "My App"
description = "Main product"

[[repositories]]
url = "github.com/org/repo"
branch = "main"

[weights]
security = 0.25
delivery = 0.30
infrastructure = 0.25
code = 0.20
```

## Design Principles

- **GitHub-first** — measure what you can pull from GitHub (APIs, files, manifests)
- **Extensible** — add new sources via a simple interface (Jira, Grafana, etc.)
- **Transparent** — see exactly how your score is calculated
- **Historical** — track complexity trends over time
- **CLI-native** — single binary, no server required

## Technical Decisions

See [TECHNICAL_DECISIONS.md](TECHNICAL_DECISIONS.md) for the full record of architectural decisions, including tech stack, database, scoring model, and more.

## Status

**Early design phase** — this project is being shaped through active discussion.

## License

MIT License — see [LICENSE](LICENSE).
