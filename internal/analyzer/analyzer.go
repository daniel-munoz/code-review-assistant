package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/constants"
	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	parserPkg "github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// Analyzer defines the interface for analyzing parsed code metrics
type Analyzer interface {
	// Analyze takes parsed file metrics and produces an analysis result
	Analyze(projectPath string, metrics []*parserPkg.FileMetrics) (*AnalysisResult, error)
}

// MetricsAnalyzer implements Analyzer for basic metrics analysis
type MetricsAnalyzer struct {
	config         *config.AnalysisConfig
	detectors      *detectors.Registry
	coverageRunner *coverage.Runner
	status         status.Reporter
	projectPath    string
}

// NewAnalyzer creates a new MetricsAnalyzer with the given configuration and status reporter
func NewAnalyzer(cfg *config.AnalysisConfig, statusReporter status.Reporter) Analyzer {
	return &MetricsAnalyzer{
		config:         cfg,
		detectors:      detectors.NewRegistry(cfg),
		coverageRunner: coverage.NewRunner(cfg.CoverageTimeout, statusReporter),
		status:         statusReporter,
	}
}

// Analyze performs comprehensive code quality analysis on parsed file metrics.
//
// This is the main analysis pipeline that orchestrates all analysis phases:
//
// 1. Metrics Aggregation:
//    - Accumulates file and function counts
//    - Detects large files and long functions
//    - Identifies high-complexity functions
//
// 2. Anti-Pattern Detection:
//    - Runs all enabled detectors (parameters, nesting, returns, magic numbers, etc.)
//    - Requires re-parsing to obtain AST for pattern matching
//
// 3. Statistical Analysis:
//    - Calculates aggregate metrics (averages, percentiles)
//    - Identifies outliers (largest files, most complex functions)
//    - Computes overall comment ratio
//
// 4. Coverage Analysis (optional):
//    - Runs go test -cover if enabled
//    - Detects packages below coverage threshold
//
// 5. Dependency Analysis:
//    - Categorizes imports (stdlib, internal, external)
//    - Detects circular dependencies
//    - Identifies packages with too many dependencies
//
// The function is fault-tolerant: coverage and dependency failures produce
// warnings but don't fail the entire analysis.
//
// Returns a complete AnalysisResult with all findings, or an error if the
// core analysis cannot be completed.
func (ma *MetricsAnalyzer) Analyze(projectPath string, metrics []*parserPkg.FileMetrics) (*AnalysisResult, error) {
	result := &AnalysisResult{
		ProjectPath: projectPath,
		TotalFiles:  len(metrics),
		Files:       make([]*FileAnalysis, 0, len(metrics)),
		Issues:      make([]*Issue, 0),
	}

	// Process all files and collect function metrics
	ma.status.Update("[ANALYZE] Processing files...")
	allFunctions := ma.processAllFiles(result, metrics)

	// Calculate aggregate metrics
	ma.status.Update("[ANALYZE] Calculating metrics...")
	result.Metrics = calculateAggregateMetrics(metrics, allFunctions)

	// Check overall code quality
	ma.checkOverallQuality(result)

	// Run additional analyses (coverage and dependencies)
	ma.runCoverageIfEnabled(projectPath, result)
	ma.runDependencyAnalysis(projectPath, metrics, result)

	return result, nil
}

// processAllFiles processes each file, collecting metrics and issues
func (ma *MetricsAnalyzer) processAllFiles(result *AnalysisResult, metrics []*parserPkg.FileMetrics) []*parserPkg.FunctionMetrics {
	var allFunctions []*parserPkg.FunctionMetrics

	for _, fileMetrics := range metrics {
		// Update totals
		result.TotalLines += fileMetrics.TotalLines
		result.TotalCodeLines += fileMetrics.CodeLines
		result.TotalFunctions += len(fileMetrics.Functions)

		// Process file
		fileAnalysis := ma.createFileAnalysis(fileMetrics)
		result.Files = append(result.Files, fileAnalysis)

		// Check file-level issues
		ma.checkFileIssues(result, fileMetrics, fileAnalysis)

		// Check function-level issues and collect functions
		for _, fn := range fileMetrics.Functions {
			allFunctions = append(allFunctions, fn)
			ma.checkFunctionIssues(result, fileMetrics, fn)
		}

		// Run anti-pattern detectors
		detectorIssues := ma.runDetectors(fileMetrics)
		result.Issues = append(result.Issues, detectorIssues...)
	}

	return allFunctions
}

// createFileAnalysis creates a file analysis record
func (ma *MetricsAnalyzer) createFileAnalysis(fileMetrics *parserPkg.FileMetrics) *FileAnalysis {
	return &FileAnalysis{
		Path:      fileMetrics.FilePath,
		Metrics:   fileMetrics,
		LargeFile: fileMetrics.TotalLines > ma.config.LargeFileThreshold,
	}
}

// checkFileIssues checks for file-level issues like large files
func (ma *MetricsAnalyzer) checkFileIssues(result *AnalysisResult, fileMetrics *parserPkg.FileMetrics, fileAnalysis *FileAnalysis) {
	if fileAnalysis.LargeFile {
		result.Issues = append(result.Issues, &Issue{
			Severity:  "warning",
			Type:      "large_file",
			File:      fileMetrics.FilePath,
			Line:      0,
			Message:   "File exceeds size threshold",
			Value:     fileMetrics.TotalLines,
			Threshold: ma.config.LargeFileThreshold,
		})
	}
}

// checkFunctionIssues checks for function-level issues
func (ma *MetricsAnalyzer) checkFunctionIssues(result *AnalysisResult, fileMetrics *parserPkg.FileMetrics, fn *parserPkg.FunctionMetrics) {
	if fn.Lines > ma.config.LongFunctionThreshold {
		result.Issues = append(result.Issues, &Issue{
			Severity:  "warning",
			Type:      "long_function",
			File:      fileMetrics.FilePath,
			Line:      fn.StartLine,
			Function:  fn.FullName(),
			Message:   "Function exceeds length threshold",
			Value:     fn.Lines,
			Threshold: ma.config.LongFunctionThreshold,
		})
	}

	if fn.Complexity > ma.config.ComplexityThreshold {
		result.Issues = append(result.Issues, &Issue{
			Severity:  "warning",
			Type:      "high_complexity",
			File:      fileMetrics.FilePath,
			Line:      fn.StartLine,
			Function:  fn.FullName(),
			Message:   "Function has high cyclomatic complexity",
			Value:     fn.Complexity,
			Threshold: ma.config.ComplexityThreshold,
		})
	}
}

// checkOverallQuality checks overall code quality metrics
func (ma *MetricsAnalyzer) checkOverallQuality(result *AnalysisResult) {
	if result.Metrics.CommentRatio < ma.config.MinCommentRatio {
		result.Issues = append(result.Issues, &Issue{
			Severity:  "info",
			Type:      "low_comment_ratio",
			File:      "",
			Line:      0,
			Message:   "Overall comment ratio is below recommended threshold",
			Value:     int(result.Metrics.CommentRatio * constants.PercentageMultiplier),
			Threshold: int(ma.config.MinCommentRatio * constants.PercentageMultiplier),
		})
	}
}

// runCoverageIfEnabled runs coverage analysis if enabled in config
func (ma *MetricsAnalyzer) runCoverageIfEnabled(projectPath string, result *AnalysisResult) {
	if !ma.config.EnableCoverage {
		return
	}

	coverageResults, err := ma.coverageRunner.RunCoverage(projectPath, ma.config.ExcludePatterns)
	if err != nil {
		fmt.Printf("Warning: Coverage analysis failed: %v\n", err)
		return
	}

	result.Coverage = ma.analyzeCoverage(coverageResults, result)
}

// runDependencyAnalysis runs dependency analysis if possible
func (ma *MetricsAnalyzer) runDependencyAnalysis(projectPath string, metrics []*parserPkg.FileMetrics, result *AnalysisResult) {
	ma.status.Update("[ANALYZE] Analyzing dependencies...")
	depAnalyzer, err := dependencies.NewAnalyzer(projectPath)
	if err != nil {
		fmt.Printf("Warning: Dependency analysis failed: %v\n", err)
		return
	}

	result.Dependencies = ma.analyzeDependencies(depAnalyzer, metrics, result)
}

// runDetectors re-parses file and runs anti-pattern detectors on all functions
func (ma *MetricsAnalyzer) runDetectors(file *parserPkg.FileMetrics) []*Issue {
	var issues []*Issue

	// Re-parse file to get AST
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file.FilePath, nil, parser.ParseComments)
	if err != nil {
		// Skip detector analysis if parse fails
		return nil
	}

	// Run detectors on each function
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Find matching function metrics
		for _, fn := range file.Functions {
			// Match by name and start line
			if funcDecl.Name.Name == fn.Name && fset.Position(funcDecl.Pos()).Line == fn.StartLine {
				detectorIssues := ma.detectors.RunAll(file, fn, fset, funcDecl)
				issues = append(issues, detectorIssues...)
				break
			}
		}
	}

	return issues
}

// analyzeCoverage processes coverage results and creates a coverage report
func (ma *MetricsAnalyzer) analyzeCoverage(coverageResults []*coverage.PackageCoverage, result *AnalysisResult) *CoverageReport {
	report := &CoverageReport{
		Packages:         make([]*PackageCoverage, 0, len(coverageResults)),
		AverageCoverage:  0,
		LowCoverageCount: 0,
	}

	var totalCoverage float64
	var packageCount int

	for _, pkgCov := range coverageResults {
		// Convert to analyzer PackageCoverage type
		analyzedPkg := &PackageCoverage{
			PackagePath: pkgCov.PackagePath,
			Coverage:    pkgCov.Coverage,
			Error:       pkgCov.Error,
			Skipped:     pkgCov.Skipped,
		}
		report.Packages = append(report.Packages, analyzedPkg)

		// Skip packages with errors or no tests for average calculation
		if pkgCov.Error != "" || pkgCov.Skipped {
			continue
		}

		// Count for average
		totalCoverage += pkgCov.Coverage
		packageCount++

		// Check if below threshold
		if pkgCov.Coverage < ma.config.MinCoverageThreshold {
			report.LowCoverageCount++

			// Create issue for low coverage
			result.Issues = append(result.Issues, &Issue{
				Severity:  "warning",
				Type:      "low_coverage",
				File:      "",
				Line:      0,
				Message:   fmt.Sprintf("Package %s has low test coverage", pkgCov.PackagePath),
				Value:     int(pkgCov.Coverage),
				Threshold: int(ma.config.MinCoverageThreshold),
			})
		}
	}

	// Calculate average coverage
	if packageCount > 0 {
		report.AverageCoverage = totalCoverage / float64(packageCount)
	}

	return report
}

// analyzeDependencies processes dependency analysis and creates a dependency report
func (ma *MetricsAnalyzer) analyzeDependencies(depAnalyzer *dependencies.Analyzer, metrics []*parserPkg.FileMetrics, result *AnalysisResult) *DependencyReport {
	// Analyze dependencies
	packageDeps, err := depAnalyzer.Analyze(metrics)
	if err != nil {
		fmt.Printf("Warning: Failed to analyze dependencies: %v\n", err)
		return nil
	}

	// Detect circular dependencies if enabled
	var circularDeps []*CircularDependency
	if ma.config.DetectCircularDeps {
		depCircular, err := depAnalyzer.DetectCircularDependencies(metrics)
		if err != nil {
			fmt.Printf("Warning: Failed to detect circular dependencies: %v\n", err)
		} else {
			// Convert from dependencies.CircularDependency to analyzer.CircularDependency
			for _, cd := range depCircular {
				circularDeps = append(circularDeps, &CircularDependency{
					Cycle: cd.Cycle,
				})
			}
		}
	}

	report := &DependencyReport{
		Packages:             make([]*PackageDependencies, 0, len(packageDeps)),
		CircularDependencies: circularDeps,
		TotalPackages:        len(packageDeps),
		HighImportCount:      0,
		HighExternalCount:    0,
	}

	// Convert package dependencies and check thresholds
	for _, pkgDep := range packageDeps {
		// Convert to analyzer PackageDependencies type
		analyzedPkg := &PackageDependencies{
			PackageName:         pkgDep.PackageName,
			StdlibImports:       pkgDep.StdlibImports,
			InternalImports:     pkgDep.InternalImports,
			ExternalImports:     pkgDep.ExternalImports,
			TotalImports:        pkgDep.TotalImports,
			ExternalImportCount: pkgDep.ExternalImportCount,
		}
		report.Packages = append(report.Packages, analyzedPkg)

		// Check if total imports exceeds threshold
		if pkgDep.TotalImports > ma.config.MaxImports {
			report.HighImportCount++

			// Create issue for too many imports
			result.Issues = append(result.Issues, &Issue{
				Severity:  "warning",
				Type:      "too_many_imports",
				File:      "",
				Line:      0,
				Message:   fmt.Sprintf("Package %s has too many imports", pkgDep.PackageName),
				Value:     pkgDep.TotalImports,
				Threshold: ma.config.MaxImports,
			})
		}

		// Check if external dependencies exceed threshold
		if pkgDep.ExternalImportCount > ma.config.MaxExternalDependencies {
			report.HighExternalCount++

			// Create issue for too many external dependencies
			result.Issues = append(result.Issues, &Issue{
				Severity:  "info",
				Type:      "too_many_external_deps",
				File:      "",
				Line:      0,
				Message:   fmt.Sprintf("Package %s has too many external dependencies", pkgDep.PackageName),
				Value:     pkgDep.ExternalImportCount,
				Threshold: ma.config.MaxExternalDependencies,
			})
		}
	}

	// Create issues for circular dependencies
	for _, cd := range circularDeps {
		result.Issues = append(result.Issues, &Issue{
			Severity: "error",
			Type:     "circular_dependency",
			File:     "",
			Line:     0,
			Message:  fmt.Sprintf("Circular dependency detected: %s", formatCycle(cd.Cycle)),
		})
	}

	return report
}

// formatCycle returns a human-readable representation of a circular dependency
func formatCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	result := ""
	for i, pkg := range cycle {
		if i > 0 {
			result += " -> "
		}
		result += pkg
	}
	return result
}
