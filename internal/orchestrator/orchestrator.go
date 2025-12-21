package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/git"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/reporter"
	"github.com/daniel-munoz/code-review-assistant/internal/storage"
	"github.com/google/uuid"
)

// Orchestrator coordinates the complete code analysis pipeline.
//
// The orchestrator implements the analysis workflow:
//   1. Load: Load previous report for comparison (if enabled)
//   2. Parse: Discover and parse all Go source files
//   3. Analyze: Apply metrics and quality checks
//   4. Compare: Compare with previous report (if enabled)
//   5. Report: Format and output results (including comparison)
//   6. Save: Store report for historical tracking (if enabled)
//
// Each stage is delegated to specialized components (Parser, Analyzer, Reporter,
// Storage, Comparator) that are configured and initialized based on the provided Config.
type Orchestrator struct {
	config     *config.Config
	parser     parser.Parser
	analyzer   analyzer.Analyzer
	reporter   reporter.Reporter
	storage    storage.Storage    // Phase 3: Persistent storage (optional)
	comparator *comparison.Comparator // Phase 3: Historical comparison (optional)
}

// New creates a new Orchestrator with the given configuration.
//
// This function initializes all pipeline components:
//   - Parser: for AST-based metrics extraction
//   - Analyzer: for applying thresholds and detecting issues
//   - Reporter: for formatted output (based on config.Output.Format)
//   - Storage: for persistent storage (if enabled in Phase 3)
//   - Comparator: for historical comparison (if enabled in Phase 3)
//
// Returns an error if component creation fails.
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

	// Phase 3: Create storage if enabled
	var store storage.Storage
	if cfg.Storage.Enabled {
		store, err = createStorage(&cfg.Storage)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}
	}

	// Phase 3: Create comparator if enabled
	var comp *comparison.Comparator
	if cfg.Comparison.Enabled {
		comp = comparison.NewComparator(cfg.Comparison.StableThreshold)
	}

	return &Orchestrator{
		config:     cfg,
		parser:     p,
		analyzer:   a,
		reporter:   r,
		storage:    store,
		comparator: comp,
	}, nil
}

// createStorage creates a storage backend based on configuration.
func createStorage(cfg *config.StorageConfig) (storage.Storage, error) {
	// Determine storage path
	storagePath := cfg.Path
	if storagePath == "" {
		// Use default path based on project mode
		if cfg.ProjectMode {
			storagePath = "./.cra"
		} else {
			storagePath = "~/.cra"
		}
	}

	// Create backend based on type
	switch cfg.Backend {
	case "file":
		return storage.NewFileStorage(storagePath)
	case "sqlite":
		// For SQLite, append database filename
		return storage.NewSQLiteStorage(storagePath + "/history.db")
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.Backend)
	}
}

// Run executes the complete analysis pipeline on the specified directory.
//
// The pipeline workflow:
//
//  1. Load Stage (Phase 3):
//     - Load previous report for comparison (if enabled)
//
//  2. Parse Stage:
//     - Recursively discovers all .go files in targetPath
//     - Applies exclude patterns (vendor/, testdata/, etc.)
//     - Parses each file to extract metrics
//     - Reports parse errors but continues with successful files
//
//  3. Analyze Stage:
//     - Applies thresholds to detect issues
//     - Runs anti-pattern detectors
//     - Executes test coverage analysis (if enabled)
//     - Performs dependency analysis
//
//  4. Compare Stage (Phase 3):
//     - Compare current results with previous report (if enabled)
//     - Calculate deltas and detect trends
//     - Categorize new and fixed issues
//
//  5. Report Stage:
//     - Formats results according to configured output format
//     - Includes comparison results if available
//     - Outputs to console or other configured destination
//
//  6. Save Stage (Phase 3):
//     - Save report to storage for historical tracking (if enabled)
//
// Returns an error if:
//   - No Go files are found in targetPath
//   - Analysis fails (unexpected internal error)
//   - Reporting fails (output error)
//   - Storage operations fail
//
// Parse errors for individual files are reported as warnings but don't fail
// the overall pipeline, allowing partial analysis when some files are malformed.
func (o *Orchestrator) Run(targetPath string) error {
	ctx := context.Background()

	// Phase 3 - Step 1: Load previous report for comparison
	var previousReport *analyzer.AnalysisResult
	var previousTimestamp time.Time
	if o.comparator != nil && o.storage != nil {
		prev, err := o.storage.GetLatest(ctx, targetPath)
		if err == nil && prev != nil {
			previousReport = prev.Result
			previousTimestamp = prev.Timestamp
		}
		// Ignore errors - no previous report is acceptable
	}

	// Step 2: Parse all Go files
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

	// Step 3: Analyze the parsed metrics
	result, err := o.analyzer.Analyze(targetPath, fileMetrics)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Phase 3 - Step 4: Run comparison if enabled
	var comparisonResult *comparison.ComparisonResult
	if o.comparator != nil && previousReport != nil {
		comparisonResult = o.comparator.Compare(result, previousReport, previousTimestamp)
	}

	// Step 5: Report the results (including comparison if available)
	if err := o.reporter.Report(result, comparisonResult); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Phase 3 - Step 6: Save report to storage
	if o.storage != nil {
		if err := o.saveReport(ctx, targetPath, result); err != nil {
			// Log error but don't fail the pipeline
			fmt.Printf("Warning: failed to save report: %v\n", err)
		}
	}

	return nil
}

// saveReport saves the analysis report to storage with metadata.
func (o *Orchestrator) saveReport(ctx context.Context, targetPath string, result *analyzer.AnalysisResult) error {
	// Extract Git metadata
	gitInfo := git.ExtractInfo(targetPath)

	// Create stored report
	report := &storage.StoredReport{
		ID:          uuid.New().String(),
		ProjectPath: targetPath,
		Timestamp:   time.Now(),
		GitCommit:   gitInfo.Commit,
		GitBranch:   gitInfo.Branch,
		Result:      result,
	}

	// Save to storage
	return o.storage.Save(ctx, report)
}

// Close cleans up resources used by the orchestrator.
//
// This should be called when the orchestrator is no longer needed,
// particularly to properly close storage backends.
func (o *Orchestrator) Close() error {
	if o.storage != nil {
		return o.storage.Close()
	}
	return nil
}
