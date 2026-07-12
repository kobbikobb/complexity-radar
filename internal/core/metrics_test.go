package core

import (
	"testing"
)

func TestRepositoryValidate(t *testing.T) {
	tests := []struct {
		name    string
		repo    Repository
		wantErr string
	}{
		{
			name:    "valid repository",
			repo:    Repository{URL: "https://github.com/user/repo", Branch: "main"},
			wantErr: "",
		},
		{
			name:    "empty URL",
			repo:    Repository{URL: "", Branch: "main"},
			wantErr: "repository URL must not be empty",
		},
		{
			name:    "empty branch",
			repo:    Repository{URL: "https://github.com/user/repo", Branch: ""},
			wantErr: "repository branch must not be empty",
		},
		{
			name:    "both empty",
			repo:    Repository{URL: "", Branch: ""},
			wantErr: "repository URL must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.repo.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Validate() expected error %q, got nil", tt.wantErr)
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Validate() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestProjectReportMetricsByDimension(t *testing.T) {
	tests := []struct {
		name    string
		report  ProjectReport
		want    map[string][]Metric
	}{
		{
			name:   "no metrics",
			report: ProjectReport{Metrics: []Metric{}},
			want:   map[string][]Metric{},
		},
		{
			name: "single dimension",
			report: ProjectReport{
				Metrics: []Metric{
					{Type: MetricType{Name: "loc", Dimension: "size", Unit: "lines"}, Value: 100},
					{Type: MetricType{Name: "files", Dimension: "size", Unit: "count"}, Value: 10},
				},
			},
			want: map[string][]Metric{
				"size": {
					{Type: MetricType{Name: "loc", Dimension: "size", Unit: "lines"}, Value: 100},
					{Type: MetricType{Name: "files", Dimension: "size", Unit: "count"}, Value: 10},
				},
			},
		},
		{
			name: "multiple dimensions",
			report: ProjectReport{
				Metrics: []Metric{
					{Type: MetricType{Name: "loc", Dimension: "size", Unit: "lines"}, Value: 100},
					{Type: MetricType{Name: "complexity", Dimension: "complexity", Unit: "score"}, Value: 5.5},
					{Type: MetricType{Name: "files", Dimension: "size", Unit: "count"}, Value: 10},
					{Type: MetricType{Name: "duplication", Dimension: "complexity", Unit: "percent"}, Value: 12.3},
				},
			},
			want: map[string][]Metric{
				"size": {
					{Type: MetricType{Name: "loc", Dimension: "size", Unit: "lines"}, Value: 100},
					{Type: MetricType{Name: "files", Dimension: "size", Unit: "count"}, Value: 10},
				},
				"complexity": {
					{Type: MetricType{Name: "complexity", Dimension: "complexity", Unit: "score"}, Value: 5.5},
					{Type: MetricType{Name: "duplication", Dimension: "complexity", Unit: "percent"}, Value: 12.3},
				},
			},
		},
		{
			name: "empty dimension string",
			report: ProjectReport{
				Metrics: []Metric{
					{Type: MetricType{Name: "unknown", Dimension: "", Unit: ""}, Value: 0},
				},
			},
			want: map[string][]Metric{
				"": {
					{Type: MetricType{Name: "unknown", Dimension: "", Unit: ""}, Value: 0},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.MetricsByDimension()
			if len(got) != len(tt.want) {
				t.Errorf("MetricsByDimension() returned %d dimensions, want %d", len(got), len(tt.want))
				return
			}
			for dim, wantMetrics := range tt.want {
				gotMetrics, ok := got[dim]
				if !ok {
					t.Errorf("MetricsByDimension() missing dimension %q", dim)
					continue
				}
				if len(gotMetrics) != len(wantMetrics) {
					t.Errorf("dimension %q: got %d metrics, want %d", dim, len(gotMetrics), len(wantMetrics))
					continue
				}
				for i := range wantMetrics {
					if gotMetrics[i] != wantMetrics[i] {
						t.Errorf("dimension %q metric %d: got %v, want %v", dim, i, gotMetrics[i], wantMetrics[i])
					}
				}
			}
		})
	}
}

func TestReportProjectByName(t *testing.T) {
	report := Report{
		Projects: []ProjectReport{
			{Repository: Repository{URL: "github.com/user/alpha", Branch: "main"}, Score: 1.0},
			{Repository: Repository{URL: "github.com/user/beta", Branch: "develop"}, Score: 2.0},
		},
	}

	t.Run("existing project", func(t *testing.T) {
		p, ok := report.ProjectByName("github.com/user/alpha")
		if !ok {
			t.Fatal("ProjectByName() returned false for existing project")
		}
		if p.Repository.URL != "github.com/user/alpha" {
			t.Errorf("got URL %q, want %q", p.Repository.URL, "github.com/user/alpha")
		}
		if p.Score != 1.0 {
			t.Errorf("got Score %v, want 1.0", p.Score)
		}
	})

	t.Run("non-existent project", func(t *testing.T) {
		p, ok := report.ProjectByName("github.com/user/gamma")
		if ok {
			t.Fatal("ProjectByName() returned true for non-existent project")
		}
		if p != nil {
			t.Errorf("expected nil, got %v", p)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		p, ok := report.ProjectByName("")
		if ok {
			t.Fatal("ProjectByName() returned true for empty name")
		}
		if p != nil {
			t.Errorf("expected nil, got %v", p)
		}
	})

	t.Run("empty projects list", func(t *testing.T) {
		empty := Report{}
		p, ok := empty.ProjectByName("anything")
		if ok {
			t.Fatal("ProjectByName() returned true for empty report")
		}
		if p != nil {
			t.Errorf("expected nil, got %v", p)
		}
	})
}

func TestMetricTypeEqual(t *testing.T) {
	base := MetricType{Name: "loc", Dimension: "size", Unit: "lines", Source: "git"}

	t.Run("equal types", func(t *testing.T) {
		other := MetricType{Name: "loc", Dimension: "size", Unit: "lines", Source: "git"}
		if !base.Equal(other) {
			t.Error("Equal() returned false for identical MetricType values")
		}
	})

	t.Run("different name", func(t *testing.T) {
		other := MetricType{Name: "files", Dimension: "size", Unit: "lines", Source: "git"}
		if base.Equal(other) {
			t.Error("Equal() returned true for different Name")
		}
	})

	t.Run("different dimension", func(t *testing.T) {
		other := MetricType{Name: "loc", Dimension: "complexity", Unit: "lines", Source: "git"}
		if base.Equal(other) {
			t.Error("Equal() returned true for different Dimension")
		}
	})

	t.Run("different unit", func(t *testing.T) {
		other := MetricType{Name: "loc", Dimension: "size", Unit: "count", Source: "git"}
		if base.Equal(other) {
			t.Error("Equal() returned true for different Unit")
		}
	})

	t.Run("different source", func(t *testing.T) {
		other := MetricType{Name: "loc", Dimension: "size", Unit: "lines", Source: "sonar"}
		if base.Equal(other) {
			t.Error("Equal() returned true for different Source")
		}
	})

	t.Run("all different", func(t *testing.T) {
		other := MetricType{Name: "duplication", Dimension: "quality", Unit: "percent", Source: "sonar"}
		if base.Equal(other) {
			t.Error("Equal() returned true for completely different MetricType values")
		}
	})

	t.Run("empty values", func(t *testing.T) {
		a := MetricType{}
		b := MetricType{}
		if !a.Equal(b) {
			t.Error("Equal() returned false for two zero-value MetricType")
		}
	})
}
