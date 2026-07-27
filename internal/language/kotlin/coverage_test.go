package kotlin

import (
	"os"
	"path/filepath"
	"testing"

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
