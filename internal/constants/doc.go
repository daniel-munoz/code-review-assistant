// Package constants defines configuration defaults and thresholds.
//
// This package centralizes all magic numbers and default values used
// throughout the application. By extracting constants to a dedicated
// package, we improve maintainability and make it easier to understand
// and modify thresholds.
//
// # Threshold Categories
//
// - Analysis thresholds: Default values for file size, function length, complexity
// - Anti-pattern thresholds: Limits for parameters, nesting, return statements
// - Coverage settings: Defaults for test coverage analysis
// - Dependency settings: Limits for imports and external dependencies
// - Reporting constants: Limits for top-N reports, percentile calculations
//
// # Usage
//
// Import and reference constants:
//
//	import "github.com/daniel-munoz/code-review-assistant/internal/constants"
//
//	if functionLength > constants.DefaultLongFunctionThreshold {
//	    // Function is too long
//	}
//
//	p95 := calculatePercentile(values, constants.Percentile95)
//
// All constants are exported and can be used throughout the codebase
// to ensure consistent threshold application.
package constants
