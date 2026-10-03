package parser

import (
	"fmt"
	"path/filepath"

	"github.com/daniel-munoz/code-review-assistant/internal/parallel"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// ParseFiles parses an explicit list of files with p, in parallel.
//
// It's the file-list counterpart of Parser.ParseDirectory, used when the files
// to analyze are already known (diff mode). The caller is responsible for
// extension and exclude filtering. workers follows the usual convention
// (0 = runtime.NumCPU, 1 = sequential). Parse errors are collected and don't
// stop the other files. Result order is not guaranteed.
func ParseFiles(p Parser, paths []string, workers int, statusReporter status.Reporter) ([]*FileMetrics, []error) {
	if len(paths) == 0 {
		return nil, nil
	}

	progress := parallel.NewProgressReporter(statusReporter, "[PARSE] Parsing changed files", len(paths))
	results := parallel.ProcessAll(workers, paths, func(path string) parseResult {
		metrics, err := p.ParseFile(path)
		progress.RecordProgress(filepath.Base(path))
		if err != nil {
			return parseResult{err: fmt.Errorf("error parsing %s: %w", path, err)}
		}
		return parseResult{metrics: metrics}
	})

	var metrics []*FileMetrics
	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
		} else if r.metrics != nil {
			metrics = append(metrics, r.metrics)
		}
	}
	return metrics, errs
}
