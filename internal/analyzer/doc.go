// Package analyzer provides code analysis functionality for Go projects.
//
// The analyzer examines parsed Go source code metrics and applies configurable
// thresholds to identify code quality issues. It supports:
//
// - File and function-level metrics analysis (size, complexity, comments)
// - Anti-pattern detection through a modular detector system
// - Test coverage integration via go test -cover
// - Dependency analysis and circular dependency detection
// - Aggregate metrics calculation (percentiles, averages, top N)
//
// # Usage
//
// Create an analyzer with configuration:
//
//	cfg := config.Default()
//	analyzer := analyzer.NewAnalyzer(&cfg.Analysis)
//
// Run analysis on parsed metrics:
//
//	result, err := analyzer.Analyze(projectPath, metrics)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The analyzer returns an AnalysisResult containing:
// - Detected issues (warnings and info)
// - Aggregate metrics (averages, percentiles)
// - Test coverage data (if enabled)
// - Dependency analysis results (if enabled)
package analyzer
