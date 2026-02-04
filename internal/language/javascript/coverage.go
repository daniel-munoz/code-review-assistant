// Package javascript provides JavaScript/TypeScript language support for code analysis.
package javascript

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// CoverageRunner executes JavaScript/TypeScript test coverage using Jest, Vitest, or npm test.
type CoverageRunner struct {
	timeout time.Duration
	status  status.Reporter
}

// NewCoverageRunner creates a new JavaScript/TypeScript coverage runner.
func NewCoverageRunner(timeoutSeconds int, statusReporter status.Reporter) *CoverageRunner {
	return &CoverageRunner{
		timeout: time.Duration(timeoutSeconds) * time.Second,
		status:  statusReporter,
	}
}

// RunCoverage executes tests and extracts coverage for the project.
func (r *CoverageRunner) RunCoverage(projectPath string, excludePatterns []string) ([]*coverage.PackageCoverage, error) {
	r.status.Update("[COVERAGE] Detecting test runner...")

	// Detect and run the appropriate test runner
	runner, err := r.detectTestRunner(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect test runner: %w", err)
	}

	r.status.Update(fmt.Sprintf("[COVERAGE] Running tests with %s...", runner))

	// Run tests with coverage
	if err := r.runTestsWithCoverage(projectPath, runner); err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	// Parse coverage results
	r.status.Update("[COVERAGE] Parsing coverage results...")
	results, err := r.parseCoverageResults(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage: %w", err)
	}

	return results, nil
}

// testRunner represents a detected test runner configuration.
type testRunner struct {
	name    string
	command string
	args    []string
}

// detectTestRunner determines which test runner to use.
func (r *CoverageRunner) detectTestRunner(projectPath string) (*testRunner, error) {
	// Check for package.json
	pkgPath := filepath.Join(projectPath, "package.json")
	pkgData, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("no package.json found: %w", err)
	}

	var pkg struct {
		Scripts struct {
			Test         string `json:"test"`
			TestCoverage string `json:"test:coverage"`
		} `json:"scripts"`
		DevDependencies map[string]string `json:"devDependencies"`
		Dependencies    map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(pkgData, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Check for test:coverage script first
	if pkg.Scripts.TestCoverage != "" {
		return &testRunner{
			name:    "npm test:coverage",
			command: "npm",
			args:    []string{"run", "test:coverage"},
		}, nil
	}

	// Check for vitest
	if _, hasVitest := pkg.DevDependencies["vitest"]; hasVitest {
		return &testRunner{
			name:    "vitest",
			command: "npx",
			args:    []string{"vitest", "run", "--coverage", "--reporter=json", "--outputFile=coverage/coverage-summary.json"},
		}, nil
	}

	// Check for jest
	if _, hasJest := pkg.DevDependencies["jest"]; hasJest {
		return &testRunner{
			name:    "jest",
			command: "npx",
			args:    []string{"jest", "--coverage", "--coverageReporters=json-summary", "--silent"},
		}, nil
	}

	// Check for @jest/core (sometimes jest is a transitive dependency)
	if _, hasJestCore := pkg.DevDependencies["@jest/core"]; hasJestCore {
		return &testRunner{
			name:    "jest",
			command: "npx",
			args:    []string{"jest", "--coverage", "--coverageReporters=json-summary", "--silent"},
		}, nil
	}

	// Fall back to npm test if it contains coverage-related keywords
	if pkg.Scripts.Test != "" {
		// Check if test script already includes coverage
		return &testRunner{
			name:    "npm test",
			command: "npm",
			args:    []string{"test", "--", "--coverage", "--coverageReporters=json-summary"},
		}, nil
	}

	return nil, fmt.Errorf("no supported test runner found (jest or vitest required)")
}

// runTestsWithCoverage executes the test runner with coverage enabled.
func (r *CoverageRunner) runTestsWithCoverage(projectPath string, runner *testRunner) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, runner.command, runner.args...)
	cmd.Dir = projectPath
	cmd.Env = append(os.Environ(), "CI=true") // Ensures non-interactive mode

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("test timeout exceeded")
		}
		// Tests might fail but still produce coverage - check for coverage file
		coveragePath := r.findCoverageFile(projectPath)
		if coveragePath == "" {
			return fmt.Errorf("tests failed and no coverage produced: %s", string(output))
		}
		// Coverage file exists, continue parsing
	}

	return nil
}

// findCoverageFile locates the coverage summary JSON file.
func (r *CoverageRunner) findCoverageFile(projectPath string) string {
	// Common coverage file locations
	candidates := []string{
		filepath.Join(projectPath, "coverage", "coverage-summary.json"),
		filepath.Join(projectPath, "coverage", "coverage-final.json"),
		filepath.Join(projectPath, ".coverage", "coverage-summary.json"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// parseCoverageResults reads and parses the coverage JSON file.
func (r *CoverageRunner) parseCoverageResults(projectPath string) ([]*coverage.PackageCoverage, error) {
	coveragePath := r.findCoverageFile(projectPath)
	if coveragePath == "" {
		return nil, fmt.Errorf("coverage file not found")
	}

	data, err := os.ReadFile(coveragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read coverage file: %w", err)
	}

	// Jest/Vitest coverage-summary.json format
	var coverageData map[string]struct {
		Lines struct {
			Total   int     `json:"total"`
			Covered int     `json:"covered"`
			Skipped int     `json:"skipped"`
			Pct     float64 `json:"pct"`
		} `json:"lines"`
		Statements struct {
			Total   int     `json:"total"`
			Covered int     `json:"covered"`
			Skipped int     `json:"skipped"`
			Pct     float64 `json:"pct"`
		} `json:"statements"`
		Functions struct {
			Total   int     `json:"total"`
			Covered int     `json:"covered"`
			Skipped int     `json:"skipped"`
			Pct     float64 `json:"pct"`
		} `json:"functions"`
		Branches struct {
			Total   int     `json:"total"`
			Covered int     `json:"covered"`
			Skipped int     `json:"skipped"`
			Pct     float64 `json:"pct"`
		} `json:"branches"`
	}

	if err := json.Unmarshal(data, &coverageData); err != nil {
		return nil, fmt.Errorf("failed to parse coverage JSON: %w", err)
	}

	var results []*coverage.PackageCoverage

	for filePath, cov := range coverageData {
		// Skip the "total" entry
		if filePath == "total" {
			continue
		}

		// Make path relative to project
		relPath, err := filepath.Rel(projectPath, filePath)
		if err != nil {
			relPath = filePath
		}

		// Use statement coverage as the primary metric (most common)
		coveragePct := cov.Statements.Pct
		if coveragePct == 0 && cov.Lines.Pct > 0 {
			coveragePct = cov.Lines.Pct
		}

		results = append(results, &coverage.PackageCoverage{
			PackagePath: relPath,
			Coverage:    coveragePct,
		})
	}

	// If we have a total, add it as a summary
	if total, ok := coverageData["total"]; ok {
		results = append([]*coverage.PackageCoverage{{
			PackagePath: "(total)",
			Coverage:    total.Statements.Pct,
		}}, results...)
	}

	return results, nil
}
