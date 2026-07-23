package language

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	// registry holds all registered language providers
	registry = make(map[string]Language)
	// registryMu protects concurrent access to the registry
	registryMu sync.RWMutex
)

// Register adds a language provider to the registry.
// This is typically called from init() functions in language packages.
// Panics if a language with the same name is already registered.
func Register(lang Language) {
	registryMu.Lock()
	defer registryMu.Unlock()

	name := lang.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("language already registered: %s", name))
	}
	registry[name] = lang
}

// Get retrieves a language provider by name.
// Returns the language and true if found, or nil and false if not found.
func Get(name string) (Language, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	lang, ok := registry[name]
	return lang, ok
}

// All returns all registered language providers.
func All() []Language {
	registryMu.RLock()
	defer registryMu.RUnlock()

	languages := make([]Language, 0, len(registry))
	for _, lang := range registry {
		languages = append(languages, lang)
	}
	return languages
}

// DetectLanguage auto-detects the primary language of a project.
//
// Detection is based on the presence of language-specific manifest files:
//   - go.mod → Go
//   - pyproject.toml, setup.py, requirements.txt → Python
//   - package.json, tsconfig.json → JavaScript/TypeScript
//   - build.gradle.kts, settings.gradle.kts, build.gradle, settings.gradle → Kotlin
//
// Returns an error if no supported language can be detected.
func DetectLanguage(projectPath string) (Language, error) {
	// Check for language-specific manifest files in priority order
	// Only includes languages that are currently implemented
	manifestChecks := []struct {
		file     string
		langName string
	}{
		// Go: go.mod
		{"go.mod", "go"},
		// Python: pyproject.toml, setup.py, or requirements.txt
		{"pyproject.toml", "python"},
		{"setup.py", "python"},
		{"requirements.txt", "python"},
		// JavaScript/TypeScript: package.json or tsconfig.json
		{"package.json", "javascript"},
		{"tsconfig.json", "javascript"},
		// Kotlin: Gradle build files (checked after other manifests; a Java-only
		// Gradle repo will also match - CRA has no Java provider to prefer)
		{"settings.gradle.kts", "kotlin"},
		{"build.gradle.kts", "kotlin"},
		{"settings.gradle", "kotlin"},
		{"build.gradle", "kotlin"},
	}

	for _, check := range manifestChecks {
		if fileExists(filepath.Join(projectPath, check.file)) {
			if lang, ok := Get(check.langName); ok {
				return lang, nil
			}
		}
	}

	return nil, fmt.Errorf("could not detect language in %s: no supported manifest files found (supported: Go, Python, JavaScript/TypeScript, Kotlin)", projectPath)
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// SupportedLanguages returns the names of all registered languages.
func SupportedLanguages() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
