package kotlin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// koverPluginID is the Gradle plugin id for Kover, JetBrains' Kotlin-first
// coverage tool. Its XML report is JaCoCo-format.
const koverPluginID = "org.jetbrains.kotlinx.kover"

// gradleTask is the Gradle invocation that produces a coverage report.
type gradleTask struct {
	name string   // for status/error messages, e.g. "koverXmlReport"
	args []string // Gradle CLI arguments
}

// detectGradleCommand locates the Gradle wrapper in the project root, or
// falls back to a gradle binary on PATH.
func detectGradleCommand(projectPath string) (string, error) {
	for _, wrapper := range []string{"gradlew", "gradlew.bat"} {
		path := filepath.Join(projectPath, wrapper)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
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
		return &gradleTask{name: "koverXmlReport", args: []string{"koverXmlReport"}}, nil
	}
	if strings.Contains(content, "jacoco") {
		return &gradleTask{name: "jacocoTestReport", args: []string{"test", "jacocoTestReport"}}, nil
	}
	return nil, fmt.Errorf("no coverage plugin detected in Gradle build files; apply Kover (%s) or JaCoCo to enable coverage", koverPluginID)
}

// readGradleBuildFiles concatenates the root and first-level subproject
// Gradle build/settings files. Missing files are simply skipped.
func readGradleBuildFiles(projectPath string) string {
	patterns := []string{
		"build.gradle", "build.gradle.kts",
		"settings.gradle", "settings.gradle.kts",
		"*/build.gradle", "*/build.gradle.kts",
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
