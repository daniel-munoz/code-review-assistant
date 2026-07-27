package coverage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// PackageCoverage represents coverage data for a single package
type PackageCoverage struct {
	PackagePath string  `json:"package_path"`
	Coverage    float64 `json:"coverage"` // Percentage (0-100)
	Error       string  `json:"error"`    // Empty if successful
	Skipped     bool    `json:"skipped"`  // True if no tests found
}

// Runner executes go test -cover for packages
type Runner struct {
	timeout time.Duration
	status  status.Reporter
}

// NewRunner creates a new coverage runner
func NewRunner(timeoutSeconds int, statusReporter status.Reporter) *Runner {
	return &Runner{
		timeout: time.Duration(timeoutSeconds) * time.Second,
		status:  statusReporter,
	}
}

// RunCoverage executes tests and extracts coverage for all packages in projectPath
func (r *Runner) RunCoverage(projectPath string, excludePatterns []string) ([]*PackageCoverage, error) {
	// Find all packages
	packages, err := r.findPackages(projectPath, excludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to find packages: %w", err)
	}

	total := len(packages)
	var results []*PackageCoverage

	// Run coverage for each package with progress reporting
	for i, pkg := range packages {
		r.status.UpdateProgress("[COVERAGE] Running tests", i+1, total, pkg)
		coverage := r.runPackageCoverage(pkg)
		results = append(results, coverage)
	}

	return results, nil
}

// findPackages discovers all Go packages in the project
func (r *Runner) findPackages(projectPath string, excludePatterns []string) ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	cmd.Dir = projectPath
	cmd.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	var packages []string
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		pkg := scanner.Text()

		// Skip excluded packages
		if r.shouldExclude(pkg, excludePatterns) {
			continue
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// shouldExclude checks if package matches exclusion patterns
func (r *Runner) shouldExclude(pkg string, patterns []string) bool {
	// Skip common excludable packages
	if strings.Contains(pkg, "testdata") || strings.Contains(pkg, "vendor") {
		return true
	}

	// Additional pattern matching could be added here
	return false
}

// runPackageCoverage runs tests for a single package and extracts coverage
func (r *Runner) runPackageCoverage(pkg string) *PackageCoverage {
	result := &PackageCoverage{
		PackagePath: pkg,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// Run go test -cover
	cmd := exec.CommandContext(ctx, "go", "test", "-cover", pkg)
	cmd.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = "test timeout exceeded"
			return result
		}

		// Check if there are no tests
		outputStr := string(output)
		if strings.Contains(outputStr, "no test files") ||
			strings.Contains(outputStr, "[no test files]") {
			result.Skipped = true
			result.Coverage = 0
			return result
		}

		// Other error
		result.Error = fmt.Sprintf("test failed: %v", err)
		return result
	}

	// Parse coverage from output
	// Expected format: "coverage: 75.2% of statements"
	coverage, err := r.parseCoverage(string(output))
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Coverage = coverage
	return result
}

// parseCoverage extracts coverage percentage from go test output
func (r *Runner) parseCoverage(output string) (float64, error) {
	// Regex to match "coverage: XX.X% of statements"
	re := regexp.MustCompile(`coverage:\s+([\d.]+)%\s+of\s+statements`)

	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("coverage data not found in output")
	}

	coverage, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse coverage value: %w", err)
	}

	return coverage, nil
}
