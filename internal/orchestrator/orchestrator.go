package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/git"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	_ "github.com/daniel-munoz/code-review-assistant/internal/language/golang"     // Register Go language
	_ "github.com/daniel-munoz/code-review-assistant/internal/language/javascript" // Register JavaScript/TypeScript language
	_ "github.com/daniel-munoz/code-review-assistant/internal/language/python"     // Register Python language
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/reporter"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
	"github.com/daniel-munoz/code-review-assistant/internal/storage"
	"github.com/google/uuid"
)

// Orchestrator coordinates the complete code analysis pipeline.
//
// The orchestrator implements the analysis workflow:
//   1. Load: Load previous report for comparison (if enabled)
//   2. Parse: Discover and parse all source files for the detected language
//   3. Analyze: Apply metrics and quality checks
//   4. Compare: Compare with previous report (if enabled)
//   5. Report: Format and output results (including comparison)
//   6. Save: Store report for historical tracking (if enabled)
//
// Each stage is delegated to specialized components (Parser, Analyzer, Reporter,
// Storage, Comparator) that are configured and initialized based on the provided Config.
type Orchestrator struct {
	config     *config.Config
	lang       language.Language       // Language provider
	parser     parser.Parser
	analyzer   analyzer.Analyzer
	reporter   reporter.Reporter
	storage    storage.Storage         // Phase 3: Persistent storage (optional)
	comparator *comparison.Comparator  // Phase 3: Historical comparison (optional)
	status     status.Reporter         // Live status reporting
}

// New creates a new Orchestrator with the given configuration.
//
// This function initializes all pipeline components:
//   - Language: detects or uses configured language for analysis
//   - Parser: for AST-based metrics extraction
//   - Analyzer: for applying thresholds and detecting issues
//   - Reporter: for formatted output (based on config.Output.Format)
//   - Storage: for persistent storage (if enabled in Phase 3)
//   - Comparator: for historical comparison (if enabled in Phase 3)
//
// The targetPath is used for language auto-detection when config.Language is "auto".
// Returns an error if component creation fails.
func New(cfg *config.Config, targetPath string) (*Orchestrator, error) {
	// Create status reporter (needed by analyzer and parser)
	statusReporter := status.NewReporter(&cfg.Output)

	// Get language provider (detect or use configured)
	lang, err := getLanguage(cfg, targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect language: %w", err)
	}

	// Merge language-specific exclude patterns with config patterns
	langPatterns := lang.DefaultExcludePatterns()
	if len(langPatterns) > 0 {
		cfg.Analysis.ExcludePatterns = mergeExcludePatterns(cfg.Analysis.ExcludePatterns, langPatterns)
	}

	// Get language-specific components
	p := lang.Parser(cfg.Analysis.Workers)
	detectorRunner := lang.DetectorRunner(&cfg.Analysis)
	coverageRunner := lang.CoverageRunner(&cfg.Analysis, statusReporter)

	// Create dependency analyzer factory from language
	depAnalyzerFactory := func(projectPath string) (analyzer.DependencyAnalyzer, error) {
		return lang.DependencyAnalyzer(projectPath)
	}

	// Create analyzer with language-specific components
	a := analyzer.NewAnalyzer(&cfg.Analysis, statusReporter, detectorRunner, coverageRunner, depAnalyzerFactory)

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

	// Inject storage into HTMLReporter for historical trends
	if store != nil {
		if htmlReporter, ok := r.(*reporter.HTMLReporter); ok {
			htmlReporter.WithStorage(store)
		}
	}

	// Phase 3: Create comparator if enabled
	var comp *comparison.Comparator
	if cfg.Comparison.Enabled {
		comp = comparison.NewComparator(cfg.Comparison.StableThreshold)
	}

	return &Orchestrator{
		config:     cfg,
		lang:       lang,
		parser:     p,
		analyzer:   a,
		reporter:   r,
		storage:    store,
		comparator: comp,
		status:     statusReporter,
	}, nil
}

// getLanguage determines the language to use for analysis.
// If cfg.Language is "auto" or empty, it attempts to detect the language from project files.
// Otherwise, it looks up the specified language by name.
func getLanguage(cfg *config.Config, targetPath string) (language.Language, error) {
	langName := cfg.Language
	if langName == "" || langName == "auto" {
		// Auto-detect from project files
		lang, err := language.DetectLanguage(targetPath)
		if err != nil {
			return nil, fmt.Errorf("language auto-detection failed: %w", err)
		}
		return lang, nil
	}

	lang, ok := language.Get(langName)
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s (available: %v)", langName, language.SupportedLanguages())
	}
	return lang, nil
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

	// Start status reporting
	o.status.Start()
	defer o.status.Stop()

	// Stage 1: Load previous report for comparison
	if o.storage != nil && o.comparator != nil {
		o.status.Update("[LOAD] Loading previous report...")
	}
	previousReport, previousTimestamp := o.loadPreviousReport(ctx, targetPath)

	// Stage 2: Parse and validate files
	o.status.Update(fmt.Sprintf("[PARSE] Discovering %s files...", o.lang.DisplayName()))
	fileMetrics, parseErrors := o.parseAndValidate(targetPath)
	o.reportParseErrors(parseErrors)

	if len(fileMetrics) == 0 {
		return fmt.Errorf("no %s files found to analyze in %s", o.lang.DisplayName(), targetPath)
	}

	// Stage 3: Analyze the parsed metrics (status reporting delegated to analyzer)
	result, err := o.analyzer.Analyze(targetPath, fileMetrics)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Stage 4: Run comparison if enabled
	if previousReport != nil {
		o.status.Update("[COMPARE] Comparing with previous report...")
	}
	comparisonResult := o.runComparison(result, previousReport, previousTimestamp)

	// Stage 5: Generate report (clear status before output)
	o.status.Clear()
	if err := o.reporter.Report(result, comparisonResult); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Stage 6: Save report to storage (non-fatal)
	o.saveReportIfEnabled(ctx, targetPath, result)

	return nil
}

// loadPreviousReport loads the most recent report for comparison
func (o *Orchestrator) loadPreviousReport(ctx context.Context, targetPath string) (*analyzer.AnalysisResult, time.Time) {
	if o.comparator == nil || o.storage == nil {
		return nil, time.Time{}
	}

	prev, err := o.storage.GetLatest(ctx, targetPath)
	if err != nil || prev == nil {
		return nil, time.Time{}
	}

	return prev.Result, prev.Timestamp
}

// parseAndValidate parses all source files for the detected language in the target path
func (o *Orchestrator) parseAndValidate(targetPath string) ([]*parser.FileMetrics, []error) {
	extensions := o.lang.Extensions()
	return o.parser.ParseDirectory(targetPath, o.config.Analysis.ExcludePatterns, extensions, o.status)
}

// reportParseErrors reports parsing errors to the user (max 5 errors shown)
func (o *Orchestrator) reportParseErrors(parseErrors []error) {
	if len(parseErrors) == 0 {
		return
	}

	fmt.Printf("Warning: %d files failed to parse:\n", len(parseErrors))
	maxErrors := 5
	for i, err := range parseErrors {
		if i >= maxErrors {
			fmt.Printf("  ... and %d more errors\n", len(parseErrors)-maxErrors)
			break
		}
		fmt.Printf("  - %v\n", err)
	}
	fmt.Println()
}

// runComparison runs comparison if enabled and previous report exists
func (o *Orchestrator) runComparison(result, previousReport *analyzer.AnalysisResult, previousTimestamp time.Time) *comparison.ComparisonResult {
	if o.comparator == nil || previousReport == nil {
		return nil
	}
	return o.comparator.Compare(result, previousReport, previousTimestamp)
}

// saveReportIfEnabled saves the report to storage if storage is enabled
func (o *Orchestrator) saveReportIfEnabled(ctx context.Context, targetPath string, result *analyzer.AnalysisResult) {
	if o.storage == nil {
		return
	}

	if err := o.saveReport(ctx, targetPath, result); err != nil {
		fmt.Printf("Warning: failed to save report: %v\n", err)
	}
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

// mergeExcludePatterns merges language-specific patterns with config patterns,
// avoiding duplicates.
func mergeExcludePatterns(configPatterns, langPatterns []string) []string {
	seen := make(map[string]bool)
	for _, p := range configPatterns {
		seen[p] = true
	}

	result := make([]string, len(configPatterns))
	copy(result, configPatterns)

	for _, p := range langPatterns {
		if !seen[p] {
			result = append(result, p)
			seen[p] = true
		}
	}

	return result
}
