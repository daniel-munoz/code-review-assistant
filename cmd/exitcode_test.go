package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func issues(severities ...string) []*analyzer.Issue {
	out := make([]*analyzer.Issue, len(severities))
	for i, s := range severities {
		out[i] = &analyzer.Issue{Severity: s}
	}
	return out
}

func TestValidateFailOn(t *testing.T) {
	for _, v := range []string{"", "none", "warning", "error"} {
		assert.NoError(t, validateFailOn(v), v)
	}
	err := validateFailOn("critical")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid --fail-on value "critical"`)
}

func TestFailOnCheck(t *testing.T) {
	cases := []struct {
		name     string
		failOn   string
		issues   []*analyzer.Issue
		wantCode int // 0 = no error
		wantN    int
	}{
		{"unset never fails", "", issues("error", "warning"), 0, 0},
		{"none never fails", "none", issues("error"), 0, 0},
		{"warning, clean", "warning", issues(), 0, 0},
		{"warning, info only", "warning", issues("info", "info"), 0, 0},
		{"warning, warnings only", "warning", issues("info", "warning", "warning"), ExitWarnings, 2},
		{"warning, with an error", "warning", issues("warning", "error"), ExitBlocking, 2},
		{"error, warnings only", "error", issues("warning"), 0, 0},
		{"error, with errors", "error", issues("warning", "error", "error"), ExitBlocking, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := failOnCheck(c.issues, c.failOn)
			if c.wantCode == 0 {
				assert.NoError(t, err)
				return
			}
			var exitErr *ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, c.wantCode, exitErr.Code)
			assert.Contains(t, err.Error(), fmt.Sprintf("%d issue(s)", c.wantN))
			assert.Contains(t, err.Error(), "--fail-on="+c.failOn)
		})
	}
}

func TestExitCode(t *testing.T) {
	assert.Equal(t, ExitClean, ExitCode(nil))
	assert.Equal(t, ExitToolError, ExitCode(errors.New("boom")))
	assert.Equal(t, ExitWarnings, ExitCode(&ExitError{Code: ExitWarnings, Msg: "w"}))
	assert.Equal(t, ExitBlocking, ExitCode(fmt.Errorf("wrapped: %w", &ExitError{Code: ExitBlocking, Msg: "b"})))
}
