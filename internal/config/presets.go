package config

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// builtinPresetNames lists the embedded presets in display order.
var builtinPresetNames = []string{"strict", "standard", "relaxed"}

//go:embed presets/*.yaml
var builtinPresetFS embed.FS

// Preset is a named set of analysis thresholds and detector switches.
//
// A preset only covers the keys in PresetAnalysis. It never changes whether
// coverage runs, output, storage, workers or exclude patterns.
type Preset struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Analysis    PresetAnalysis `yaml:"analysis"`

	// Label identifies the preset in reports: the built-in name, or the
	// file path exactly as given.
	Label string `yaml:"-"`
}

// PresetAnalysis holds the analysis settings a preset may set. Nil fields are
// left unchanged when the preset is applied.
type PresetAnalysis struct {
	LargeFileThreshold      *int     `yaml:"large_file_threshold"`
	LongFunctionThreshold   *int     `yaml:"long_function_threshold"`
	MinCommentRatio         *float64 `yaml:"min_comment_ratio"`
	ComplexityThreshold     *int     `yaml:"complexity_threshold"`
	MaxParameters           *int     `yaml:"max_parameters"`
	MaxNestingDepth         *int     `yaml:"max_nesting_depth"`
	MaxReturnStatements     *int     `yaml:"max_return_statements"`
	DetectMagicNumbers      *bool    `yaml:"detect_magic_numbers"`
	DetectDuplicateErrors   *bool    `yaml:"detect_duplicate_errors"`
	DetectNonNullAssertions *bool    `yaml:"detect_non_null_assertions"`
	DetectRunBlocking       *bool    `yaml:"detect_run_blocking"`
	MinCoverageThreshold    *float64 `yaml:"min_coverage_threshold"`
	MaxImports              *int     `yaml:"max_imports"`
	MaxExternalDependencies *int     `yaml:"max_external_dependencies"`
	DetectCircularDeps      *bool    `yaml:"detect_circular_deps"`
}

// presetField maps one preset key to its PresetAnalysis and AnalysisConfig fields.
type presetField struct {
	key   string // key under "analysis." in config files
	value func(*PresetAnalysis) (any, bool)
	apply func(*PresetAnalysis, *AnalysisConfig)
}

func (f presetField) isSet(p *PresetAnalysis) bool {
	_, ok := f.value(p)
	return ok
}

func field[T any](key string, src func(*PresetAnalysis) *T, dst func(*AnalysisConfig) *T) presetField {
	return presetField{
		key: key,
		value: func(p *PresetAnalysis) (any, bool) {
			if v := src(p); v != nil {
				return *v, true
			}
			return nil, false
		},
		apply: func(p *PresetAnalysis, a *AnalysisConfig) {
			if v := src(p); v != nil {
				*dst(a) = *v
			}
		},
	}
}

// presetFields is the single list of keys a preset may set.
var presetFields = []presetField{
	field("large_file_threshold", func(p *PresetAnalysis) *int { return p.LargeFileThreshold }, func(a *AnalysisConfig) *int { return &a.LargeFileThreshold }),
	field("long_function_threshold", func(p *PresetAnalysis) *int { return p.LongFunctionThreshold }, func(a *AnalysisConfig) *int { return &a.LongFunctionThreshold }),
	field("min_comment_ratio", func(p *PresetAnalysis) *float64 { return p.MinCommentRatio }, func(a *AnalysisConfig) *float64 { return &a.MinCommentRatio }),
	field("complexity_threshold", func(p *PresetAnalysis) *int { return p.ComplexityThreshold }, func(a *AnalysisConfig) *int { return &a.ComplexityThreshold }),
	field("max_parameters", func(p *PresetAnalysis) *int { return p.MaxParameters }, func(a *AnalysisConfig) *int { return &a.MaxParameters }),
	field("max_nesting_depth", func(p *PresetAnalysis) *int { return p.MaxNestingDepth }, func(a *AnalysisConfig) *int { return &a.MaxNestingDepth }),
	field("max_return_statements", func(p *PresetAnalysis) *int { return p.MaxReturnStatements }, func(a *AnalysisConfig) *int { return &a.MaxReturnStatements }),
	field("detect_magic_numbers", func(p *PresetAnalysis) *bool { return p.DetectMagicNumbers }, func(a *AnalysisConfig) *bool { return &a.DetectMagicNumbers }),
	field("detect_duplicate_errors", func(p *PresetAnalysis) *bool { return p.DetectDuplicateErrors }, func(a *AnalysisConfig) *bool { return &a.DetectDuplicateErrors }),
	field("detect_non_null_assertions", func(p *PresetAnalysis) *bool { return p.DetectNonNullAssertions }, func(a *AnalysisConfig) *bool { return &a.DetectNonNullAssertions }),
	field("detect_run_blocking", func(p *PresetAnalysis) *bool { return p.DetectRunBlocking }, func(a *AnalysisConfig) *bool { return &a.DetectRunBlocking }),
	field("min_coverage_threshold", func(p *PresetAnalysis) *float64 { return p.MinCoverageThreshold }, func(a *AnalysisConfig) *float64 { return &a.MinCoverageThreshold }),
	field("max_imports", func(p *PresetAnalysis) *int { return p.MaxImports }, func(a *AnalysisConfig) *int { return &a.MaxImports }),
	field("max_external_dependencies", func(p *PresetAnalysis) *int { return p.MaxExternalDependencies }, func(a *AnalysisConfig) *int { return &a.MaxExternalDependencies }),
	field("detect_circular_deps", func(p *PresetAnalysis) *bool { return p.DetectCircularDeps }, func(a *AnalysisConfig) *bool { return &a.DetectCircularDeps }),
}

// ApplyTo copies the preset's set values onto a, leaving the rest unchanged.
func (p *Preset) ApplyTo(a *AnalysisConfig) {
	for _, f := range presetFields {
		f.apply(&p.Analysis, a)
	}
}

// SetViperDefaults registers the preset's set values as viper defaults, so
// values from a config file still override them.
func (p *Preset) SetViperDefaults(v *viper.Viper) {
	for _, f := range presetFields {
		if val, ok := f.value(&p.Analysis); ok {
			v.SetDefault("analysis."+f.key, val)
		}
	}
}

// IsPresetPath reports whether value names a preset file rather than a
// built-in preset: it contains a path separator or has a YAML extension.
func IsPresetPath(value string) bool {
	if strings.ContainsRune(value, '/') || strings.ContainsRune(value, filepath.Separator) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(value))
	return ext == ".yaml" || ext == ".yml"
}

// LoadPreset loads a built-in preset by name (case-insensitive) or a preset
// file by path. Relative paths resolve against baseDir (the working directory
// when baseDir is empty).
func LoadPreset(value, baseDir string) (*Preset, error) {
	value = strings.TrimSpace(value)
	if IsPresetPath(value) {
		return loadPresetFile(value, baseDir)
	}

	name := strings.ToLower(value)
	data, err := BuiltinPresetYAML(name)
	if err != nil {
		return nil, err
	}
	p, err := parsePreset(data, name)
	if err != nil {
		return nil, err
	}
	p.Label = name
	return p, nil
}

func loadPresetFile(value, baseDir string) (*Preset, error) {
	path := value
	if !filepath.IsAbs(path) && baseDir != "" {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read preset %q: %w", value, err)
	}
	p, err := parsePreset(data, path)
	if err != nil {
		return nil, err
	}
	p.Label = value
	return p, nil
}

// parsePreset decodes preset YAML, rejecting any key a preset may not set.
// source names the preset in error messages.
func parsePreset(data []byte, source string) (*Preset, error) {
	var p Preset
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid preset %q: %w", source, nameTypeErrorKeys(data, err))
	}
	return &p, nil
}

// nameTypeErrorKeys rewrites yaml.v3 type errors, which only give a line
// number ("line 2: cannot unmarshal ..."), to start with the offending key.
func nameTypeErrorKeys(data []byte, err error) error {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return err
	}
	var root yaml.Node
	if yaml.Unmarshal(data, &root) != nil {
		return err
	}
	keys := map[int]string{}
	collectKeyLines(&root, "", keys)

	msgs := make([]string, 0, len(typeErr.Errors))
	for _, msg := range typeErr.Errors {
		var line int
		if _, scanErr := fmt.Sscanf(msg, "line %d:", &line); scanErr == nil && keys[line] != "" {
			msg = keys[line] + ": " + msg
		}
		msgs = append(msgs, msg)
	}
	return errors.New(strings.Join(msgs, "; "))
}

// collectKeyLines records the dotted key path of every mapping key by line.
func collectKeyLines(n *yaml.Node, prefix string, keys map[int]string) {
	if n.Kind == yaml.DocumentNode {
		for _, c := range n.Content {
			collectKeyLines(c, prefix, keys)
		}
		return
	}
	if n.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		path := k.Value
		if prefix != "" {
			path = prefix + "." + k.Value
		}
		keys[k.Line] = path
		collectKeyLines(v, path, keys)
	}
}

// BuiltinPresetYAML returns the embedded YAML of a built-in preset.
func BuiltinPresetYAML(name string) ([]byte, error) {
	for _, n := range builtinPresetNames {
		if n == name {
			return builtinPresetFS.ReadFile("presets/" + n + ".yaml")
		}
	}
	return nil, fmt.Errorf("unknown preset %q (available: %s)", name, strings.Join(builtinPresetNames, ", "))
}

// BuiltinPresets returns the built-in presets in display order.
func BuiltinPresets() []*Preset {
	presets := make([]*Preset, 0, len(builtinPresetNames))
	for _, name := range builtinPresetNames {
		p, err := LoadPreset(name, "")
		if err != nil {
			panic(fmt.Sprintf("embedded preset %s is invalid: %v", name, err)) // covered by tests
		}
		presets = append(presets, p)
	}
	return presets
}
