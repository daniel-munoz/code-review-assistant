package orchestrator

import (
	"fmt"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/reporter"
)

// Orchestrator coordinates the complete code analysis pipeline.
//
// The orchestrator implements the three-stage analysis workflow:
//   1. Parse: Discover and parse all Go source files
//   2. Analyze: Apply metrics and quality checks
//   3. Report: Format and output results
//
// Each stage is delegated to specialized components (Parser, Analyzer, Reporter)
// that are configured and initialized based on the provided Config.
type Orchestrator struct {
	config   *config.Config
	parser   parser.Parser
	analyzer analyzer.Analyzer
	reporter reporter.Reporter
}

// New creates a new Orchestrator with the given configuration.
//
// This function initializes all pipeline components:
//   - Parser: for AST-based metrics extraction
//   - Analyzer: for applying thresholds and detecting issues
//   - Reporter: for formatted output (based on config.Output.Format)
//
// Returns an error if reporter creation fails (e.g., unsupported format).
func New(cfg *config.Config) (*Orchestrator, error) {
	// Create parser
	p := parser.NewParser()

	// Create analyzer
	a := analyzer.NewAnalyzer(&cfg.Analysis)

	// Create reporter
	r, err := reporter.NewReporter(&cfg.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to create reporter: %w", err)
	}

	return &Orchestrator{
		config:   cfg,
		parser:   p,
		analyzer: a,
		reporter: r,
	}, nil
}

// Run executes the complete analysis pipeline on the specified directory.
//
// The pipeline consists of three stages:
//
//  1. Parse Stage:
//     - Recursively discovers all .go files in targetPath
//     - Applies exclude patterns (vendor/, testdata/, etc.)
//     - Parses each file to extract metrics
//     - Reports parse errors but continues with successful files
//
//  2. Analyze Stage:
//     - Applies thresholds to detect issues
//     - Runs anti-pattern detectors
//     - Executes test coverage analysis (if enabled)
//     - Performs dependency analysis
//
//  3. Report Stage:
//     - Formats results according to configured output format
//     - Outputs to console or other configured destination
//
// Returns an error if:
//   - No Go files are found in targetPath
//   - Analysis fails (unexpected internal error)
//   - Reporting fails (output error)
//
// Parse errors for individual files are reported as warnings but don't fail
// the overall pipeline, allowing partial analysis when some files are malformed.
func (o *Orchestrator) Run(targetPath string) error {
	// Step 1: Parse all Go files
	fileMetrics, parseErrors := o.parser.ParseDirectory(targetPath, o.config.Analysis.ExcludePatterns)

	// Report parse errors but continue with successfully parsed files
	if len(parseErrors) > 0 {
		fmt.Printf("Warning: %d files failed to parse:\n", len(parseErrors))
		for i, err := range parseErrors {
			if i >= 5 {
				fmt.Printf("  ... and %d more errors\n", len(parseErrors)-5)
				break
			}
			fmt.Printf("  - %v\n", err)
		}
		fmt.Println()
	}

	// Check if we have any files to analyze
	if len(fileMetrics) == 0 {
		return fmt.Errorf("no Go files found to analyze in %s", targetPath)
	}

	// Step 2: Analyze the parsed metrics
	result, err := o.analyzer.Analyze(targetPath, fileMetrics)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Step 3: Report the results
	if err := o.reporter.Report(result); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}
