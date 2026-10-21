package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withPreset(name string) func(*config.Config) {
	return func(c *config.Config) { c.Preset = name }
}

func TestRun_SetsPreset(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	cfg := config.Default()
	cfg.Preset = "strict"
	cfg.Output.Format = "json"
	cfg.Output.OutputFile = out
	cfg.Output.QuietMode = true
	cfg.Analysis.EnableCoverage = false
	target := filepath.Join("..", "..", "testdata", "sample")

	orch, err := New(cfg, target)
	require.NoError(t, err)
	require.NoError(t, orch.Run(target))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var doc struct {
		Result *analyzer.AnalysisResult `json:"result"`
	}
	require.NoError(t, json.Unmarshal(data, &doc))
	assert.Equal(t, "strict", doc.Result.Preset)
}

func TestDiffRun_SetsPreset(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	gitT(t, dir, "add", "a/a.go")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, withPreset("strict"))

	require.NotZero(t, run.result.TotalFiles)
	assert.Equal(t, "strict", run.result.Preset)
}

func TestDiffRun_NoChangedFiles_SetsPreset(t *testing.T) {
	dir := newGoRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("docs\n"), 0o644))
	gitT(t, dir, "add", "README.md")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, withPreset("strict"))

	assert.Equal(t, 0, run.result.TotalFiles)
	assert.Equal(t, "strict", run.result.Preset)
}
