package reporter

import (
	"fmt"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
)

// Reporter defines the interface for reporting analysis results
type Reporter interface {
	// Report formats and outputs the analysis results
	Report(result *analyzer.AnalysisResult) error
}

// NewReporter creates a new Reporter based on the output format
func NewReporter(cfg *config.OutputConfig) (Reporter, error) {
	switch cfg.Format {
	case "console", "":
		return NewConsoleReporter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", cfg.Format)
	}
}
