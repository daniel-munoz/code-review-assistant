// Package cmd implements the command-line interface for the code review assistant.
//
// The cmd package provides Cobra-based CLI commands for analyzing Go projects.
// It handles command-line flag parsing, configuration loading, and orchestrates
// the analysis pipeline.
//
// # Commands
//
// - analyze: Run code analysis on a Go project
//
// # Usage
//
// Run analysis on current directory:
//
//	code-review-assistant analyze
//
// Run analysis on specific directory:
//
//	code-review-assistant analyze /path/to/project
//
// Override configuration via flags:
//
//	code-review-assistant analyze --complexity-threshold 15 --verbose
//
// Use custom configuration file:
//
//	code-review-assistant analyze --config custom-config.yaml
//
// # Configuration Priority
//
// Settings are applied in the following order (highest to lowest):
// 1. CLI flags
// 2. Environment variables
// 3. Configuration file
// 4. Default values
//
// This allows flexible configuration while maintaining sensible defaults.
package cmd
