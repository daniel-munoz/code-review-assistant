package orchestrator

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runForResult runs the orchestrator with output discarded to a temp file and
// returns it so the test can inspect Result().
func runForResult(t *testing.T, target string, diff config.DiffConfig) *Orchestrator {
	t.Helper()
	cfg := config.Default()
	cfg.Language = "go"
	cfg.Output.Format = "json"
	cfg.Output.OutputFile = filepath.Join(t.TempDir(), "report.json")
	cfg.Output.QuietMode = true
	cfg.Analysis.EnableCoverage = false
	cfg.Analysis.Diff = diff

	orch, err := New(cfg, target)
	require.NoError(t, err)
	t.Cleanup(func() { orch.Close() })
	orch.stderr = io.Discard
	require.NoError(t, orch.Run(target))
	return orch
}

func TestResult_NilBeforeRun(t *testing.T) {
	cfg := config.Default()
	cfg.Language = "go"
	orch, err := New(cfg, t.TempDir())
	require.NoError(t, err)
	defer orch.Close()
	assert.Nil(t, orch.Result())
}

func TestResult_FullRun(t *testing.T) {
	dir := newGoRepo(t)
	orch := runForResult(t, dir, config.DiffConfig{})
	require.NotNil(t, orch.Result())
	assert.NotEmpty(t, orch.Result().Issues)
}

func TestResult_DiffRun(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	gitT(t, dir, "add", "a/a.go")

	orch := runForResult(t, dir, config.DiffConfig{Mode: "staged"})
	require.NotNil(t, orch.Result())
	assert.NotEmpty(t, orch.Result().Issues)
}

func TestResult_EmptyDiffRun(t *testing.T) {
	dir := newGoRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("docs\n"), 0o644))
	gitT(t, dir, "add", "README.md")

	orch := runForResult(t, dir, config.DiffConfig{Mode: "staged"})
	require.NotNil(t, orch.Result())
	assert.Empty(t, orch.Result().Issues)
}
