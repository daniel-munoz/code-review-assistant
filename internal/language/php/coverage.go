// Package php provides PHP language support for code analysis.
package php

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// CoverageRunner executes PHP test coverage using PHPUnit with Clover XML output.
type CoverageRunner struct {
	timeout time.Duration
	status  status.Reporter
}

// NewCoverageRunner creates a new PHP coverage runner.
func NewCoverageRunner(timeoutSeconds int, statusReporter status.Reporter) *CoverageRunner {
	return &CoverageRunner{
		timeout: time.Duration(timeoutSeconds) * time.Second,
		status:  statusReporter,
	}
}

// RunCoverage executes PHPUnit tests and extracts coverage from Clover XML output.
func (r *CoverageRunner) RunCoverage(projectPath string, excludePatterns []string) ([]*coverage.PackageCoverage, error) {
	r.status.Update("[COVERAGE] Detecting PHP test runner...")

	// Detect test runner (PHPUnit, Pest, etc.)
	runner, err := r.detectTestRunner(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect test runner: %w", err)
	}

	r.status.Update(fmt.Sprintf("[COVERAGE] Running tests with %s...", runner.name))

	// Create a temp file for coverage output
	coveragePath := filepath.Join(projectPath, "coverage-clover.xml")

	// Run tests with coverage
	if err := r.runTestsWithCoverage(projectPath, runner, coveragePath); err != nil {
		// Clean up temp file on failure
		os.Remove(coveragePath)
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	// Parse coverage results
	r.status.Update("[COVERAGE] Parsing coverage results...")
	results, err := r.parseCoverageResults(projectPath, coveragePath)

	// Clean up generated coverage file
	os.Remove(coveragePath)

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
	workDir string // directory to run from (defaults to projectPath if empty)
}

// detectTestRunner determines which PHP test runner to use.
// It searches the projectPath and its immediate subdirectories to support monorepos.
func (r *CoverageRunner) detectTestRunner(projectPath string) (*testRunner, error) {
	// Collect candidate directories: the project root plus immediate subdirectories
	// that contain a composer.json (monorepo packages)
	candidates := r.findCandidateDirs(projectPath)

	// Check for PHPUnit/Pest binaries across all candidate directories
	for _, dir := range candidates {
		for _, binDir := range r.vendorBinDirs(dir) {
			phpunitPath := filepath.Join(binDir, "phpunit")
			if _, err := os.Stat(phpunitPath); err == nil {
				runner := &testRunner{
					name:    "phpunit",
					command: phpunitPath,
				}
				r.resolveConfigAndWorkDir(dir, runner)
				return runner, nil
			}

			pestPath := filepath.Join(binDir, "pest")
			if _, err := os.Stat(pestPath); err == nil {
				runner := &testRunner{
					name:    "pest",
					command: pestPath,
				}
				r.resolveConfigAndWorkDir(dir, runner)
				return runner, nil
			}
		}
	}

	// Check for phpunit.phar at the project root
	pharPath := filepath.Join(projectPath, "phpunit.phar")
	if _, err := os.Stat(pharPath); err == nil {
		return &testRunner{
			name:    "phpunit",
			command: pharPath,
			args:    []string{},
		}, nil
	}

	// Check for Composer test script across all candidate directories
	for _, dir := range candidates {
		if runner := r.detectComposerTestScript(dir); runner != nil {
			return runner, nil
		}
	}

	// Check if phpunit is globally available
	if phpunitPath, err := exec.LookPath("phpunit"); err == nil {
		return &testRunner{
			name:    "phpunit",
			command: phpunitPath,
			args:    []string{},
		}, nil
	}

	// No binary found — provide a helpful error based on what we can detect
	return nil, r.buildDetectionError(projectPath, candidates)
}

// findCandidateDirs returns the project root and immediate subdirectories that
// contain a composer.json file. This supports monorepo layouts where PHP packages
// live in subdirectories (e.g., api/, packages/foo/).
func (r *CoverageRunner) findCandidateDirs(projectPath string) []string {
	candidates := []string{projectPath}

	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return candidates
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := filepath.Join(projectPath, entry.Name())
		if _, err := os.Stat(filepath.Join(subdir, "composer.json")); err == nil {
			candidates = append(candidates, subdir)
		}
		// Also check one level deeper (e.g., packages/platform-common/)
		subEntries, err := os.ReadDir(subdir)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}
			deepDir := filepath.Join(subdir, subEntry.Name())
			if _, err := os.Stat(filepath.Join(deepDir, "composer.json")); err == nil {
				candidates = append(candidates, deepDir)
			}
		}
	}

	return candidates
}

// vendorBinDirs returns candidate bin directories for a given project directory.
// It checks composer.json for a custom vendor-dir setting and always includes the default "vendor/bin".
func (r *CoverageRunner) vendorBinDirs(dir string) []string {
	dirs := []string{filepath.Join(dir, "vendor", "bin")}

	composerFile := filepath.Join(dir, "composer.json")
	data, err := os.ReadFile(composerFile)
	if err != nil {
		return dirs
	}

	var composer struct {
		Config struct {
			VendorDir string `json:"vendor-dir"`
		} `json:"config"`
	}
	if json.Unmarshal(data, &composer) == nil && composer.Config.VendorDir != "" && composer.Config.VendorDir != "vendor" {
		dirs = append([]string{filepath.Join(dir, composer.Config.VendorDir, "bin")}, dirs...)
	}

	return dirs
}

// resolveConfigAndWorkDir finds the PHPUnit/Pest config file near the candidate directory
// and sets workDir and -c args on the runner so the test framework finds its configuration.
func (r *CoverageRunner) resolveConfigAndWorkDir(candidateDir string, runner *testRunner) {
	configNames := []string{"phpunit.xml", "phpunit.xml.dist", "phpunit.dist.xml"}
	if runner.name == "pest" {
		configNames = []string{"phpunit.xml", "phpunit.xml.dist", "phpunit.dist.xml"}
	}

	// Check the candidate directory itself
	for _, cf := range configNames {
		configPath := filepath.Join(candidateDir, cf)
		if _, err := os.Stat(configPath); err == nil {
			runner.workDir = candidateDir
			runner.args = []string{"-c", configPath}
			return
		}
	}

	// Check one level of subdirectories (e.g., api/tests/phpunit.xml)
	entries, err := os.ReadDir(candidateDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		for _, cf := range configNames {
			configPath := filepath.Join(candidateDir, entry.Name(), cf)
			if _, err := os.Stat(configPath); err == nil {
				runner.workDir = candidateDir
				runner.args = []string{"-c", configPath}
				return
			}
		}
	}

	// No config found — just set workDir to the candidate directory
	runner.workDir = candidateDir
}

// detectComposerTestScript checks a directory's composer.json for a "test" script that runs PHPUnit.
func (r *CoverageRunner) detectComposerTestScript(dir string) *testRunner {
	composerFile := filepath.Join(dir, "composer.json")
	data, err := os.ReadFile(composerFile)
	if err != nil {
		return nil
	}

	var composer struct {
		Scripts map[string]any `json:"scripts"`
	}
	if err := json.Unmarshal(data, &composer); err != nil {
		return nil
	}

	// Check if "test" script exists and references phpunit
	if testScript, ok := composer.Scripts["test"]; ok {
		scriptStr, isString := testScript.(string)
		if isString && (strings.Contains(scriptStr, "phpunit") || strings.Contains(scriptStr, "pest")) {
			composerBin, err := exec.LookPath("composer")
			if err != nil {
				return nil
			}
			return &testRunner{
				name:    "composer test",
				command: composerBin,
				args:    []string{"test", "--"},
			}
		}
	}

	return nil
}

// buildDetectionError produces a helpful error message based on what's present in the project.
func (r *CoverageRunner) buildDetectionError(projectPath string, candidates []string) error {
	// Check for PHPUnit config files across all candidate directories and their subdirectories
	configNames := []string{"phpunit.xml", "phpunit.xml.dist", "phpunit.dist.xml"}
	for _, dir := range candidates {
		for _, cf := range configNames {
			// Check at the directory root
			if _, err := os.Stat(filepath.Join(dir, cf)); err == nil {
				relDir, _ := filepath.Rel(projectPath, dir)
				if relDir == "." {
					return fmt.Errorf("PHPUnit config found (%s) but phpunit binary not found in vendor/bin/phpunit; run 'composer install' first", cf)
				}
				return fmt.Errorf("PHPUnit config found (%s in %s) but vendor/bin/phpunit not found; run 'composer install' in %s first", cf, relDir, relDir)
			}
			// Check one level deep (e.g., api/tests/phpunit.xml)
			subEntries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range subEntries {
				if !entry.IsDir() {
					continue
				}
				if _, err := os.Stat(filepath.Join(dir, entry.Name(), cf)); err == nil {
					relDir, _ := filepath.Rel(projectPath, dir)
					if relDir == "." {
						return fmt.Errorf("PHPUnit config found (%s/%s) but phpunit binary not found in vendor/bin/phpunit; run 'composer install' first", entry.Name(), cf)
					}
					return fmt.Errorf("PHPUnit config found (%s/%s/%s) but vendor/bin/phpunit not found; run 'composer install' in %s first", relDir, entry.Name(), cf, relDir)
				}
			}
		}
	}

	// Check if any composer.json lists phpunit as a dependency
	for _, dir := range candidates {
		composerFile := filepath.Join(dir, "composer.json")
		data, err := os.ReadFile(composerFile)
		if err != nil {
			continue
		}

		var composer struct {
			RequireDev map[string]string `json:"require-dev"`
		}
		if json.Unmarshal(data, &composer) != nil {
			continue
		}

		relDir, _ := filepath.Rel(projectPath, dir)
		location := ""
		if relDir != "." {
			location = " (in " + relDir + ")"
		}

		if _, ok := composer.RequireDev["phpunit/phpunit"]; ok {
			return fmt.Errorf("PHPUnit is listed in composer.json%s require-dev but vendor/bin/phpunit not found; run 'composer install' first", location)
		}
		if _, ok := composer.RequireDev["pestphp/pest"]; ok {
			return fmt.Errorf("Pest is listed in composer.json%s require-dev but vendor/bin/pest not found; run 'composer install' first", location)
		}
	}

	return fmt.Errorf("no supported PHP test runner found (PHPUnit or Pest required)")
}

// runTestsWithCoverage executes the test runner with Clover XML coverage output.
func (r *CoverageRunner) runTestsWithCoverage(projectPath string, runner *testRunner, coveragePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	args := append(runner.args, "--coverage-clover", coveragePath)
	cmd := exec.CommandContext(ctx, runner.command, args...)
	if runner.workDir != "" {
		cmd.Dir = runner.workDir
	} else {
		cmd.Dir = projectPath
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("test timeout exceeded")
		}
		// Tests might fail but still produce coverage - check for coverage file
		if _, statErr := os.Stat(coveragePath); statErr != nil {
			return fmt.Errorf("tests failed and no coverage produced: %s", string(output))
		}
		// Coverage file exists, continue parsing
	}

	return nil
}

// cloverCoverage represents the Clover XML coverage report structure.
type cloverCoverage struct {
	XMLName xml.Name      `xml:"coverage"`
	Project cloverProject `xml:"project"`
}

// cloverProject represents the project-level Clover data.
type cloverProject struct {
	Metrics  cloverMetrics   `xml:"metrics"`
	Packages []cloverPackage `xml:"package"`
	Files    []cloverFile    `xml:"file"` // Files outside packages
}

// cloverPackage represents a package in the Clover report.
type cloverPackage struct {
	Name    string         `xml:"name,attr"`
	Metrics cloverMetrics  `xml:"metrics"`
	Files   []cloverFile   `xml:"file"`
}

// cloverFile represents a file in the Clover report.
type cloverFile struct {
	Name    string        `xml:"name,attr"`
	Metrics cloverMetrics `xml:"metrics"`
}

// cloverMetrics represents coverage metrics in the Clover report.
type cloverMetrics struct {
	Statements        int `xml:"statements,attr"`
	CoveredStatements int `xml:"coveredstatements,attr"`
	Methods           int `xml:"methods,attr"`
	CoveredMethods    int `xml:"coveredmethods,attr"`
	Conditionals      int `xml:"conditionals,attr"`
	CoveredConditionals int `xml:"coveredconditionals,attr"`
	Elements          int `xml:"elements,attr"`
	CoveredElements   int `xml:"coveredelements,attr"`
}

// coveragePercent calculates the coverage percentage from metrics.
func (m *cloverMetrics) coveragePercent() float64 {
	total := m.Statements + m.Methods + m.Conditionals
	if total == 0 {
		return 0
	}
	covered := m.CoveredStatements + m.CoveredMethods + m.CoveredConditionals
	return float64(covered) / float64(total) * 100
}

// parseCoverageResults reads and parses the Clover XML coverage file.
func (r *CoverageRunner) parseCoverageResults(projectPath string, coveragePath string) ([]*coverage.PackageCoverage, error) {
	data, err := os.ReadFile(coveragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read coverage file: %w", err)
	}

	var cov cloverCoverage
	if err := xml.Unmarshal(data, &cov); err != nil {
		return nil, fmt.Errorf("failed to parse Clover XML: %w", err)
	}

	var results []*coverage.PackageCoverage

	// Process packages
	for _, pkg := range cov.Project.Packages {
		pct := pkg.Metrics.coveragePercent()
		pkgName := pkg.Name
		if pkgName == "" {
			pkgName = "(default)"
		}

		results = append(results, &coverage.PackageCoverage{
			PackagePath: pkgName,
			Coverage:    pct,
		})
	}

	// Process files outside packages
	for _, file := range cov.Project.Files {
		pct := file.Metrics.coveragePercent()
		relPath, relErr := filepath.Rel(projectPath, file.Name)
		if relErr != nil {
			relPath = file.Name
		}

		results = append(results, &coverage.PackageCoverage{
			PackagePath: relPath,
			Coverage:    pct,
		})
	}

	// Add project total
	if totalPct := cov.Project.Metrics.coveragePercent(); totalPct > 0 || len(results) > 0 {
		total := &coverage.PackageCoverage{
			PackagePath: "(total)",
			Coverage:    cov.Project.Metrics.coveragePercent(),
		}
		results = append([]*coverage.PackageCoverage{total}, results...)
	}

	return results, nil
}
