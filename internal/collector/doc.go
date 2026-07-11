// Package collector orchestrates data collection from all configured sources.
//
// The collector is responsible for:
//   - Reading the project configuration
//   - Iterating over repositories
//   - Calling each source to collect metrics
//   - Storing results in the database
//
// Usage:
//
//	collector.Collect(ctx, config)
package collector
