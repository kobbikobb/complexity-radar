package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Project      ProjectConfig      `toml:"project"`
	Repositories []RepositoryConfig `toml:"repositories"`
	Weights      WeightsConfig      `toml:"weights"`
	Thresholds   ThresholdsConfig   `toml:"thresholds"`
}

type ProjectConfig struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}

type RepositoryConfig struct {
	URL    string `toml:"url"`
	Branch string `toml:"branch"`
}

// WeightsConfig defines dimension weights for scoring (must sum to 1.0).
type WeightsConfig struct {
	Security       float64 `toml:"security"`
	Delivery       float64 `toml:"delivery"`
	Infrastructure float64 `toml:"infrastructure"`
	Code           float64 `toml:"code"`
}

type ThresholdsConfig struct {
	SecurityVulnerabilitiesCriticalMax *int     `toml:"security_vulnerabilities_critical_max"`
	BuildSuccessRatioMin               *float64 `toml:"build_success_ratio_min"`
	StalePullRequestsMax               *int     `toml:"stale_pull_requests_max"`
	K8sDeploymentsMax                  *int     `toml:"k8s_deployments_max"`
	ContainerImagesMax                 *int     `toml:"container_images_max"`
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

// Load reads and parses a TOML config file, applies defaults, and validates.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return cfg, nil
}

// Parse parses TOML bytes into a Config, applies defaults, and validates.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid TOML: %w", err)
	}

	applyDefaults(&cfg)

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Weights == (WeightsConfig{}) {
		cfg.Weights = DefaultWeights()
	}

	for i := range cfg.Repositories {
		if cfg.Repositories[i].Branch == "" {
			cfg.Repositories[i].Branch = "main"
		}
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
		} else if !isValidRepoURL(repo.URL) {
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

func isValidRepoURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return false
	}
	return strings.Contains(url, "/")
}
