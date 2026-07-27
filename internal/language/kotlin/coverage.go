package kotlin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
// falls back to a gradle binary on PATH. The returned path is always
// absolute, so callers may exec it with cmd.Dir set to the project root
// regardless of whether projectPath was relative.
func detectGradleCommand(projectPath string) (string, error) {
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve project path %s: %w", projectPath, err)
	}

	wrappers := []string{"gradlew", "gradlew.bat"}
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
		return &gradleTask{name: "koverXmlReport", args: []string{"koverXmlReport"}}, nil
	}
	if strings.Contains(content, "jacoco") {
		return &gradleTask{name: "jacocoTestReport", args: []string{"test", "jacocoTestReport"}}, nil
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
