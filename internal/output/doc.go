// Package output defines the OutputFormatter interface and implementations.
//
// Interface:
//
//	type OutputFormatter interface {
//	    Format(report Report) string
//	}
//
// Built-in formatters:
//   - terminal: Human-readable terminal output (v1)
//   - json: Machine-readable JSON (planned)
//   - html: Visual HTML reports (planned)
package output
