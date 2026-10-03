package orchestrator

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sevenParams triggers too_many_parameters (default max 5).
const sevenParams = "package %s\n\nfunc F(a, b, c, d, e, f, g int) int { return a + b + c + d + e + f + g }\n"

func gitT(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{
		"-c", "user.name=Test", "-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
}

func writeGo(t *testing.T, dir, rel, pkg string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(strings.ReplaceAll(sevenParams, "%s", pkg)), 0o644))
}

// newGoRepo creates a committed Go module with packages a, b and vendor/v,
// each containing one function that triggers too_many_parameters.
func newGoRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n\ngo 1.22\n"), 0o644))
	writeGo(t, dir, "a/a.go", "a")
	writeGo(t, dir, "b/b.go", "b")
	writeGo(t, dir, "vendor/v/v.go", "v")
	gitT(t, dir, "init", "-q")
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

// touch appends a comment so the file shows up as modified.
func touch(t *testing.T, dir, rel string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, filepath.FromSlash(rel)), os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString("// changed\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

type diffRun struct {
	result *analyzer.AnalysisResult
	stderr string
}

// runDiffJSON runs the orchestrator in diff mode with JSON output and returns
// the decoded result and captured stderr. mutate may adjust the config.
func runDiffJSON(t *testing.T, target string, diff config.DiffConfig, mutate func(*config.Config)) diffRun {
	t.Helper()
	out := filepath.Join(t.TempDir(), "report.json")
	cfg := config.Default()
	cfg.Language = "go"
	cfg.Output.Format = "json"
	cfg.Output.OutputFile = out
	cfg.Output.QuietMode = true
	cfg.Analysis.Diff = diff
	if mutate != nil {
		mutate(cfg)
	}

	orch, err := New(cfg, target)
	require.NoError(t, err)
	defer orch.Close()
	var stderr bytes.Buffer
	orch.stderr = &stderr

	require.NoError(t, orch.Run(target))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var doc struct {
		Result *analyzer.AnalysisResult `json:"result"`
	}
	require.NoError(t, json.Unmarshal(data, &doc))
	return diffRun{result: doc.Result, stderr: stderr.String()}
}

func issueFiles(r *analyzer.AnalysisResult) []string {
	var out []string
	for _, i := range r.Issues {
		if i.File != "" {
			out = append(out, i.File)
		}
	}
	return out
}

func TestRunDiff_StagedAnalyzesOnlyChangedFiles(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	gitT(t, dir, "add", "a/a.go")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, nil)

	want := filepath.Join(dir, "a", "a.go")
	require.NotNil(t, run.result.Scope)
	assert.Equal(t, "staged", run.result.Scope.Mode)
	assert.Equal(t, []string{want}, run.result.Scope.Files)
	assert.Equal(t, 1, run.result.TotalFiles)
	assert.NotEmpty(t, issueFiles(run.result))
	for _, f := range issueFiles(run.result) {
		assert.Equal(t, want, f, "issues must come only from changed files")
	}
	assert.Nil(t, run.result.Coverage, "coverage is off by default in diff mode")
	assert.Nil(t, run.result.Dependencies, "deps are off by default in diff mode")
}

func TestRunDiff_EmptyChangeSetSucceeds(t *testing.T) {
	dir := newGoRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("docs\n"), 0o644))
	gitT(t, dir, "add", "README.md")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, nil)

	require.NotNil(t, run.result.Scope)
	assert.NotNil(t, run.result.Scope.Files)
	assert.Empty(t, run.result.Scope.Files)
	assert.Equal(t, 0, run.result.TotalFiles)
	assert.Empty(t, run.result.Issues)
}

func TestRunDiff_ExcludedChangedFileIsSkipped(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "vendor/v/v.go") // matches the Go default "vendor/**"
	gitT(t, dir, "add", "-A")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, nil)
	assert.Empty(t, run.result.Scope.Files)
}

func TestRunDiff_SkipsFilesMissingOnDisk(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	touch(t, dir, "b/b.go")
	gitT(t, dir, "add", "-A")
	require.NoError(t, os.Remove(filepath.Join(dir, "b", "b.go"))) // staged edit, then deleted

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, nil)
	assert.Equal(t, []string{filepath.Join(dir, "a", "a.go")}, run.result.Scope.Files)
	assert.NotContains(t, run.stderr, "failed to parse")
}

func TestRunDiff_SubdirectoryTarget(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	touch(t, dir, "b/b.go")
	gitT(t, dir, "add", "-A")

	target := filepath.Join(dir, "b")
	run := runDiffJSON(t, target, config.DiffConfig{Mode: "staged"}, nil)
	assert.Equal(t, []string{filepath.Join(target, "b.go")}, run.result.Scope.Files)
}

func TestRunDiff_PartiallyStagedWarnsOnStderr(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	gitT(t, dir, "add", "a/a.go")
	touch(t, dir, "a/a.go") // unstaged on top

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, nil)
	assert.Contains(t, run.stderr, "Warning: 1 staged file(s) also have unstaged changes; results reflect the working tree:")
	assert.Contains(t, run.stderr, filepath.Join("a", "a.go"))
}

func TestRunDiff_WithDepsUsesFullProject(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	gitT(t, dir, "add", "a/a.go")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged", WithDeps: true}, nil)

	require.NotNil(t, run.result.Dependencies)
	assert.Equal(t, 2, run.result.Dependencies.TotalPackages, "packages a and b (vendor excluded), not just the changed one")
	assert.Equal(t, 1, run.result.TotalFiles, "detectors still only see changed files")
	for _, f := range issueFiles(run.result) {
		assert.Equal(t, filepath.Join(dir, "a", "a.go"), f)
	}
}

func TestRunDiff_ConfigHistoryIsSkippedWithNote(t *testing.T) {
	dir := newGoRepo(t)
	touch(t, dir, "a/a.go")
	gitT(t, dir, "add", "a/a.go")
	store := filepath.Join(t.TempDir(), "store")

	run := runDiffJSON(t, dir, config.DiffConfig{Mode: "staged"}, func(c *config.Config) {
		c.Storage.Enabled = true
		c.Storage.Path = store
		c.Comparison.Enabled = true
	})

	assert.Contains(t, run.stderr, "Note: diff runs are not saved or compared (storage/comparison skipped)")
	_, err := os.Stat(store)
	assert.True(t, os.IsNotExist(err), "no storage must be created in diff mode")
}

func TestRunDiff_GitErrorIsReturned(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir() // not a repo
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644))

	cfg := config.Default()
	cfg.Language = "go"
	cfg.Output.QuietMode = true
	cfg.Analysis.Diff = config.DiffConfig{Mode: "staged"}
	orch, err := New(cfg, dir)
	require.NoError(t, err)

	err = orch.Run(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--staged requires a git repository")
}

func TestRun_FullModeHasNoScope(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	cfg := config.Default()
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
	assert.NotContains(t, string(data), `"scope"`)
}
