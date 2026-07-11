// Package scorer calculates complexity scores from collected metrics.
//
// The scorer is responsible for:
//   - Normalizing metrics to 0-100 scale
//   - Grouping metrics by dimension
//   - Applying weights to calculate dimension scores
//   - Aggregating dimension scores into an overall score
//
// Scoring model:
//
//	dimension_score = weighted_average(metrics_in_dimension)
//	overall_score = weighted_average(dimension_scores)
package scorer
