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
	m, ok := findMetric(metrics, model.MetricTypeDependencyCount)
	if !ok {
		t.Fatal("missing dependency_count metric")
	}
	return m.Value
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
