package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Project      ProjectConfig
	Repositories []RepositoryConfig
	Weights      WeightsConfig
	Thresholds   ThresholdsConfig
}

type ProjectConfig struct {
	Name               string
	Description        string
	DevCycleProjectKey string
}

type RepositoryConfig struct {
	URL                string
	Branch             string
	GitopsRepoURL      string
	DeployDetection    string
	IncludePrereleases bool
	TagPrefix          string
}

const (
	DeployDetectionReleases = "github-releases"
	DeployDetectionTags     = "git-tags"
)

// WeightsConfig defines dimension weights for scoring (must sum to 1.0).
type WeightsConfig struct {
	Security       float64
	Delivery       float64
	Infrastructure float64
	Code           float64
}

// Weight returns the weight for the given dimension.
func (w WeightsConfig) Weight(d string) float64 {
	switch d {
	case "security":
		return w.Security
	case "delivery":
		return w.Delivery
	case "infrastructure":
		return w.Infrastructure
	case "code":
		return w.Code
	default:
		return 0
	}
}

type ThresholdsConfig struct {
	SecurityVulnerabilitiesCriticalMax *int
	BuildSuccessRatioMin               *float64
	StalePullRequestsMax               *int
	K8sDeploymentsMax                  *int
	ContainerImagesMax                 *int
}

// DefaultWeights returns the default weight configuration.
func DefaultWeights() WeightsConfig {
	return WeightsConfig{
		Security:       0.25,
		Delivery:       0.30,
		Infrastructure: 0.25,
		Code:           0.20,
	}
}

// Validate checks the config for required fields and constraints.
func Validate(cfg *Config) error {
	var errs []string

	if strings.TrimSpace(cfg.Project.Name) == "" {
		errs = append(errs, "project.name is required")
	}

	if len(cfg.Repositories) == 0 {
		errs = append(errs, "at least one repository is required")
	}

	for i, repo := range cfg.Repositories {
		if strings.TrimSpace(repo.URL) == "" {
			errs = append(errs, fmt.Sprintf("repositories[%d].url is required", i))
		} else if !IsValidRepoURL(repo.URL) {
			errs = append(errs, fmt.Sprintf("repositories[%d].url is not a valid repository URL: %q", i, repo.URL))
		}
	}

	if err := validateWeights(cfg.Weights); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

func validateWeights(w WeightsConfig) error {
	total := w.Security + w.Delivery + w.Infrastructure + w.Code

	if total == 0 {
		return nil
	}

	const epsilon = 0.001
	if total < 1.0-epsilon || total > 1.0+epsilon {
		return fmt.Errorf("weights must sum to 1.0, got %.2f", total)
	}

	if w.Security < 0 || w.Security > 1 {
		return fmt.Errorf("weights.security must be between 0 and 1, got %.2f", w.Security)
	}
	if w.Delivery < 0 || w.Delivery > 1 {
		return fmt.Errorf("weights.delivery must be between 0 and 1, got %.2f", w.Delivery)
	}
	if w.Infrastructure < 0 || w.Infrastructure > 1 {
		return fmt.Errorf("weights.infrastructure must be between 0 and 1, got %.2f", w.Infrastructure)
	}
	if w.Code < 0 || w.Code > 1 {
		return fmt.Errorf("weights.code must be between 0 and 1, got %.2f", w.Code)
	}

	return nil
}

// IsValidRepoURL checks if a URL looks like a valid repository URL.
func IsValidRepoURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return false
	}
	return strings.Contains(url, "/")
}
