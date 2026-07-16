package devcycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	authURL  = "https://auth.devcycle.com/oauth/token"
	apiBase  = "https://api.devcycle.com/v2"
	audience = "https://api.devcycle.com/"
	perPage  = 100
)

// Feature is a DevCycle feature flag. A non-nil Staleness means DevCycle
// has flagged the feature as stale tech debt.
type Feature struct {
	Key       string     `json:"key"`
	Status    string     `json:"status"`
	UpdatedAt string     `json:"updatedAt"`
	Staleness *Staleness `json:"staleness"`
}

type Staleness struct {
	Reason string `json:"reason"`
}

// Client talks to the DevCycle Management API using OAuth2 client credentials.
type Client struct {
	clientID     string
	clientSecret string
	http         *http.Client
	token        string
}

func NewClient(clientID, clientSecret string, hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{clientID: clientID, clientSecret: clientSecret, http: hc}
}

func (c *Client) authenticate(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"audience":      audience,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("devcycle auth failed: %s: %s", resp.Status, data)
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("parsing devcycle token: %w", err)
	}
	c.token = out.AccessToken
	return nil
}

// ListFeatures returns all features for a project, paging until exhausted.
func (c *Client) ListFeatures(ctx context.Context, projectKey string) ([]Feature, error) {
	if c.token == "" {
		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}
	}

	var all []Feature
	for page := 1; ; page++ {
		escaped := escapePathSegments(projectKey)
		endpoint := fmt.Sprintf("%s/projects/%s/features?perPage=%d&page=%d",
			apiBase, escaped, perPage, page)
		batch, err := c.getFeatures(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

func (c *Client) getFeatures(ctx context.Context, endpoint string) ([]Feature, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("devcycle features request failed (%s): %s", strconv.Itoa(resp.StatusCode), data)
	}

	var features []Feature
	if err := json.Unmarshal(data, &features); err != nil {
		return nil, fmt.Errorf("parsing devcycle features: %w", err)
	}
	return features, nil
}

func escapePathSegments(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
