package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderToFile runs the reporter for format on a result with the given preset
// and returns the written output.
func renderToFile(t *testing.T, format, preset string) string {
	t.Helper()
	res := emptyDiffResult()
	res.Preset = preset
	out := filepath.Join(t.TempDir(), "report."+format)
	r, err := NewReporter(&config.OutputConfig{Format: format, OutputFile: out, JSONPretty: true})
	require.NoError(t, err)
	require.NoError(t, r.Report(res, nil))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	return string(data)
}

func TestJSON_RecordsPreset(t *testing.T) {
	var doc struct {
		Result map[string]interface{} `json:"result"`
	}

	require.NoError(t, json.Unmarshal([]byte(renderToFile(t, "json", "strict")), &doc))
	assert.Equal(t, "strict", doc.Result["preset"])

	doc.Result = nil // Unmarshal merges into an existing map
	require.NoError(t, json.Unmarshal([]byte(renderToFile(t, "json", "")), &doc))
	_, ok := doc.Result["preset"]
	assert.False(t, ok, "no preset key without a preset")
}

func TestSARIF_RecordsPresetInRunProperties(t *testing.T) {
	var doc struct {
		Runs []map[string]json.RawMessage `json:"runs"`
	}

	require.NoError(t, json.Unmarshal([]byte(renderToFile(t, "sarif", "relaxed")), &doc))
	require.Len(t, doc.Runs, 1)
	var props map[string]string
	require.NoError(t, json.Unmarshal(doc.Runs[0]["properties"], &props))
	assert.Equal(t, "relaxed", props["preset"])

	doc.Runs = nil
	require.NoError(t, json.Unmarshal([]byte(renderToFile(t, "sarif", "")), &doc))
	_, ok := doc.Runs[0]["properties"]
	assert.False(t, ok, "no properties without a preset")
}

func TestMarkdownAndHTML_ShowPreset(t *testing.T) {
	assert.Contains(t, renderToFile(t, "markdown", "strict"), "**Preset:** strict")
	assert.NotContains(t, renderToFile(t, "markdown", ""), "Preset")

	assert.Contains(t, renderToFile(t, "html", "strict"), "Preset:")
	assert.Contains(t, renderToFile(t, "html", "strict"), "<strong>strict</strong>")
	assert.NotContains(t, renderToFile(t, "html", ""), "Preset:")
}

func TestConsole_ShowsPreset(t *testing.T) {
	r := NewConsoleReporter(&config.OutputConfig{Format: "console"})

	res := emptyDiffResult()
	res.Preset = "strict"
	out := captureStdout(t, func() { require.NoError(t, r.Report(res, nil)) })
	assert.Contains(t, out, "Preset: strict")

	res.Preset = ""
	out = captureStdout(t, func() { require.NoError(t, r.Report(res, nil)) })
	assert.NotContains(t, out, "Preset")
}

// presetComparison compares two empty results that used the given presets.
func presetComparison(prev, cur string) *comparison.ComparisonResult {
	previous, current := emptyDiffResult(), emptyDiffResult()
	previous.Preset, current.Preset = prev, cur
	return comparison.NewComparator(5).Compare(current, previous, time.Now())
}

func TestComparisonSummary_NotesPresetMismatch(t *testing.T) {
	const note = "different presets (relaxed → strict)"

	t.Run("console", func(t *testing.T) {
		r := NewConsoleReporter(&config.OutputConfig{Format: "console"})
		comp := presetComparison("relaxed", "strict")
		out := captureStdout(t, func() { require.NoError(t, r.Report(comp.Current, comp)) })
		assert.Contains(t, out, note)

		comp = presetComparison("strict", "strict")
		out = captureStdout(t, func() { require.NoError(t, r.Report(comp.Current, comp)) })
		assert.NotContains(t, out, "different presets")
	})

	t.Run("markdown", func(t *testing.T) {
		render := func(comp *comparison.ComparisonResult) string {
			out := filepath.Join(t.TempDir(), "report.md")
			r, err := NewReporter(&config.OutputConfig{Format: "markdown", OutputFile: out})
			require.NoError(t, err)
			require.NoError(t, r.Report(comp.Current, comp))
			data, err := os.ReadFile(out)
			require.NoError(t, err)
			return string(data)
		}
		assert.Contains(t, render(presetComparison("relaxed", "strict")), note)
		assert.NotContains(t, render(presetComparison("strict", "strict")), "different presets")
	})
}
