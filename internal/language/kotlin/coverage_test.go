package kotlin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGradleFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDetectCoverageTask_Kover(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `
plugins {
    kotlin("jvm") version "2.0.0"
    id("org.jetbrains.kotlinx.kover") version "0.9.1"
}
`)

	task, err := detectCoverageTask(dir)
	require.NoError(t, err)
	assert.Equal(t, "koverXmlReport", task.name)
	assert.Equal(t, []string{"koverXmlReport"}, task.args)
}

func TestDetectCoverageTask_JaCoCo(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle"), `
plugins {
    id 'org.jetbrains.kotlin.jvm' version '2.0.0'
    id 'jacoco'
}
`)

	task, err := detectCoverageTask(dir)
	require.NoError(t, err)
	assert.Equal(t, "jacocoTestReport", task.name)
	assert.Equal(t, []string{"test", "jacocoTestReport"}, task.args)
}

func TestDetectCoverageTask_KoverWinsOverJaCoCo(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `
plugins {
    id("org.jetbrains.kotlinx.kover") version "0.9.1"
    jacoco
}
`)

	task, err := detectCoverageTask(dir)
	require.NoError(t, err)
	assert.Equal(t, "koverXmlReport", task.name)
}

func TestDetectCoverageTask_SubprojectBuildFile(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "settings.gradle.kts"), `include(":app")`)
	writeGradleFile(t, filepath.Join(dir, "app", "build.gradle.kts"), `
plugins {
    id("org.jetbrains.kotlinx.kover")
}
`)

	task, err := detectCoverageTask(dir)
	require.NoError(t, err)
	assert.Equal(t, "koverXmlReport", task.name)
}

func TestDetectCoverageTask_NoPluginErrors(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `
plugins {
    kotlin("jvm")
}
`)

	_, err := detectCoverageTask(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Kover")
	assert.Contains(t, err.Error(), "JaCoCo")
}

func TestDetectCoverageTask_KoverViaVersionCatalog(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `
plugins {
    alias(libs.plugins.kover)
}
`)
	writeGradleFile(t, filepath.Join(dir, "gradle", "libs.versions.toml"), `
[plugins]
kover = { id = "org.jetbrains.kotlinx.kover", version = "0.9.1" }
`)

	task, err := detectCoverageTask(dir)
	require.NoError(t, err)
	assert.Equal(t, "koverXmlReport", task.name)
}

func TestDetectGradleCommand_PrefersWrapperOverPathGradle(t *testing.T) {
	// Put a stub gradle on PATH so there is genuinely something to prefer over.
	binDir := t.TempDir()
	writeGradleFile(t, filepath.Join(binDir, "gradle"), "#!/bin/sh\n")
	require.NoError(t, os.Chmod(filepath.Join(binDir, "gradle"), 0o755))
	t.Setenv("PATH", binDir)

	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	cmd, err := detectGradleCommand(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "gradlew"), cmd)
	assert.True(t, filepath.IsAbs(cmd), "callers exec with cmd.Dir set; the path must be absolute")
}

func TestDetectGradleCommand_RelativeProjectPathYieldsAbsoluteWrapper(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))
	t.Chdir(dir)

	cmd, err := detectGradleCommand(".")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(cmd), "relative projectPath must still yield an absolute wrapper path")
	assert.FileExists(t, cmd)
}

func TestDetectGradleCommand_FallsBackToPathGradle(t *testing.T) {
	binDir := t.TempDir()
	stub := filepath.Join(binDir, "gradle")
	writeGradleFile(t, stub, "#!/bin/sh\n")
	require.NoError(t, os.Chmod(stub, 0o755))
	t.Setenv("PATH", binDir)

	cmd, err := detectGradleCommand(t.TempDir()) // no wrapper in the project
	require.NoError(t, err)
	assert.Equal(t, stub, cmd)
	assert.True(t, filepath.IsAbs(cmd))
}

func TestDetectGradleCommand_BatOnlyCheckoutFallsThroughOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows behavior")
	}
	t.Setenv("PATH", t.TempDir()) // no gradle on PATH either

	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "gradlew.bat"), "@echo off\r\n")

	_, err := detectGradleCommand(dir)
	require.Error(t, err, "a .bat wrapper is not executable on Unix; expect the no-Gradle error, not chmod advice")
	assert.NotContains(t, err.Error(), "chmod")
}

func TestDetectGradleCommand_NonExecutableWrapperErrors(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\n") // 0o644, not executable

	_, err := detectGradleCommand(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chmod +x")
}

func TestDetectGradleCommand_NoWrapperNoPathGradleErrors(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir: no gradle binary anywhere

	_, err := detectGradleCommand(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gradlew")
}

func TestFindReports_KoverAndMultiModule(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build", "reports", "kover", "report.xml"), "<report/>")
	writeGradleFile(t, filepath.Join(dir, "app", "build", "reports", "kover", "report.xml"), "<report/>")
	// Stale report from the other tool: must never be returned when
	// searching for Kover's locations.
	writeGradleFile(t, filepath.Join(dir, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"), "<report/>")

	reports := findReports(dir, koverReportGlobs)

	assert.ElementsMatch(t, []string{
		filepath.Join(dir, "build", "reports", "kover", "report.xml"),
		filepath.Join(dir, "app", "build", "reports", "kover", "report.xml"),
	}, reports)
}

func TestFindReports_JaCoCo(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"), "<report/>")

	reports := findReports(dir, jacocoReportGlobs)

	assert.Equal(t, []string{
		filepath.Join(dir, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
	}, reports)
}

func TestFindReports_NoneFound(t *testing.T) {
	assert.Empty(t, findReports(t.TempDir(), koverReportGlobs))
}

// TestRunCoverage_EndToEnd fakes Gradle with a shell script that writes a
// Kover report, exercising the full detect -> run -> parse pipeline without
// a real Gradle installation.
func TestRunCoverage_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)

	reportPath := filepath.Join(dir, "build", "reports", "kover", "report.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
	script := "#!/bin/sh\ncat > \"" + reportPath + "\" <<'EOF'\n" +
		`<?xml version="1.0" encoding="UTF-8"?>
<report name="fake">
  <package name="com/example/alpha">
    <counter type="LINE" missed="5" covered="15"/>
  </package>
</report>` + "\nEOF\n"
	writeGradleFile(t, filepath.Join(dir, "gradlew"), script)
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	results, err := runner.RunCoverage(dir, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "com.example.alpha", results[0].PackagePath)
	assert.InDelta(t, 75.0, results[0].Coverage, 0.001)
}

func TestRunCoverage_StaleOtherToolReportIgnored(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)

	// Stale JaCoCo report from before a migration to Kover, with divergent numbers.
	writeGradleFile(t, filepath.Join(dir, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
		`<?xml version="1.0"?><report name="stale"><package name="com/example/alpha"><counter type="LINE" missed="100" covered="0"/></package></report>`)

	reportPath := filepath.Join(dir, "build", "reports", "kover", "report.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
	script := "#!/bin/sh\ncat > \"" + reportPath + "\" <<'EOF'\n" +
		`<?xml version="1.0" encoding="UTF-8"?>
<report name="fake">
  <package name="com/example/alpha">
    <counter type="LINE" missed="5" covered="15"/>
  </package>
</report>` + "\nEOF\n"
	writeGradleFile(t, filepath.Join(dir, "gradlew"), script)
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	results, err := runner.RunCoverage(dir, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.InDelta(t, 75.0, results[0].Coverage, 0.001,
		"stale JaCoCo report must not blend into Kover results")
}

func TestRunCoverage_SucceedsWhenDaemonHoldsPipe(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)

	reportPath := filepath.Join(dir, "build", "reports", "kover", "report.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
	// Exits 0 immediately but leaves a background child holding stdout,
	// like a spawned Gradle daemon; WaitDelay drains after ~2s and the
	// run must still be treated as a success.
	script := "#!/bin/sh\ncat > \"" + reportPath + "\" <<'EOF'\n" +
		`<?xml version="1.0" encoding="UTF-8"?>
<report name="fake">
  <package name="com/example/alpha">
    <counter type="LINE" missed="5" covered="15"/>
  </package>
</report>` + "\nEOF\n" +
		"sleep 4 &\nexit 0\n"
	writeGradleFile(t, filepath.Join(dir, "gradlew"), script)
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	results, err := runner.RunCoverage(dir, nil)
	require.NoError(t, err, "a successful build with a lingering daemon must not be reported as failure")
	require.Len(t, results, 1)
}

func TestRunCoverage_NoReportAfterRunErrors(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\nexit 0\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	_, err := runner.RunCoverage(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no coverage report found")
}

func TestRunCoverage_GradleFailureSurfacesOutput(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\necho 'BUILD FAILED: 3 tests failed' >&2\nexit 1\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	_, err := runner.RunCoverage(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BUILD FAILED")
	assert.Contains(t, err.Error(), "koverXmlReport", "errors must name the Gradle task")
}

func TestRunCoverage_TaskNotFoundGetsFriendlyHint(t *testing.T) {
	dir := t.TempDir()
	// "jacoco" mentioned but plugin not applied -> detection picks jacocoTestReport,
	// Gradle then fails at configuration time with "task not found".
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `// jacoco agent only, plugin not applied`)
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\necho \"Task 'jacocoTestReport' not found in root project 'demo'.\" >&2\nexit 1\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	_, err := runner.RunCoverage(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "may not actually be applied", "task-not-found should hint that detection can false-positive on comments/coordinates")
}

func TestRunCoverage_TimeoutSurfaced(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\nsleep 5\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(1, &status.SilentReporter{})
	_, err := runner.RunCoverage(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestRunCoverage_FailureWithNoOutputHasCleanMessage(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)
	writeGradleFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\nexit 1\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	runner := NewCoverageRunner(60, &status.SilentReporter{})
	_, err := runner.RunCoverage(dir, nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\n", "empty Gradle output must not leave a dangling newline")
}

func TestOutputTail_TruncatesLongOutput(t *testing.T) {
	long := strings.Repeat("x", 3000)

	tail := outputTail(long)

	assert.Len(t, tail, 2003, "3-char ellipsis prefix plus 2000-char tail")
	assert.True(t, strings.HasPrefix(tail, "..."))
}
