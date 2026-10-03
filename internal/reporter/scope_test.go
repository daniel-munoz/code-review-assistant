package reporter

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sha = "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678"

func TestFormatScope(t *testing.T) {
	assert.Equal(t, "", formatScope(nil))
	assert.Equal(t, "--staged · no matching files changed",
		formatScope(&analyzer.Scope{Mode: "staged", Files: []string{}}))
	assert.Equal(t, "--since=HEAD~1 (a1b2c3d) · 1 file analyzed",
		formatScope(&analyzer.Scope{Mode: "since", Ref: "HEAD~1", Base: sha, Files: []string{"a.go"}}))
	assert.Equal(t, "--branch=main (merge-base a1b2c3d) · 2 files analyzed",
		formatScope(&analyzer.Scope{Mode: "branch", Ref: "main", Base: sha, Files: []string{"a.go", "b.go"}}))
}

// emptyDiffResult mirrors orchestrator.emptyResult: what an empty diff run reports.
func emptyDiffResult() *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		ProjectPath: "/p",
		Metrics:     &analyzer.AggregateMetrics{LargestFiles: []*analyzer.FileSize{}},
		Files:       []*analyzer.FileAnalysis{},
		Issues:      []*analyzer.Issue{},
		Scope:       &analyzer.Scope{Mode: "staged", Files: []string{}},
	}
}

func TestReporters_RenderScopeOnEmptyDiffResult(t *testing.T) {
	for _, format := range []string{"markdown", "html", "json"} {
		t.Run(format, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "report."+format)
			r, err := NewReporter(&config.OutputConfig{Format: format, OutputFile: out, JSONPretty: true})
			require.NoError(t, err)
			require.NoError(t, r.Report(emptyDiffResult(), nil))

			data, err := os.ReadFile(out)
			require.NoError(t, err)
			switch format {
			case "json":
				var doc struct {
					Result struct {
						Scope map[string]interface{} `json:"scope"`
					} `json:"result"`
				}
				require.NoError(t, json.Unmarshal(data, &doc))
				scope := doc.Result.Scope
				assert.Equal(t, "staged", scope["mode"])
				assert.Equal(t, []interface{}{}, scope["files"])
			default:
				assert.Contains(t, string(data), "--staged · no matching files changed")
			}
		})
	}
}

func TestConsoleReporter_PrintsScope(t *testing.T) {
	r := NewConsoleReporter(&config.OutputConfig{Format: "console"})
	out := captureStdout(t, func() { require.NoError(t, r.Report(emptyDiffResult(), nil)) })
	assert.Contains(t, out, "Scope: --staged · no matching files changed")
}

func TestReporters_NoScopeLineOnFullRun(t *testing.T) {
	res := emptyDiffResult()
	res.Scope = nil
	r := NewConsoleReporter(&config.OutputConfig{Format: "console"})
	out := captureStdout(t, func() { require.NoError(t, r.Report(res, nil)) })
	assert.False(t, strings.Contains(out, "Scope:"))
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	rd, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	require.NoError(t, w.Close())
	data, err := io.ReadAll(rd)
	require.NoError(t, err)
	return string(data)
}
