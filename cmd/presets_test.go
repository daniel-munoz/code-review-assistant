package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// runPresetCmd runs c's RunE with args and returns what it wrote to stdout.
func runPresetCmd(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	c.SetOut(&out)
	t.Cleanup(func() { c.SetOut(nil) })
	if err := c.Args(c, args); err != nil {
		return "", err
	}
	err := c.RunE(c, args)
	return out.String(), err
}

func TestListPresets(t *testing.T) {
	out, err := runPresetCmd(t, listPresetsCmd)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 3)
	for i, p := range config.BuiltinPresets() {
		assert.True(t, strings.HasPrefix(lines[i], p.Name), lines[i])
		assert.Contains(t, lines[i], p.Description)
	}
}

func TestShowPreset_Builtin(t *testing.T) {
	out, err := runPresetCmd(t, showPresetCmd, "strict")
	require.NoError(t, err)

	want, err := config.BuiltinPresetYAML("strict")
	require.NoError(t, err)
	assert.Equal(t, string(want), out)
}

func TestShowPreset_File(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid", func(t *testing.T) {
		content := "name: team\nanalysis:\n  max_parameters: 9\n"
		path := filepath.Join(dir, "team.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		out, err := runPresetCmd(t, showPresetCmd, path)
		require.NoError(t, err)
		assert.Equal(t, content, out)
	})

	t.Run("out-of-scope key", func(t *testing.T) {
		path := filepath.Join(dir, "bad.yaml")
		require.NoError(t, os.WriteFile(path, []byte("output:\n  format: json\n"), 0o644))

		_, err := runPresetCmd(t, showPresetCmd, path)
		require.Error(t, err)
	})
}

func TestShowPreset_Unknown(t *testing.T) {
	_, err := runPresetCmd(t, showPresetCmd, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "available:")
}

func TestShowPreset_RequiresOneArg(t *testing.T) {
	_, err := runPresetCmd(t, showPresetCmd)
	require.Error(t, err)
}

func TestInitTemplate_HasPresetHint(t *testing.T) {
	assert.Contains(t, defaultConfigYAML, "# preset: standard")

	var parsed map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(defaultConfigYAML), &parsed))
	_, active := parsed["preset"]
	assert.False(t, active, "preset hint must stay commented out")
}
