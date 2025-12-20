// Package coverage integrates Go test coverage analysis.
//
// The coverage package runs `go test -cover` on packages and parses the
// output to extract coverage percentages. It supports:
// - Configurable timeouts per package
// - Automatic exclusion of testdata and vendor packages
// - Coverage threshold detection
//
// # Usage
//
// Create a coverage runner with timeout:
//
//	runner := coverage.NewRunner(30) // 30 second timeout
//
// Run coverage analysis:
//
//	results, err := runner.RunCoverage("/path/to/project", excludePatterns)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, result := range results {
//	    if result.Skipped {
//	        fmt.Printf("Skipped: %s (no tests)\n", result.PackagePath)
//	    } else if result.Error != "" {
//	        fmt.Printf("Error: %s - %s\n", result.PackagePath, result.Error)
//	    } else {
//	        fmt.Printf("%s: %.1f%%\n", result.PackagePath, result.Coverage)
//	    }
//	}
//
// The runner automatically skips packages without tests and handles
// compilation errors gracefully.
package coverage
