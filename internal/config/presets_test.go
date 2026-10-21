package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writePresetFile writes content to name inside a temp dir and returns the path.
func writePresetFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestBuiltinPresets_Values(t *testing.T) {
	tests := []struct {
		name     string
		expected AnalysisConfig
	}{
		{"relaxed", AnalysisConfig{
			LargeFileThreshold: 1000, LongFunctionThreshold: 100, ComplexityThreshold: 20,
			MinCommentRatio: 0.05, MaxParameters: 7, MaxNestingDepth: 6, MaxReturnStatements: 5,
			MinCoverageThreshold: 30, MaxImports: 15, MaxExternalDependencies: 15,
			DetectMagicNumbers: false, DetectDuplicateErrors: false,
			DetectNonNullAssertions: true, DetectRunBlocking: true, DetectCircularDeps: true,
		}},
		{"standard", AnalysisConfig{
			LargeFileThreshold: 500, LongFunctionThreshold: 50, ComplexityThreshold: 10,
			MinCommentRatio: 0.15, MaxParameters: 5, MaxNestingDepth: 4, MaxReturnStatements: 3,
			MinCoverageThreshold: 50, MaxImports: 10, MaxExternalDependencies: 10,
			DetectMagicNumbers: true, DetectDuplicateErrors: true,
			DetectNonNullAssertions: true, DetectRunBlocking: true, DetectCircularDeps: true,
		}},
		{"strict", AnalysisConfig{
			LargeFileThreshold: 300, LongFunctionThreshold: 30, ComplexityThreshold: 7,
			MinCommentRatio: 0.20, MaxParameters: 4, MaxNestingDepth: 3, MaxReturnStatements: 3,
			MinCoverageThreshold: 80, MaxImports: 7, MaxExternalDependencies: 7,
			DetectMagicNumbers: true, DetectDuplicateErrors: true,
			DetectNonNullAssertions: true, DetectRunBlocking: true, DetectCircularDeps: true,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := LoadPreset(tt.name, "")
			require.NoError(t, err)

			// Start from the zero value so every in-scope field must come from the preset.
			var got AnalysisConfig
			p.ApplyTo(&got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuiltinPresets_StandardMatchesDefaults(t *testing.T) {
	p, err := LoadPreset("standard", "")
	require.NoError(t, err)

	got := Default().Analysis
	p.ApplyTo(&got)
	assert.Equal(t, Default().Analysis, got)

	for _, f := range presetFields {
		assert.True(t, f.isSet(&p.Analysis), "standard preset should set %s", f.key)
	}
}

func TestBuiltinPresets_Order(t *testing.T) {
	presets := BuiltinPresets()
	names := make([]string, 0, len(presets))
	for _, p := range presets {
		names = append(names, p.Name)
		assert.NotEmpty(t, p.Description, "preset %s needs a description", p.Name)
	}
	assert.Equal(t, []string{"strict", "standard", "relaxed"}, names)
}

func TestLoadPreset_NameNormalization(t *testing.T) {
	for _, value := range []string{"Strict", " strict "} {
		p, err := LoadPreset(value, "")
		require.NoError(t, err, value)
		assert.Equal(t, "strict", p.Label)
	}
}

func TestLoadPreset_UnknownName(t *testing.T) {
	_, err := LoadPreset("strct", "")
	require.Error(t, err)
	for _, want := range []string{"strct", "strict", "standard", "relaxed"} {
		assert.Contains(t, err.Error(), want)
	}
}

func TestLoadPreset_File(t *testing.T) {
	dir := t.TempDir()
	path := writePresetFile(t, dir, "team.yaml", "analysis:\n  complexity_threshold: 12\n")

	p, err := LoadPreset(path, "")
	require.NoError(t, err)
	assert.Equal(t, path, p.Label)
	require.NotNil(t, p.Analysis.ComplexityThreshold)
	assert.Equal(t, 12, *p.Analysis.ComplexityThreshold)

	for _, f := range presetFields {
		if f.key != "complexity_threshold" {
			assert.False(t, f.isSet(&p.Analysis), "%s should be unset", f.key)
		}
	}
}

func TestLoadPreset_RelativeToBaseDir(t *testing.T) {
	dir := t.TempDir()
	writePresetFile(t, dir, "team.yaml", "analysis:\n  max_parameters: 9\n")

	p, err := LoadPreset("team.yaml", dir)
	require.NoError(t, err)
	assert.Equal(t, "team.yaml", p.Label)
	require.NotNil(t, p.Analysis.MaxParameters)
	assert.Equal(t, 9, *p.Analysis.MaxParameters)
}

func TestLoadPreset_EmptyAndMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"empty.yaml": "",
		"meta.yaml":  "name: team\ndescription: Our rules\n",
	} {
		p, err := LoadPreset(writePresetFile(t, dir, name, content), "")
		require.NoError(t, err, name)
		for _, f := range presetFields {
			assert.False(t, f.isSet(&p.Analysis), "%s: %s should be unset", name, f.key)
		}
	}
}

func TestLoadPreset_OutOfScopeKey(t *testing.T) {
	dir := t.TempDir()
	tests := map[string]string{
		"enable_coverage": "analysis:\n  enable_coverage: false\n",
		"workers":         "analysis:\n  workers: 4\n",
		"output":          "output:\n  format: json\n",
	}
	for key, content := range tests {
		t.Run(key, func(t *testing.T) {
			_, err := LoadPreset(writePresetFile(t, dir, key+".yaml", content), "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), key)
		})
	}
}

func TestLoadPreset_WrongType(t *testing.T) {
	path := writePresetFile(t, t.TempDir(), "bad.yaml", "analysis:\n  complexity_threshold: high\n")

	_, err := LoadPreset(path, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "complexity_threshold")
	assert.Contains(t, err.Error(), path)
}

func TestLoadPreset_MissingFile(t *testing.T) {
	_, err := LoadPreset("./missing.yaml", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing.yaml")
}

func TestIsPresetPath(t *testing.T) {
	for _, v := range []string{"./a.yaml", "a.yml", "dir/a", "/abs/x"} {
		assert.True(t, IsPresetPath(v), v)
	}
	for _, v := range []string{"strict", "Strict"} {
		assert.False(t, IsPresetPath(v), v)
	}
}

func TestApplyTo_OnlySetFields(t *testing.T) {
	nine := 9
	p := &Preset{Analysis: PresetAnalysis{MaxParameters: &nine}}

	got := Default().Analysis
	p.ApplyTo(&got)

	want := Default().Analysis
	want.MaxParameters = 9
	assert.Equal(t, want, got)
}
