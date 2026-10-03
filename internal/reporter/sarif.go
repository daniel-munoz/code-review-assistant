package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/git"
)

const (
	sarifVersion = "2.1.0"
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	toolName     = "code-review-assistant"
	toolURI      = "https://github.com/daniel-munoz/code-review-assistant"
)

// sarifRuleCatalog describes every file-level issue type CRA emits. Rule IDs
// are the issue types with hyphens; they are part of the SARIF contract and
// must stay stable across releases. Project-level types (coverage,
// dependencies, comment ratio) have no file location and are not reported in
// SARIF.
var sarifRuleCatalog = map[string]struct {
	description string
	level       string
}{
	"large_file":               {"File exceeds the configured size threshold", "note"},
	"long_function":            {"Function exceeds the configured length threshold", "warning"},
	"high_complexity":          {"Function exceeds the configured cyclomatic complexity threshold", "warning"},
	"too_many_parameters":      {"Function has more parameters than the configured maximum", "warning"},
	"deep_nesting":             {"Code is nested deeper than the configured maximum", "warning"},
	"too_many_returns":         {"Function has more return statements than the configured maximum", "note"},
	"magic_number":             {"Numeric literal that should probably be a named constant", "note"},
	"duplicate_error_handling": {"Repeated identical error-handling blocks", "note"},
	"non_null_assertion":       {"Kotlin non-null assertion (!!) that can throw at runtime", "warning"},
	"run_blocking":             {"Kotlin runBlocking used outside a top-level main function", "warning"},
}

// SARIF 2.1.0 types (only the parts CRA emits).
type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string             `json:"id"`
	ShortDescription     sarifMessage       `json:"shortDescription"`
	DefaultConfiguration sarifConfiguration `json:"defaultConfiguration"`
}

type sarifConfiguration struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

// SARIFReporter implements Reporter for SARIF 2.1.0 output, consumed by
// GitHub code scanning (upload-sarif) and reviewdog (-f=sarif).
type SARIFReporter struct {
	config *config.OutputConfig
	output io.Writer // nil means os.Stdout at report time
}

// NewSARIFReporter creates a SARIFReporter writing to stdout or cfg.OutputFile.
func NewSARIFReporter(cfg *config.OutputConfig) *SARIFReporter {
	return &SARIFReporter{config: cfg}
}

// Report writes one SARIF document with a result per file-level issue.
// Comparison data has no SARIF representation and is ignored.
func (sr *SARIFReporter) Report(result *analyzer.AnalysisResult, _ *comparison.ComparisonResult) error {
	out := sr.output
	if sr.config.OutputFile != "" {
		file, err := CreateOutputFile(sr.config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		out = file
	} else if out == nil {
		out = os.Stdout
	}

	data, err := json.MarshalIndent(sr.buildLog(result), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}
	if _, err := out.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write SARIF output: %w", err)
	}
	return nil
}

// buildLog converts the analysis result into a SARIF log.
func (sr *SARIFReporter) buildLog(result *analyzer.AnalysisResult) sarifLog {
	base := uriBase(result.ProjectPath)
	results := make([]sarifResult, 0, len(result.Issues))
	extra := map[string]bool{} // issue types missing from the catalog

	for _, issue := range result.Issues {
		if issue.File == "" {
			continue // project-level finding: no location to annotate
		}
		if _, ok := sarifRuleCatalog[issue.Type]; !ok {
			extra[issue.Type] = true
		}
		line := issue.Line
		if line < 1 {
			line = 1
		}
		results = append(results, sarifResult{
			RuleID:  sarifRuleID(issue.Type),
			Level:   sarifLevel(issue.Severity),
			Message: sarifMessage{Text: sarifMessageText(issue)},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: relativeURI(issue.File, base)},
				Region:           sarifRegion{StartLine: line},
			}}},
		})
	}

	return sarifLog{
		Schema:  sarifSchema,
		Version: sarifVersion,
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           toolName,
				Version:        sr.config.ToolVersion,
				InformationURI: toolURI,
				Rules:          sarifRules(extra),
			}},
			Results: results,
		}},
	}
}

// sarifRules returns the catalog plus a generic rule for each extra type,
// sorted by ID so output is deterministic.
func sarifRules(extra map[string]bool) []sarifRule {
	rules := make([]sarifRule, 0, len(sarifRuleCatalog)+len(extra))
	for typ, r := range sarifRuleCatalog {
		rules = append(rules, sarifRule{
			ID:                   sarifRuleID(typ),
			ShortDescription:     sarifMessage{Text: r.description},
			DefaultConfiguration: sarifConfiguration{Level: r.level},
		})
	}
	for typ := range extra {
		rules = append(rules, sarifRule{
			ID:                   sarifRuleID(typ),
			ShortDescription:     sarifMessage{Text: strings.ReplaceAll(typ, "_", " ")},
			DefaultConfiguration: sarifConfiguration{Level: "warning"},
		})
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	return rules
}

// sarifRuleID turns an issue type into its stable SARIF rule ID.
func sarifRuleID(issueType string) string {
	return strings.ReplaceAll(issueType, "_", "-")
}

// sarifLevel maps CRA severities to SARIF levels.
func sarifLevel(severity string) string {
	switch severity {
	case "error":
		return "error"
	case "info":
		return "note"
	default:
		return "warning"
	}
}

// sarifMessageText adds the function and the value/threshold to the message,
// e.g. "Function has too many parameters: F (7 > 5)".
func sarifMessageText(issue *analyzer.Issue) string {
	text := issue.Message
	if issue.Function != "" {
		text += ": " + issue.Function
	}
	if issue.Value != 0 || issue.Threshold != 0 {
		text += fmt.Sprintf(" (%d > %d)", issue.Value, issue.Threshold)
	}
	return text
}

// uriBase returns the directory SARIF URIs are relative to: the root of the
// git work tree containing projectPath (so GitHub can match results to the
// PR diff), or projectPath itself outside git. Symlinks are resolved.
func uriBase(projectPath string) string {
	if root, err := git.TopLevel(projectPath); err == nil {
		return root
	}
	return resolvePath(projectPath)
}

// relativeURI returns file relative to base, slash-separated. If file is not
// under base it falls back to the path as given.
func relativeURI(file, base string) string {
	rel, err := filepath.Rel(base, resolvePath(file))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

// resolvePath makes p absolute and resolves symlinks in its deepest existing
// ancestor, so paths that no longer exist on disk still compare correctly
// against a symlink-resolved base (e.g. macOS /var vs /private/var).
func resolvePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	rest := ""
	for dir := abs; ; dir = filepath.Dir(dir) {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return abs
		}
		rest = filepath.Join(filepath.Base(dir), rest)
	}
}
