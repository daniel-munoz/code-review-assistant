package kotlin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDetectCoverageTask_Kover(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "build.gradle.kts"), `
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
	writeFile(t, filepath.Join(dir, "build.gradle"), `
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
	writeFile(t, filepath.Join(dir, "build.gradle.kts"), `
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
	writeFile(t, filepath.Join(dir, "settings.gradle.kts"), `include(":app")`)
	writeFile(t, filepath.Join(dir, "app", "build.gradle.kts"), `
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
	writeFile(t, filepath.Join(dir, "build.gradle.kts"), `
plugins {
    kotlin("jvm")
}
`)

	_, err := detectCoverageTask(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Kover")
	assert.Contains(t, err.Error(), "JaCoCo")
}

func TestDetectGradleCommand_PrefersWrapper(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "gradlew"), "#!/bin/sh\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "gradlew"), 0o755))

	cmd, err := detectGradleCommand(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "gradlew"), cmd)
}
