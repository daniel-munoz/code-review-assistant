package reporter

import (
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

// sarifDoc is the subset of SARIF 2.1.0 the tests inspect.
type sarifDoc struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []struct {
		Tool struct {
			Driver struct {
				Name           string `json:"name"`
				Version        string `json:"version"`
				InformationURI string `json:"informationUri"`
				Rules          []struct {
					ID               string `json:"id"`
					ShortDescription struct {
						Text string `json:"text"`
					} `json:"shortDescription"`
					DefaultConfiguration struct {
						Level string `json:"level"`
					} `json:"defaultConfiguration"`
				} `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []sarifTestResult `json:"results"`
	} `json:"runs"`
}

type sarifTestResult struct {
	RuleID  string `json:"ruleId"`
	Level   string `json:"level"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine int `json:"startLine"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
}

func (r sarifTestResult) uri() string         { return r.Locations[0].PhysicalLocation.ArtifactLocation.URI }
func (r sarifTestResult) line() int           { return r.Locations[0].PhysicalLocation.Region.StartLine }
func (d sarifDoc) results() []sarifTestResult { return d.Runs[0].Results }

func findResult(t *testing.T, d sarifDoc, ruleID string) sarifTestResult {
	t.Helper()
	for _, r := range d.results() {
		if r.RuleID == ruleID {
			return r
		}
	}
	t.Fatalf("no result with ruleId %q", ruleID)
	return sarifTestResult{}
}

// reportSARIF runs the sarif reporter (via the factory) into a file and decodes it.
func reportSARIF(t *testing.T, result *analyzer.AnalysisResult) sarifDoc {
	t.Helper()
	out := filepath.Join(t.TempDir(), "cra.sarif")
	r, err := NewReporter(&config.OutputConfig{Format: "sarif", OutputFile: out, ToolVersion: "9.9.9"})
	require.NoError(t, err)
	require.NoError(t, r.Report(result, nil))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var doc sarifDoc
	require.NoError(t, json.Unmarshal(data, &doc), string(data))
	require.Len(t, doc.Runs, 1)
	return doc
}

func sampleResult(project string) *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		ProjectPath: project,
		Metrics:     &analyzer.AggregateMetrics{},
		Issues: []*analyzer.Issue{
			{Severity: "warning", Type: "too_many_parameters", File: filepath.Join(project, "pkg", "a.go"), Line: 12,
				Function: "F", Message: "Function has too many parameters", Value: 7, Threshold: 5},
			{Severity: "info", Type: "large_file", File: filepath.Join(project, "big.go"), Line: 0,
				Message: "File exceeds recommended size", Value: 900, Threshold: 500},
			{Severity: "error", Type: "high_complexity", File: filepath.Join(project, "pkg", "a.go"), Line: 30,
				Function: "G", Message: "Function has high cyclomatic complexity", Value: 25, Threshold: 10},
			{Severity: "warning", Type: "low_coverage", File: "", Line: 0,
				Message: "Package pkg has low test coverage", Value: 10, Threshold: 50},
			{Severity: "info", Type: "low_comment_ratio", File: "", Line: 0,
				Message: "Overall comment ratio is below recommended threshold", Value: 5, Threshold: 15},
		},
	}
}

func TestSARIF_DocumentShape(t *testing.T) {
	doc := reportSARIF(t, sampleResult(t.TempDir()))

	assert.Equal(t, "2.1.0", doc.Version)
	assert.Equal(t, "https://json.schemastore.org/sarif-2.1.0.json", doc.Schema)
	driver := doc.Runs[0].Tool.Driver
	assert.Equal(t, "code-review-assistant", driver.Name)
	assert.Equal(t, "9.9.9", driver.Version)
	assert.Equal(t, "https://github.com/daniel-munoz/code-review-assistant", driver.InformationURI)

	require.Len(t, doc.results(), 3, "project-level findings (no file) are omitted")

	params := findResult(t, doc, "too-many-parameters")
	assert.Equal(t, "warning", params.Level)
	assert.Equal(t, "Function has too many parameters: F (7 > 5)", params.Message.Text)
	assert.Equal(t, "pkg/a.go", params.uri())
	assert.Equal(t, 12, params.line())

	large := findResult(t, doc, "large-file")
	assert.Equal(t, "note", large.Level)
	assert.Equal(t, "File exceeds recommended size (900 > 500)", large.Message.Text)
	assert.Equal(t, 1, large.line(), "file-level findings are anchored at line 1")

	assert.Equal(t, "error", findResult(t, doc, "high-complexity").Level)
}

func TestSARIF_LevelMapping(t *testing.T) {
	assert.Equal(t, "error", sarifLevel("error"))
	assert.Equal(t, "warning", sarifLevel("warning"))
	assert.Equal(t, "note", sarifLevel("info"))
	assert.Equal(t, "warning", sarifLevel("something-new"))
}

func TestSARIF_RuleCatalogCoversFileLevelIssueTypes(t *testing.T) {
	// Every issue type CRA emits with a file. Project-level types are never in SARIF.
	fileLevel := []string{
		"large_file", "long_function", "high_complexity",
		"too_many_parameters", "deep_nesting", "too_many_returns",
		"magic_number", "duplicate_error_handling",
		"non_null_assertion", "run_blocking",
	}
	doc := reportSARIF(t, &analyzer.AnalysisResult{ProjectPath: t.TempDir(), Metrics: &analyzer.AggregateMetrics{}})
	rules := map[string]string{}
	for _, r := range doc.Runs[0].Tool.Driver.Rules {
		rules[r.ID] = r.ShortDescription.Text
		assert.NotEmpty(t, r.DefaultConfiguration.Level, "rule %s needs a default level", r.ID)
	}
	for _, typ := range fileLevel {
		id := strings.ReplaceAll(typ, "_", "-")
		assert.NotEmpty(t, rules[id], "rule catalog is missing %s", id)
	}
}

func TestSARIF_UnknownIssueTypeGetsGenericRule(t *testing.T) {
	project := t.TempDir()
	doc := reportSARIF(t, &analyzer.AnalysisResult{
		ProjectPath: project,
		Metrics:     &analyzer.AggregateMetrics{},
		Issues:      []*analyzer.Issue{{Severity: "warning", Type: "brand_new_check", File: filepath.Join(project, "a.go"), Line: 3, Message: "New check"}},
	})

	r := findResult(t, doc, "brand-new-check")
	assert.Equal(t, "New check", r.Message.Text, "no value/threshold suffix when both are zero")
	found := false
	for _, rule := range doc.Runs[0].Tool.Driver.Rules {
		found = found || rule.ID == "brand-new-check"
	}
	assert.True(t, found, "every ruleId used by a result must be declared")
}

func TestSARIF_EmptyResultIsValidDocument(t *testing.T) {
	out := filepath.Join(t.TempDir(), "cra.sarif")
	r, err := NewReporter(&config.OutputConfig{Format: "sarif", OutputFile: out})
	require.NoError(t, err)
	require.NoError(t, r.Report(emptyDiffResult(), nil))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"results": []`, "results must be an empty array, not null")
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput()
	require.NoError(t, err, "%s", out)
}

func TestSARIF_PathsAreRelativeToRepoRoot(t *testing.T) {
	repo := t.TempDir() // on macOS this is a symlinked /var path; git reports /private/var
	gitInit(t, repo)
	target := filepath.Join(repo, "services", "api")
	require.NoError(t, os.MkdirAll(target, 0o755))

	doc := reportSARIF(t, sampleResult(target))
	assert.Equal(t, "services/api/pkg/a.go", findResult(t, doc, "too-many-parameters").uri())
}

func TestSARIF_RelativeTargetFromSubdirectory(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)
	sub := filepath.Join(repo, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	t.Chdir(sub) // CLI default target "." from inside a subdirectory

	doc := reportSARIF(t, sampleResult("."))
	assert.Equal(t, "sub/pkg/a.go", findResult(t, doc, "too-many-parameters").uri())
}

func TestSARIF_OutsideGitPathsAreRelativeToTarget(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	doc := reportSARIF(t, sampleResult(t.TempDir()))
	assert.Equal(t, "pkg/a.go", findResult(t, doc, "too-many-parameters").uri())
}

func TestSARIF_WritesOnlyTheDocumentToStdout(t *testing.T) {
	r := NewSARIFReporter(&config.OutputConfig{Format: "sarif"})
	out := captureStdout(t, func() { require.NoError(t, r.Report(sampleResult(t.TempDir()), nil)) })

	var doc sarifDoc
	require.NoError(t, json.Unmarshal([]byte(out), &doc), "stdout must be exactly one SARIF document:\n%s", out)
	assert.Equal(t, "2.1.0", doc.Version)
}

func TestNewReporter_ListsSarifInError(t *testing.T) {
	_, err := NewReporter(&config.OutputConfig{Format: "xml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sarif")
}
