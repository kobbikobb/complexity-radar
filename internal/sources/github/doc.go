// Package github implements the GitHub source for ComplexityRadar.
//
// This source collects metrics from GitHub using:
//   - GitHub REST API for PRs, issues, Dependabot alerts
//   - GitHub Actions API for build/deploy metrics
//   - File contents for workflow and manifest analysis
//
// Supported metrics:
//   - security_vulnerabilities
//   - build_success_ratio
//   - build_time
//   - deploy_frequency
//   - stale_pull_requests
//   - k8s_deployments
//   - container_images
//   - deploy_targets
//   - ci_complexity
//   - dependency_count
package github
