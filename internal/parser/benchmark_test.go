package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// BenchmarkParseDirectory benchmarks directory parsing with different worker counts.
// Run with: go test -bench=BenchmarkParseDirectory -benchtime=5s ./internal/parser/
func BenchmarkParseDirectory(b *testing.B) {
	// Find a suitable test directory with Go files
	testDir := "../../testdata/sample"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		// Try the current directory as fallback
		testDir = "."
	}

	reporter := status.NewSilentReporter()
	extensions := []string{".go"}
	excludePatterns := []string{"**/testdata/**", "**/*_test.go"}

	for _, workers := range []int{1, 2, 4, 8, 0} {
		name := fmt.Sprintf("workers=%d", workers)
		if workers == 0 {
			name = "workers=auto"
		}

		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				p := NewParser(workers)
				_, _ = p.ParseDirectory(testDir, excludePatterns, extensions, reporter)
			}
		})
	}
}

// BenchmarkParseFile benchmarks single file parsing.
func BenchmarkParseFile(b *testing.B) {
	// Find a Go file to parse
	testFile := ""
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".go" && testFile == "" {
			testFile = path
			return filepath.SkipAll
		}
		return nil
	})

	if testFile == "" {
		b.Skip("No Go file found for benchmarking")
	}

	p := NewParser(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.ParseFile(testFile)
	}
}

// BenchmarkParallelVsSequential compares parallel and sequential parsing performance.
func BenchmarkParallelVsSequential(b *testing.B) {
	testDir := "../../testdata/sample"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		testDir = "."
	}

	reporter := status.NewSilentReporter()
	extensions := []string{".go"}
	excludePatterns := []string{"**/testdata/**", "**/*_test.go"}

	b.Run("sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := NewParser(1) // Sequential
			_, _ = p.ParseDirectory(testDir, excludePatterns, extensions, reporter)
		}
	})

	b.Run("parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := NewParser(0) // Auto (parallel)
			_, _ = p.ParseDirectory(testDir, excludePatterns, extensions, reporter)
		}
	})
}
