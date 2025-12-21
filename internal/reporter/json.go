package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
)

// JSONReporter implements Reporter for JSON output
type JSONReporter struct {
	config *config.OutputConfig
	output io.Writer
}

// JSONOutput wraps the analysis result with metadata for JSON export
type JSONOutput struct {
	Timestamp  string                       `json:"timestamp"`
	Version    string                       `json:"version"`
	Result     *analyzer.AnalysisResult     `json:"result"`
	Comparison *comparison.ComparisonResult `json:"comparison,omitempty"` // Phase 3: Optional comparison
}

// NewJSONReporter creates a new JSONReporter
func NewJSONReporter(cfg *config.OutputConfig) *JSONReporter {
	return &JSONReporter{
		config: cfg,
		output: os.Stdout, // Default to stdout
	}
}

// Report generates a JSON formatted report
func (jr *JSONReporter) Report(result *analyzer.AnalysisResult, comp *comparison.ComparisonResult) error {
	// Sort issues by severity for consistent output
	analyzer.SortIssuesBySeverity(result.Issues)

	// If output file is specified, write to file instead of stdout
	if jr.config.OutputFile != "" {
		file, err := jr.createOutputFile(jr.config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		jr.output = file
	}

	// Wrap result with metadata
	output := JSONOutput{
		Timestamp:  time.Now().Format(time.RFC3339),
		Version:    "0.3.0", // Phase 3 version
		Result:     result,
		Comparison: comp, // Phase 3: Include comparison if available
	}

	// Marshal to JSON
	var data []byte
	var err error

	if jr.config.JSONPretty {
		data, err = json.MarshalIndent(output, "", "  ")
	} else {
		data, err = json.Marshal(output)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to output
	if _, err := jr.output.Write(data); err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}

	// Add newline for better terminal output
	if _, err := jr.output.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

func (jr *JSONReporter) createOutputFile(path string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Create or truncate file
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}
