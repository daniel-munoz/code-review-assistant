package cmd

import (
	"errors"
	"fmt"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
)

// Exit codes. CI and git hooks rely on these, so they are a stable contract.
const (
	ExitClean     = 0 // No issues at or above --fail-on (or --fail-on not set)
	ExitWarnings  = 1 // --fail-on=warning and the worst issue is a warning
	ExitBlocking  = 2 // --fail-on is set and at least one error-severity issue was found
	ExitToolError = 3 // CRA itself failed: bad flags, unreadable config, git errors, ...
)

// failOnLevels ranks the severities --fail-on accepts. Info issues never fail a run.
var failOnLevels = map[string]int{"warning": 1, "error": 2}

// ExitError is returned when analysis succeeded but found issues at or above
// the --fail-on level. Code is the process exit code.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

// ExitCode maps the error returned by Execute to a process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitClean
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return ExitToolError
}

// validateFailOn accepts "", "none", "warning" and "error".
func validateFailOn(value string) error {
	if value == "" || value == "none" {
		return nil
	}
	if _, ok := failOnLevels[value]; !ok {
		return fmt.Errorf("invalid --fail-on value %q (want none, warning or error)", value)
	}
	return nil
}

// failOnCheck returns an *ExitError when issues contain any at or above the
// failOn severity, and nil otherwise.
func failOnCheck(issues []*analyzer.Issue, failOn string) error {
	threshold, ok := failOnLevels[failOn]
	if !ok {
		return nil // "" or "none"
	}
	count, worst := 0, 0
	for _, issue := range issues {
		level := failOnLevels[issue.Severity]
		if level >= threshold {
			count++
			worst = max(worst, level)
		}
	}
	if count == 0 {
		return nil
	}
	code := ExitWarnings
	if worst == failOnLevels["error"] {
		code = ExitBlocking
	}
	return &ExitError{
		Code: code,
		Msg:  fmt.Sprintf("found %d issue(s) at or above --fail-on=%s", count, failOn),
	}
}
