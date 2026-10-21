package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cyclicRepo is a Go module whose packages a and b import each other, which
// CRA reports as an error-severity circular dependency.
func cyclicRepo(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	files := map[string]string{
		"go.mod":  "module example.com/cyc\n\ngo 1.22\n",
		"a/a.go":  "package a\n\nimport \"example.com/cyc/b\"\n\nfunc A() int { return b.B() }\n",
		"b/b.go":  "package b\n\nimport \"example.com/cyc/a\"\n\nfunc B() int { return a.A() }\n",
		"main.go": "package main\n\nimport \"example.com/cyc/a\"\n\nfunc main() { _ = a.A() }\n",
	}
	for name, src := range files {
		p := filepath.Join(dir, name)
		require.NoError(tb, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(tb, os.WriteFile(p, []byte(src), 0o644))
	}
	return dir
}

// The sample fixture has warning and info issues but no errors.
func TestFailOnCLI_ExitCodes(t *testing.T) {
	sample := fixtureRepo(t)
	cyclic := cyclicRepo(t)
	cases := []struct {
		name string
		dir  string
		args []string
		want int
	}{
		{"no --fail-on, warnings present", sample, nil, 0},
		{"--fail-on=none", sample, []string{"--fail-on=none"}, 0},
		{"--fail-on=error, warnings only", sample, []string{"--fail-on=error"}, 0},
		{"--fail-on=warning, warnings only", sample, []string{"--fail-on=warning"}, 1},
		{"--fail-on=warning, error present", cyclic, []string{"--fail-on=warning"}, 2},
		{"--fail-on=error, error present", cyclic, []string{"--fail-on=error"}, 2},
		{"no --fail-on, error present", cyclic, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := append([]string{"--quiet", "--enable-coverage=false", "--format", "json"}, c.args...)
			res := runCRA(t, c.dir, args...)
			assert.Equal(t, c.want, res.exitCode, res.stderr)
			var doc map[string]interface{}
			assert.NoError(t, json.Unmarshal([]byte(res.stdout), &doc), "report must still be written:\n%s", res.stdout)
			if c.want != 0 {
				assert.Contains(t, res.stderr, "--fail-on=")
				assert.NotContains(t, res.stderr, "Usage:")
			}
		})
	}
}

func TestFailOnCLI_ToolErrorsExitThree(t *testing.T) {
	sample := fixtureRepo(t)
	t.Run("invalid --fail-on", func(t *testing.T) {
		res := runCRA(t, sample, "--fail-on=critical")
		assert.Equal(t, 3, res.exitCode)
		assert.Contains(t, res.stderr, `invalid --fail-on value "critical"`)
	})
	t.Run("unknown ref", func(t *testing.T) {
		res := runCRA(t, sample, "--since=nope")
		assert.Equal(t, 3, res.exitCode)
	})
}
