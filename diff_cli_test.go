package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// craBinary builds the CLI once per test run.
func craBinary(tb testing.TB) string {
	tb.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "cra-bin")
		if err != nil {
			buildErr = err
			return
		}
		binPath = filepath.Join(dir, "cra")
		out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build: %v: %s", err, out)
		}
	})
	require.NoError(tb, buildErr)
	return binPath
}

func gitCLI(tb testing.TB, dir string, args ...string) {
	tb.Helper()
	full := append([]string{
		"-c", "user.name=Test", "-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(tb, err, "git %s: %s", strings.Join(args, " "), out)
}

// fixtureRepo copies testdata/sample into a fresh committed repo.
func fixtureRepo(tb testing.TB) string {
	tb.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		tb.Skip("git not available")
	}
	dir := tb.TempDir()
	entries, err := os.ReadDir("testdata/sample")
	require.NoError(tb, err)
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join("testdata/sample", e.Name()))
		require.NoError(tb, err)
		require.NoError(tb, os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644))
	}
	require.NoError(tb, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/sample\n\ngo 1.22\n"), 0o644))
	gitCLI(tb, dir, "init", "-q")
	gitCLI(tb, dir, "add", "-A")
	gitCLI(tb, dir, "commit", "-q", "-m", "initial")
	return dir
}

type cliResult struct {
	stdout, stderr string
	exitCode       int
}

func runCRA(tb testing.TB, dir string, args ...string) cliResult {
	tb.Helper()
	cmd := exec.Command(craBinary(tb), append([]string{"analyze", "."}, args...)...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else {
		require.NoError(tb, err)
	}
	return cliResult{stdout.String(), stderr.String(), code}
}

func appendTo(tb testing.TB, path, text string) {
	tb.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(tb, err)
	_, err = f.WriteString(text)
	require.NoError(tb, err)
	require.NoError(tb, f.Close())
}

func decodeScope(t *testing.T, stdout string) map[string]interface{} {
	t.Helper()
	var doc struct {
		Result struct {
			Scope map[string]interface{} `json:"scope"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc), "stdout must be pure JSON:\n%s", stdout)
	require.NotNil(t, doc.Result.Scope, "scope missing")
	return doc.Result.Scope
}

func TestDiffCLI_StagedJSONOnStdout(t *testing.T) {
	dir := fixtureRepo(t)
	appendTo(t, filepath.Join(dir, "util.go"), "\n// changed\n")
	gitCLI(t, dir, "add", "util.go")

	res := runCRA(t, dir, "--staged", "--format", "json", "--quiet")
	require.Equal(t, 0, res.exitCode, res.stderr)

	scope := decodeScope(t, res.stdout)
	assert.Equal(t, "staged", scope["mode"])
	assert.Equal(t, []interface{}{"util.go"}, scope["files"])
}

func TestDiffCLI_EmptyChangeSetExitsZero(t *testing.T) {
	dir := fixtureRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644))
	gitCLI(t, dir, "add", "README.md")

	res := runCRA(t, dir, "--staged", "--format", "json", "--quiet")
	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Equal(t, []interface{}{}, decodeScope(t, res.stdout)["files"])
}

func TestDiffCLI_ParseErrorKeepsStdoutClean(t *testing.T) {
	dir := fixtureRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package sample\n\nfunc {\n"), 0o644))
	gitCLI(t, dir, "add", "broken.go")

	res := runCRA(t, dir, "--staged", "--format", "json", "--quiet")
	require.Equal(t, 0, res.exitCode, res.stderr)
	decodeScope(t, res.stdout) // stdout is still valid JSON
	assert.Contains(t, res.stderr, "failed to parse")
}

func TestDiffCLI_BranchOnFeatureBranch(t *testing.T) {
	dir := fixtureRepo(t)
	gitCLI(t, dir, "checkout", "-q", "-b", "feature")
	appendTo(t, filepath.Join(dir, "complex.go"), "\n// feature work\n")
	gitCLI(t, dir, "commit", "-q", "-am", "feature work")

	res := runCRA(t, dir, "--branch=main", "--format", "json", "--quiet")
	require.Equal(t, 0, res.exitCode, res.stderr)
	scope := decodeScope(t, res.stdout)
	assert.Equal(t, "branch", scope["mode"])
	assert.Equal(t, "main", scope["ref"])
	assert.Equal(t, []interface{}{"complex.go"}, scope["files"])
}

func TestDiffCLI_FlagErrors(t *testing.T) {
	dir := fixtureRepo(t)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"mutually exclusive", []string{"--staged", "--since=HEAD"}, "none of the others can be"},
		{"history flag", []string{"--staged", "--save-report"}, "--save-report cannot be combined with --staged"},
		{"empty since", []string{"--since="}, "--since requires a ref"},
		{"unknown ref", []string{"--since=nope"}, `unknown ref "nope" for --since`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := runCRA(t, dir, c.args...)
			assert.NotEqual(t, 0, res.exitCode)
			assert.Contains(t, res.stderr, c.want)
		})
	}
}

func TestDiffCLI_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))

	res := runCRA(t, dir, "--staged")
	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "--staged requires a git repository")
}

// BenchmarkDiffStaged measures `analyze --staged` end-to-end (process start
// included) on a 1000-file Go repo with 3 staged changes. Target: < 2s/op.
func BenchmarkDiffStaged(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not available")
	}
	dir := b.TempDir()
	require.NoError(b, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/big\n\ngo 1.22\n"), 0o644))
	for i := 0; i < 1000; i++ {
		pkg := fmt.Sprintf("p%03d", i/10)
		path := filepath.Join(dir, pkg, fmt.Sprintf("f%04d.go", i))
		require.NoError(b, os.MkdirAll(filepath.Dir(path), 0o755))
		src := fmt.Sprintf("package %s\n\nfunc F%d(a, b int) int {\n\tif a > b {\n\t\treturn a\n\t}\n\treturn b\n}\n", pkg, i)
		require.NoError(b, os.WriteFile(path, []byte(src), 0o644))
	}
	gitCLI(b, dir, "init", "-q")
	gitCLI(b, dir, "add", "-A")
	gitCLI(b, dir, "commit", "-q", "-m", "initial")
	for _, i := range []int{1, 500, 999} {
		appendTo(b, filepath.Join(dir, fmt.Sprintf("p%03d", i/10), fmt.Sprintf("f%04d.go", i)), "// changed\n")
	}
	gitCLI(b, dir, "add", "-A")
	craBinary(b)

	b.ResetTimer()
	var worst time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		res := runCRA(b, dir, "--staged", "--format", "json", "--quiet")
		if d := time.Since(start); d > worst {
			worst = d
		}
		require.Equal(b, 0, res.exitCode, res.stderr)
	}
	b.ReportMetric(worst.Seconds(), "worst-s")
}
