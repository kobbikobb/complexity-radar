package github

import (
	"context"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

func depCount(t *testing.T, files map[string]string) float64 {
	t.Helper()

	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	client := &mockClient{responses: defaultResponses(), fileContents: files}
	client.responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(paths)

	src := NewSourceWithClient(client)
	metrics, err := src.Collect(context.Background(), model.Repository{URL: "github.com/org/repo", Branch: "main"})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	m, ok := findMetric(metrics, model.MetricTypeDependencyTotal)
	if !ok {
		t.Fatal("missing dependency_total metric")
	}
	return m.Value
}

func TestCollectDependencyPerService(t *testing.T) {
	t.Run("should score distinct deps divided by service count", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"app/package.json": `{"dependencies": {"react": "^18", "next": "^13", "lodash": "^4"}}`,
			"svc/go.mod":       "module x\nrequire (\n\tgithub.com/foo/bar v1\n\tgithub.com/baz/qux v2\n)",
		}
		paths := []string{"app/package.json", "svc/go.mod"}
		client := &mockClient{responses: defaultResponses(), fileContents: files}
		client.responses["/repos/org/repo/git/trees/main"] = makeTreeJSON(paths)
		src := NewSourceWithClient(client)

		// Act
		metrics, err := src.Collect(context.Background(), model.Repository{URL: "github.com/org/repo", Branch: "main"})
		if err != nil {
			t.Fatalf("Collect: %v", err)
		}

		// Assert
		scored, _ := findMetric(metrics, model.MetricTypeDependencyCount)
		total, _ := findMetric(metrics, model.MetricTypeDependencyTotal)
		if scored.Value != 2.5 {
			t.Errorf("dependency_count = %v, want 2.5 (5 deps / 2 services)", scored.Value)
		}
		if total.Value != 5 {
			t.Errorf("dependency_total = %v, want 5 (raw distinct total)", total.Value)
		}
	})
}

func TestCollectDependencyCountUnion(t *testing.T) {
	t.Run("should count a dep shared by two manifests only once", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"app/package.json": `{"dependencies": {"react": "^18", "next": "^13"}}`,
			"lib/package.json": `{"dependencies": {"react": "^18", "lodash": "^4"}}`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 3 {
			t.Errorf("distinct dependency count = %v, want 3 (react, next, lodash)", got)
		}
	})

	t.Run("should find a pyproject.toml dependency", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"pyproject.toml": `[project]
dependencies = [
    "requests>=2.0",
    "flask",
]

[tool.poetry.dependencies]
python = "^3.11"
django = "^4.0"`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 3 {
			t.Errorf("distinct dependency count = %v, want 3 (requests, flask, django)", got)
		}
	})

	t.Run("should find a csproj PackageReference", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"src/App.csproj": `<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Serilog" Version="3.0.0" />
  </ItemGroup>
</Project>`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 2 {
			t.Errorf("distinct dependency count = %v, want 2 (Newtonsoft.Json, Serilog)", got)
		}
	})

	t.Run("should exclude a manifest under node_modules", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"package.json":                  `{"dependencies": {"react": "^18"}}`,
			"node_modules/dep/package.json": `{"dependencies": {"vendored-a": "1", "vendored-b": "2"}}`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 1 {
			t.Errorf("distinct dependency count = %v, want 1 (react only)", got)
		}
	})

	t.Run("should dedup the same dep pinned at different versions", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"requirements.txt":     `requests>=2.0`,
			"requirements-dev.txt": `requests==3.1`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 1 {
			t.Errorf("distinct dependency count = %v, want 1 (requests, version-stripped)", got)
		}
	})

	t.Run("should exclude vendored paths but keep substring matches", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"vendor/dep/package.json":            `{"dependencies": {"vendored": "1"}}`,
			"my-node_modules-utils/package.json": `{"dependencies": {"realdep": "1"}}`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 1 {
			t.Errorf("distinct dependency count = %v, want 1 (realdep only)", got)
		}
	})

	t.Run("should count poetry group and optional dependencies", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"pyproject.toml": `[tool.poetry.group.dev.dependencies]
pytest = "^7.0"

[project.optional-dependencies]
test = ["coverage", "tox>=4"]`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 3 {
			t.Errorf("distinct dependency count = %v, want 3 (pytest, coverage, tox)", got)
		}
	})

	t.Run("should pick up a requirements-dev.txt", func(t *testing.T) {
		// Arrange
		files := map[string]string{
			"requirements-dev.txt": `pytest==7.0
black`,
		}

		// Act
		got := depCount(t, files)

		// Assert
		if got != 2 {
			t.Errorf("distinct dependency count = %v, want 2 (pytest, black)", got)
		}
	})
}
