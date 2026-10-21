package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullThresholdsYAML sets every preset-scoped key to a value no preset uses.
const fullThresholdsYAML = `analysis:
  large_file_threshold: 111
  long_function_threshold: 11
  min_comment_ratio: 0.33
  complexity_threshold: 15
  max_parameters: 11
  max_nesting_depth: 11
  max_return_statements: 11
  detect_magic_numbers: false
  detect_duplicate_errors: false
  detect_non_null_assertions: false
  detect_run_blocking: false
  min_coverage_threshold: 11
  max_imports: 11
  max_external_dependencies: 11
  detect_circular_deps: false
`

// writeConfig writes a config file into dir (created if needed) and returns its path.
func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// presetAnalysis returns the analysis config a built-in preset produces over the defaults.
func presetAnalysis(t *testing.T, name string) AnalysisConfig {
	t.Helper()
	p, err := LoadPreset(name, "")
	require.NoError(t, err)
	a := Default().Analysis
	p.ApplyTo(&a)
	return a
}

func TestLoadConfigWithPreset_NoPreset(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	path := writeConfig(t, t.TempDir(), "analysis:\n  complexity_threshold: 12\n")

	want, err := LoadConfig(path)
	require.NoError(t, err)
	got, err := LoadConfigWithPreset(path, "")
	require.NoError(t, err)

	assert.Equal(t, want, got)
	assert.Empty(t, got.Preset)
}

func TestLoadConfigWithPreset_ExplicitBeatsConfigFile(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	path := writeConfig(t, t.TempDir(), fullThresholdsYAML)

	cfg, err := LoadConfigWithPreset(path, "strict")
	require.NoError(t, err)

	strict := presetAnalysis(t, "strict")
	got := cfg.Analysis
	got.ExcludePatterns = strict.ExcludePatterns
	assert.Equal(t, strict, got)
	assert.Equal(t, "strict", cfg.Preset)
}

func TestLoadConfigWithPreset_ExplicitKeepsOutOfScope(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	path := writeConfig(t, t.TempDir(), `analysis:
  enable_coverage: false
  workers: 3
  exclude_patterns: ["x/**"]
output:
  format: json
`)

	cfg, err := LoadConfigWithPreset(path, "relaxed")
	require.NoError(t, err)

	assert.False(t, cfg.Analysis.EnableCoverage)
	assert.Equal(t, 3, cfg.Analysis.Workers)
	assert.Equal(t, []string{"x/**"}, cfg.Analysis.ExcludePatterns)
	assert.Equal(t, "json", cfg.Output.Format)
	assert.Equal(t, 20, cfg.Analysis.ComplexityThreshold, "relaxed still applied")
}

func TestLoadConfigWithPreset_EnvPreset(t *testing.T) {
	t.Setenv("CRA_PRESET", "relaxed")
	path := writeConfig(t, t.TempDir(), fullThresholdsYAML)

	cfg, err := LoadConfigWithPreset(path, "")
	require.NoError(t, err)

	assert.Equal(t, 20, cfg.Analysis.ComplexityThreshold)
	assert.False(t, cfg.Analysis.DetectMagicNumbers)
	assert.Equal(t, "relaxed", cfg.Preset)
}

func TestLoadConfigWithPreset_FlagBeatsEnv(t *testing.T) {
	t.Setenv("CRA_PRESET", "relaxed")
	path := writeConfig(t, t.TempDir(), "")

	cfg, err := LoadConfigWithPreset(path, "strict")
	require.NoError(t, err)

	assert.Equal(t, 7, cfg.Analysis.ComplexityThreshold)
	assert.Equal(t, "strict", cfg.Preset)
}

func TestLoadConfigWithPreset_ConfigFilePreset(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	path := writeConfig(t, t.TempDir(), "preset: strict\nanalysis:\n  complexity_threshold: 12\n")

	cfg, err := LoadConfigWithPreset(path, "")
	require.NoError(t, err)

	want := presetAnalysis(t, "strict")
	want.ComplexityThreshold = 12
	assert.Equal(t, want, cfg.Analysis)
	assert.Equal(t, "strict", cfg.Preset)
}

func TestLoadConfigWithPreset_ExplicitIgnoresConfigPreset(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	path := writeConfig(t, t.TempDir(), "preset: relaxed\n")

	cfg, err := LoadConfigWithPreset(path, "strict")
	require.NoError(t, err)

	assert.Equal(t, presetAnalysis(t, "strict"), cfg.Analysis)
	assert.Equal(t, "strict", cfg.Preset)
}

func TestLoadConfigWithPreset_ConfigRelativePath(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	cfgDir := filepath.Join(t.TempDir(), "cfg")
	path := writeConfig(t, cfgDir, "preset: team.yaml\n")
	writePresetFile(t, cfgDir, "team.yaml", "analysis:\n  max_parameters: 9\n")
	t.Chdir(t.TempDir()) // the working directory must not matter

	cfg, err := LoadConfigWithPreset(path, "")
	require.NoError(t, err)

	assert.Equal(t, 9, cfg.Analysis.MaxParameters)
	assert.Equal(t, "team.yaml", cfg.Preset)
}

func TestLoadConfigWithPreset_ExplicitRelativeToCwd(t *testing.T) {
	t.Setenv("CRA_PRESET", "")
	dir := t.TempDir()
	writePresetFile(t, dir, "team.yaml", "analysis:\n  max_parameters: 9\n")
	path := writeConfig(t, t.TempDir(), "")
	t.Chdir(dir)

	cfg, err := LoadConfigWithPreset(path, "./team.yaml")
	require.NoError(t, err)

	assert.Equal(t, 9, cfg.Analysis.MaxParameters)
	assert.Equal(t, "./team.yaml", cfg.Preset)
}

func TestLoadConfigWithPreset_InvalidPresetErrors(t *testing.T) {
	t.Setenv("CRA_PRESET", "")

	t.Run("explicit", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), "")
		_, err := LoadConfigWithPreset(path, "nope")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown preset")
	})

	t.Run("config file", func(t *testing.T) {
		path := writeConfig(t, t.TempDir(), "preset: nope\n")
		_, err := LoadConfigWithPreset(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown preset")
	})
}
