package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/git"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// runDiff analyzes only the files changed according to the configured diff
// mode. Diff runs are never saved or compared.
func (o *Orchestrator) runDiff(targetPath string) error {
	diff := o.config.Analysis.Diff
	changes, err := git.ChangedFiles(targetPath, git.Spec{Mode: git.Mode(diff.Mode), Ref: diff.Ref})
	if err != nil {
		return err
	}
	o.warnPartiallyStaged(changes.PartiallyStaged)
	if o.config.Storage.Enabled || o.config.Comparison.Enabled {
		fmt.Fprintln(o.errOut(), "Note: diff runs are not saved or compared (storage/comparison skipped)")
	}

	scope := &analyzer.Scope{Mode: diff.Mode, Ref: diff.Ref, Base: changes.Base, Files: []string{}}
	candidates := o.filterChanged(targetPath, changes.Files)

	o.status.Start()
	defer o.status.Stop()

	if len(candidates) == 0 {
		o.status.Clear()
		return o.reporter.Report(emptyResult(targetPath, scope), nil)
	}

	o.status.Update(fmt.Sprintf("[PARSE] Parsing %d changed %s files...", len(candidates), o.lang.DisplayName()))
	changed, parseErrors := o.parseChanged(targetPath, candidates)
	o.reportParseErrors(parseErrors)

	for _, m := range changed {
		scope.Files = append(scope.Files, m.FilePath)
	}
	sort.Strings(scope.Files)

	if len(changed) == 0 {
		o.status.Clear()
		return o.reporter.Report(emptyResult(targetPath, scope), nil)
	}

	result, err := o.analyzer.Analyze(targetPath, changed)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}
	result.Scope = scope

	o.status.Clear()
	if err := o.reporter.Report(result, nil); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	return nil
}

// filterChanged joins git's target-relative paths onto targetPath (the same
// form a directory walk produces) and keeps the files this language analyzes:
// matching extension, not excluded, and still present on disk.
func (o *Orchestrator) filterChanged(targetPath string, rel []string) []string {
	exts := o.lang.Extensions()
	out := make([]string, 0, len(rel))
	for _, r := range rel {
		p := filepath.Join(targetPath, r)
		if !parser.HasMatchingExtension(p, exts) || parser.ShouldExclude(p, targetPath, o.config.Analysis.ExcludePatterns) {
			continue
		}
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			continue // e.g. staged edit, then deleted from the working tree
		}
		out = append(out, p)
	}
	return out
}

// parseChanged parses the candidate files. With --with-deps it parses the
// whole project (kept in o.fullMetrics for dependency analysis) and returns the
// changed subset, so detectors still only see changed files.
func (o *Orchestrator) parseChanged(targetPath string, candidates []string) ([]*parser.FileMetrics, []error) {
	if !o.config.Analysis.Diff.WithDeps {
		return parser.ParseFiles(o.parser, candidates, o.config.Analysis.Workers, o.status)
	}

	all, errs := o.parseAndValidate(targetPath)
	o.fullMetrics = all

	want := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		want[filepath.Clean(c)] = true
	}
	var changed []*parser.FileMetrics
	for _, m := range all {
		if want[filepath.Clean(m.FilePath)] {
			changed = append(changed, m)
		}
	}
	return changed, errs
}

// warnPartiallyStaged tells the user that --staged reads the working tree.
func (o *Orchestrator) warnPartiallyStaged(files []string) {
	if len(files) == 0 {
		return
	}
	fmt.Fprintf(o.errOut(), "Warning: %d staged file(s) also have unstaged changes; results reflect the working tree:\n", len(files))
	for _, f := range files {
		fmt.Fprintf(o.errOut(), "  - %s\n", f)
	}
}

// emptyResult is the report for a diff run with nothing to analyze.
func emptyResult(targetPath string, scope *analyzer.Scope) *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		ProjectPath: targetPath,
		Metrics:     &analyzer.AggregateMetrics{LargestFiles: []*analyzer.FileSize{}},
		Files:       []*analyzer.FileAnalysis{},
		Issues:      []*analyzer.Issue{},
		Scope:       scope,
	}
}

// depFactory returns the dependency-analyzer factory for this run, or nil
// when dependency analysis is skipped (diff mode without --with-deps).
func (o *Orchestrator) depFactory() analyzer.DependencyAnalyzerFactory {
	diff := o.config.Analysis.Diff
	if diff.Enabled() && !diff.WithDeps {
		return nil
	}
	return func(projectPath string) (analyzer.DependencyAnalyzer, error) {
		da, err := o.lang.DependencyAnalyzer(projectPath)
		if err != nil || da == nil {
			return nil, err
		}
		if diff.Enabled() {
			return &fullProjectDeps{inner: da, files: o.fullMetrics}, nil
		}
		return da, nil
	}
}

// fullProjectDeps runs dependency analysis over the whole project even though
// the analyzer only sees the changed files (diff mode with --with-deps), so
// the import graph and cycle detection are complete.
type fullProjectDeps struct {
	inner language.DependencyAnalyzer
	files []*parser.FileMetrics
}

func (d *fullProjectDeps) Analyze(_ []*parser.FileMetrics) ([]*dependencies.PackageDependencies, error) {
	return d.inner.Analyze(d.files)
}

func (d *fullProjectDeps) DetectCircularDependencies(_ []*parser.FileMetrics) ([]*dependencies.CircularDependency, error) {
	return d.inner.DetectCircularDependencies(d.files)
}
