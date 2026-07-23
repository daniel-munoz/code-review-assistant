package language_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/language"
	_ "github.com/daniel-munoz/code-review-assistant/internal/language/golang" // Register Go language
	_ "github.com/daniel-munoz/code-review-assistant/internal/language/kotlin" // Register Kotlin language
	_ "github.com/daniel-munoz/code-review-assistant/internal/language/python" // Register Python language
)

func TestGet(t *testing.T) {
	t.Run("returns registered Go language", func(t *testing.T) {
		lang, ok := language.Get("go")
		assert.True(t, ok, "go language should be registered")
		assert.NotNil(t, lang)
		assert.Equal(t, "go", lang.Name())
		assert.Equal(t, "Go", lang.DisplayName())
	})

	t.Run("returns registered Python language", func(t *testing.T) {
		lang, ok := language.Get("python")
		assert.True(t, ok, "python language should be registered")
		assert.NotNil(t, lang)
		assert.Equal(t, "python", lang.Name())
		assert.Equal(t, "Python", lang.DisplayName())
	})

	t.Run("returns false for unregistered language", func(t *testing.T) {
		lang, ok := language.Get("nonexistent")
		assert.False(t, ok, "nonexistent language should not be found")
		assert.Nil(t, lang)
	})

	t.Run("returns false for empty name", func(t *testing.T) {
		lang, ok := language.Get("")
		assert.False(t, ok)
		assert.Nil(t, lang)
	})
}

func TestAll(t *testing.T) {
	t.Run("returns all registered languages", func(t *testing.T) {
		languages := language.All()

		// Should have at least go and python
		assert.GreaterOrEqual(t, len(languages), 2, "should have at least go and python")

		// Collect names
		names := make([]string, len(languages))
		for i, lang := range languages {
			names[i] = lang.Name()
		}
		sort.Strings(names)

		assert.Contains(t, names, "go")
		assert.Contains(t, names, "python")
	})
}

func TestSupportedLanguages(t *testing.T) {
	t.Run("returns names of registered languages", func(t *testing.T) {
		names := language.SupportedLanguages()

		assert.GreaterOrEqual(t, len(names), 2, "should have at least go and python")
		assert.Contains(t, names, "go")
		assert.Contains(t, names, "python")
	})
}

func TestDetectLanguage(t *testing.T) {
	t.Run("detects Go project via go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "go", lang.Name())
	})

	t.Run("detects Python project via requirements.txt", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create requirements.txt
		err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("requests>=2.0\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "python", lang.Name())
	})

	t.Run("detects Python project via pyproject.toml", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create pyproject.toml
		content := `[build-system]
requires = ["setuptools"]
build-backend = "setuptools.build_meta"
`
		err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(content), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "python", lang.Name())
	})

	t.Run("detects Python project via setup.py", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create setup.py
		content := `from setuptools import setup
setup(name="test")
`
		err := os.WriteFile(filepath.Join(tmpDir, "setup.py"), []byte(content), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "python", lang.Name())
	})

	t.Run("Go takes priority over Python when both present", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create both go.mod and requirements.txt
		err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("requests\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "go", lang.Name(), "Go should take priority")
	})

	t.Run("detects Kotlin project via build.gradle.kts", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "build.gradle.kts"), []byte("plugins { kotlin(\"jvm\") }\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "kotlin", lang.Name())
	})

	t.Run("detects Kotlin project via settings.gradle.kts", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "settings.gradle.kts"), []byte("rootProject.name = \"svc\"\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "kotlin", lang.Name())
	})

	t.Run("detects Kotlin project via Groovy build.gradle", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte("apply plugin: 'kotlin'\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "kotlin", lang.Name())
	})

	t.Run("Go takes priority over Kotlin when both present", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "build.gradle.kts"), []byte("plugins { }\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "go", lang.Name(), "existing manifest priority should be preserved")
	})

	t.Run("settings.gradle.kts takes priority among Gradle manifests", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "settings.gradle.kts"), []byte("rootProject.name = \"svc\"\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte("apply plugin: 'kotlin'\n"), 0644)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "kotlin", lang.Name())
	})

	t.Run("returns error when no manifest found", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Empty directory
		lang, err := language.DetectLanguage(tmpDir)
		assert.Error(t, err)
		assert.Nil(t, lang)
		assert.Contains(t, err.Error(), "could not detect language")
		assert.Contains(t, err.Error(), "no supported manifest files found")
	})

	t.Run("returns error for nonexistent directory", func(t *testing.T) {
		lang, err := language.DetectLanguage("/nonexistent/path/that/does/not/exist")
		assert.Error(t, err)
		assert.Nil(t, lang)
	})

	t.Run("ignores directories with manifest names", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a directory named go.mod (not a file)
		err := os.Mkdir(filepath.Join(tmpDir, "go.mod"), 0755)
		require.NoError(t, err)

		lang, err := language.DetectLanguage(tmpDir)
		assert.Error(t, err, "should not detect directory as manifest")
		assert.Nil(t, lang)
	})
}
