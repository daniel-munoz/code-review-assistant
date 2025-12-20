// Package detectors provides anti-pattern detection for Go code.
//
// The detectors package implements a modular system for detecting code
// anti-patterns and quality issues. Each detector focuses on a specific
// pattern and can be enabled/disabled via configuration.
//
// # Available Detectors
//
// - ParameterCountDetector: Detects functions with too many parameters
// - NestingDepthDetector: Detects deeply nested code blocks
// - ReturnStatementDetector: Detects functions with too many return statements
// - MagicNumberDetector: Detects magic numbers in code
// - DuplicateErrorDetector: Detects repetitive error handling patterns
//
// # Detector Interface
//
// All detectors implement the Detector interface:
//
//	type Detector interface {
//	    Name() string
//	    Enabled(cfg *config.AnalysisConfig) bool
//	    Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics,
//	           fn *parser.FunctionMetrics, fset *token.FileSet,
//	           funcDecl *ast.FuncDecl) []*Issue
//	}
//
// # Usage
//
// Create a detector registry:
//
//	cfg := config.Default()
//	registry := detectors.NewRegistry(&cfg.Analysis)
//
// Run all enabled detectors on a function:
//
//	issues := registry.RunAll(fileMetrics, funcMetrics, fset, funcDecl)
//
// The registry automatically skips disabled detectors based on configuration.
package detectors
