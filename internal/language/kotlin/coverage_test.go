package kotlin

import (
	"os"
	"path/filepath"
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

	reports := findReports(dir)

	assert.ElementsMatch(t, []string{
		filepath.Join(dir, "build", "reports", "kover", "report.xml"),
		filepath.Join(dir, "app", "build", "reports", "kover", "report.xml"),
	}, reports)
}

func TestFindReports_JaCoCo(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"), "<report/>")

	reports := findReports(dir)

	assert.Equal(t, []string{
		filepath.Join(dir, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
	}, reports)
}

func TestFindReports_NoneFound(t *testing.T) {
	assert.Empty(t, findReports(t.TempDir()))
}

// TestRunCoverage_EndToEnd fakes Gradle with a shell script that writes a
// Kover report, exercising the full detect -> run -> parse pipeline without
// a real Gradle installation.
func TestRunCoverage_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	writeGradleFile(t, filepath.Join(dir, "build.gradle.kts"), `id("org.jetbrains.kotlinx.kover")`)

	reportPath := filepath.Join(dir, "build", "reports", "kover", "report.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
	script := "#!/bin/sh\ncat > " + reportPath + " <<'EOF'\n" +
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
