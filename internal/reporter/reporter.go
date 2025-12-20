package reporter

import (
	"fmt"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
)

// Reporter defines the interface for formatting and outputting analysis results.
//
// Implementations of this interface are responsible for:
//   - Formatting the AnalysisResult into the desired output format
//   - Writing the formatted output to the appropriate destination
//   - Handling output errors gracefully
//
// The analyzer package is output-agnostic; all formatting decisions are
// delegated to Reporter implementations.
type Reporter interface {
	// Report formats and outputs the analysis results.
	//
	// The result parameter contains all analysis data including issues,
	// metrics, coverage, and dependencies. The implementation determines
	// how this data is formatted and where it's sent.
	//
	// Returns an error if output fails (e.g., I/O error, formatting error).
	Report(result *analyzer.AnalysisResult) error
}

// NewReporter creates a new Reporter based on the configured output format.
//
// Currently supported formats:
//   - "console" (default): Human-readable console output with tables
//
// Returns an error if the configured format is not supported.
func NewReporter(cfg *config.OutputConfig) (Reporter, error) {
	switch cfg.Format {
	case "console", "":
		return NewConsoleReporter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", cfg.Format)
	}
}
