package kotlin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// koverPluginID is the Gradle plugin id for Kover, JetBrains' Kotlin-first
// coverage tool. Its XML report is JaCoCo-format.
const koverPluginID = "org.jetbrains.kotlinx.kover"

// gradlePipeDrainDelay bounds how long runGradle waits for a killed Gradle's
// orphaned children (e.g. the daemon JVM) to release our stdout/stderr pipe
// after the process itself has exited or been cancelled.
const gradlePipeDrainDelay = 2 * time.Second

// Report locations per tool, for the root project and first-level
// subprojects (multi-module builds).
var (
	koverReportGlobs = []string{
		filepath.Join("build", "reports", "kover", "report.xml"),
		filepath.Join("*", "build", "reports", "kover", "report.xml"),
	}
	jacocoReportGlobs = []string{
		filepath.Join("build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
		filepath.Join("*", "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
	}
)

// gradleTask is the Gradle invocation that produces a coverage report.
type gradleTask struct {
	name        string   // for status/error messages, e.g. "koverXmlReport"
	args        []string // Gradle CLI arguments
	reportGlobs []string // where this tool writes its XML reports
}

// detectGradleCommand locates the Gradle wrapper in the project root, or
// falls back to a gradle binary on PATH. The returned path is always
// absolute, so callers may exec it with cmd.Dir set to the project root
// regardless of whether projectPath was relative.
func detectGradleCommand(projectPath string) (string, error) {
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve project path %s: %w", projectPath, err)
	}

	wrappers := []string{"gradlew"}
	if runtime.GOOS == "windows" {
		wrappers = []string{"gradlew.bat", "gradlew"}
	}
	for _, wrapper := range wrappers {
		path := filepath.Join(root, wrapper)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("found %s but it is not executable; run: chmod +x %s", path, path)
		}
		return path, nil
	}
	if path, err := exec.LookPath("gradle"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("no Gradle wrapper (gradlew) in %s and no gradle binary on PATH; add the wrapper or install Gradle", projectPath)
}

// detectCoverageTask determines which coverage plugin the build applies.
// Kover wins when both are present (it is Kotlin-aware). The scan covers the
// root and one level of subprojects, matching common multi-module layouts.
func detectCoverageTask(projectPath string) (*gradleTask, error) {
	content := readGradleBuildFiles(projectPath)

	if strings.Contains(content, koverPluginID) {
		return &gradleTask{name: "koverXmlReport", args: []string{"koverXmlReport"}, reportGlobs: koverReportGlobs}, nil
	}
	if strings.Contains(content, "jacoco") {
		return &gradleTask{name: "jacocoTestReport", args: []string{"test", "jacocoTestReport"}, reportGlobs: jacocoReportGlobs}, nil
	}
	return nil, fmt.Errorf("no coverage plugin detected in Gradle build files; apply Kover (%s) or JaCoCo to enable coverage", koverPluginID)
}

// readGradleBuildFiles concatenates the root and first-level subproject
// Gradle build/settings files, plus the version catalog (gradle/libs.versions.toml):
// modern builds declare plugins there and apply them via
// alias(libs.plugins.kover), so the plugin id never appears in the build
// file itself. Missing files are simply skipped.
func readGradleBuildFiles(projectPath string) string {
	patterns := []string{
		"build.gradle", "build.gradle.kts",
		"settings.gradle", "settings.gradle.kts",
		"*/build.gradle", "*/build.gradle.kts",
		filepath.Join("gradle", "libs.versions.toml"),
	}

	var sb strings.Builder
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(projectPath, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if data, err := os.ReadFile(match); err == nil {
				sb.Write(data)
				sb.WriteByte('\n')
			}
		}
	}
	return sb.String()
}

// CoverageRunner executes Kotlin test coverage via Gradle (Kover or JaCoCo)
// and parses the resulting JaCoCo-format XML reports.
type CoverageRunner struct {
	timeout time.Duration
	status  status.Reporter
}

// NewCoverageRunner creates a new Kotlin coverage runner.
func NewCoverageRunner(timeoutSeconds int, statusReporter status.Reporter) *CoverageRunner {
	return &CoverageRunner{
		timeout: time.Duration(timeoutSeconds) * time.Second,
		status:  statusReporter,
	}
}

// RunCoverage runs the Gradle coverage task and parses the reports.
// excludePatterns is unused: exclusions are governed by the Gradle build
// itself (mirroring the JS runner, which also ignores them).
func (r *CoverageRunner) RunCoverage(projectPath string, excludePatterns []string) ([]*coverage.PackageCoverage, error) {
	r.status.Update("[COVERAGE] Detecting Gradle build...")
	gradleCmd, err := detectGradleCommand(projectPath)
	if err != nil {
		return nil, err
	}
	task, err := detectCoverageTask(projectPath)
	if err != nil {
		return nil, err
	}

	r.status.Update(fmt.Sprintf("[COVERAGE] Running Gradle task %s...", task.name))
	if err := r.runGradle(projectPath, gradleCmd, task); err != nil {
		return nil, err
	}

	r.status.Update("[COVERAGE] Parsing coverage reports...")
	reports := findReports(projectPath, task.reportGlobs)
	if len(reports) == 0 {
		return nil, fmt.Errorf("no coverage report found after Gradle run; check the %s report task configuration", task.name)
	}
	return parseReports(reports)
}

// runGradle executes the Gradle coverage task with a timeout. Failures embed
// the task name and a bounded excerpt of Gradle's combined output, because
// plugin detection is heuristic and the output is what makes its false
// positives ("Task 'jacocoTestReport' not found") diagnosable. A run that
// hits the timeout surfaces its error only after timeout + gradlePipeDrainDelay,
// since CombinedOutput waits up to gradlePipeDrainDelay for orphaned children
// to release the output pipe before giving up.
func (r *CoverageRunner) runGradle(projectPath, command string, task *gradleTask) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, task.args...)
	cmd.Dir = projectPath
	// A killed Gradle can leave children (the JVM/daemon) holding our stdout
	// pipe; without a WaitDelay, CombinedOutput would block until they exit,
	// making the timeout error arrive arbitrarily late.
	cmd.WaitDelay = gradlePipeDrainDelay

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil {
			// The Gradle process itself exited successfully; only an orphaned
			// child (e.g. the Gradle daemon) held our pipe past the drain delay.
			// Output may be truncated, but we only need the report files.
			return nil
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gradle %s timed out after %s", task.name, r.timeout)
		}
		out := string(output)
		if strings.Contains(out, "not found in root project") || strings.Contains(out, "not found in project") {
			return fmt.Errorf("gradle has no task %q — the coverage plugin was detected from a mention in the Gradle build files but may not actually be applied (e.g. a comment or dependency coordinate): %w\n%s",
				task.name, err, outputTail(out))
		}
		if tail := outputTail(out); tail != "" {
			return fmt.Errorf("gradle %s failed: %w\n%s", task.name, err, tail)
		}
		return fmt.Errorf("gradle %s failed: %w", task.name, err)
	}
	return nil
}

// outputTail returns the last portion of Gradle output for error messages.
func outputTail(s string) string {
	const maxTail = 2000
	if len(s) <= maxTail {
		return s
	}
	return "..." + s[len(s)-maxTail:]
}

// findReports globs the given report-location patterns (relative to
// projectPath) for the selected coverage tool only. Restricting the glob to
// the tool that actually ran keeps a stale report left behind by the other
// tool (e.g. a leftover jacocoTestReport.xml after migrating to Kover) from
// silently blending into the results.
func findReports(projectPath string, patterns []string) []string {
	var reports []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(projectPath, pattern))
		if err != nil {
			continue
		}
		reports = append(reports, matches...)
	}
	return reports
}
