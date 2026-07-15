package github

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/kobbikobb/complexity-radar/internal/model"
)

// pagingAlertClient returns only page 1 from Get but all pages from GetPaginated,
// so a test can prove the collector reads every page instead of a single Get.
type pagingAlertClient struct {
	page1 json.RawMessage
	all   json.RawMessage
}

func (c *pagingAlertClient) Get(context.Context, string) (json.RawMessage, error) {
	return c.page1, nil
}

func (c *pagingAlertClient) GetWithParams(context.Context, string, map[string]string) (json.RawMessage, error) {
	return c.page1, nil
}

func (c *pagingAlertClient) GetPaginated(context.Context, string, map[string]string, int) (json.RawMessage, error) {
	return c.all, nil
}

func (c *pagingAlertClient) GetFileContent(context.Context, string, string, string, string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func TestCollectSecurityVulnerabilitiesPaginates(t *testing.T) {
	t.Run("should count open alerts across more than one page", func(t *testing.T) {
		// Arrange
		page1 := `[{"state":"open","security_advisory":{"ghsa_id":"GHSA-crit","severity":"critical"}}]`
		allPages := `[{"state":"open","security_advisory":{"ghsa_id":"GHSA-crit","severity":"critical"}},{"state":"open","security_advisory":{"ghsa_id":"GHSA-high","severity":"high"}},{"state":"dismissed","security_advisory":{"ghsa_id":"GHSA-crit2","severity":"critical"}}]`
		client := &pagingAlertClient{page1: json.RawMessage(page1), all: json.RawMessage(allPages)}
		src := NewSourceWithClient(client)

		// Act
		metrics, err := src.collectSecurityVulnerabilities(context.Background(), "org", "repo")
		if err != nil {
			t.Fatalf("collectSecurityVulnerabilities: %v", err)
		}

		// Assert
		m, ok := findMetric(metrics, model.MetricTypeSecurityVulnerabilities)
		if !ok {
			t.Fatal("missing security_vulnerabilities metric")
		}
		// page1 alone (single Get) would score critical=1.0; paginated counts page 2's
		// open high=0.7 too, while the dismissed alert stays filtered out: 1.0+0.7=1.7.
		want := 1.7
		if math.Abs(m.Value-want) > 1e-9 {
			t.Errorf("security vulnerabilities = %v, want %v (page 2 must be counted, dismissed excluded)", m.Value, want)
		}
	})
}

func TestCollectSecurityVulnerabilitiesDedup(t *testing.T) {
	t.Run("should count two alerts sharing a ghsa_id as one advisory", func(t *testing.T) {
		// Arrange
		alerts := `[{"state":"open","security_advisory":{"ghsa_id":"GHSA-x","severity":"high"},"dependency":{"scope":"runtime"}},{"state":"open","security_advisory":{"ghsa_id":"GHSA-x","severity":"high"},"dependency":{"scope":"runtime"}}]`
		client := &pagingAlertClient{page1: json.RawMessage(alerts), all: json.RawMessage(alerts)}
		src := NewSourceWithClient(client)

		// Act
		metrics, err := src.collectSecurityVulnerabilities(context.Background(), "org", "repo")
		if err != nil {
			t.Fatalf("collectSecurityVulnerabilities: %v", err)
		}

		// Assert
		weighted, _ := findMetric(metrics, model.MetricTypeSecurityVulnerabilities)
		if math.Abs(weighted.Value-0.7) > 1e-9 {
			t.Errorf("weighted sum = %v, want 0.7 (one distinct high advisory)", weighted.Value)
		}
		high, _ := findMetric(metrics, model.MetricTypeSecurityHigh)
		if high.Value != 1 {
			t.Errorf("high count = %v, want 1", high.Value)
		}
	})

	t.Run("should read severity from security_advisory", func(t *testing.T) {
		// Arrange
		alerts := `[{"state":"open","severity":null,"security_advisory":{"ghsa_id":"GHSA-y","severity":"critical"},"security_vulnerability":{"severity":"low"}}]`
		client := &pagingAlertClient{page1: json.RawMessage(alerts), all: json.RawMessage(alerts)}
		src := NewSourceWithClient(client)

		// Act
		metrics, err := src.collectSecurityVulnerabilities(context.Background(), "org", "repo")
		if err != nil {
			t.Fatalf("collectSecurityVulnerabilities: %v", err)
		}

		// Assert
		crit, _ := findMetric(metrics, model.MetricTypeSecurityCritical)
		if crit.Value != 1 {
			t.Errorf("critical count = %v, want 1 (severity from security_advisory)", crit.Value)
		}
	})

	t.Run("should weight a dev-then-runtime advisory as runtime", func(t *testing.T) {
		// Arrange
		alerts := `[{"state":"open","security_advisory":{"ghsa_id":"GHSA-z","severity":"critical"},"dependency":{"scope":"development"}},{"state":"open","security_advisory":{"ghsa_id":"GHSA-z","severity":"critical"},"dependency":{"scope":"runtime"}}]`
		client := &pagingAlertClient{page1: json.RawMessage(alerts), all: json.RawMessage(alerts)}
		src := NewSourceWithClient(client)

		// Act
		metrics, err := src.collectSecurityVulnerabilities(context.Background(), "org", "repo")
		if err != nil {
			t.Fatalf("collectSecurityVulnerabilities: %v", err)
		}

		// Assert
		weighted, _ := findMetric(metrics, model.MetricTypeSecurityVulnerabilities)
		if math.Abs(weighted.Value-1.0) > 1e-9 {
			t.Errorf("weighted sum = %v, want 1.0 (runtime in one manifest = full weight)", weighted.Value)
		}
	})
}
