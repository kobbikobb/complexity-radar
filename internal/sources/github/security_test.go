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
		page1 := `[{"state":"open","severity":"critical"}]`
		allPages := `[{"state":"open","severity":"critical"},{"state":"open","severity":"high"},{"state":"dismissed","severity":"critical"}]`
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
