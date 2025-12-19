package orchestrator

import (
	"fmt"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/reporter"
)

// Orchestrator coordinates the analysis pipeline
type Orchestrator struct {
	config   *config.Config
	parser   parser.Parser
	analyzer analyzer.Analyzer
	reporter reporter.Reporter
}

// New creates a new Orchestrator with the given configuration
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

// Run executes the analysis pipeline on the target path
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
