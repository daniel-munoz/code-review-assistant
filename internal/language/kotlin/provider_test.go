package kotlin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func TestKotlinLanguage_Identity(t *testing.T) {
	lang := &KotlinLanguage{}

	assert.Equal(t, "kotlin", lang.Name())
	assert.Equal(t, "Kotlin", lang.DisplayName())
	assert.Equal(t, []string{".kt"}, lang.Extensions(), ".kts build scripts are deliberately excluded")
}

func TestKotlinLanguage_ExcludePatterns(t *testing.T) {
	patterns := (&KotlinLanguage{}).DefaultExcludePatterns()

	assert.Contains(t, patterns, "**/build/**")
	assert.Contains(t, patterns, "**/.gradle/**")
	assert.Contains(t, patterns, "**/src/test/**")
	assert.Contains(t, patterns, "**/*Test.kt")
}

func TestKotlinLanguage_Components(t *testing.T) {
	lang := &KotlinLanguage{}
	cfg := &config.AnalysisConfig{}

	require.NotNil(t, lang.Parser(1))
	require.NotNil(t, lang.DetectorRunner(cfg))

	assert.NotNil(t, lang.CoverageRunner(cfg, &status.SilentReporter{}), "Kotlin supports coverage via Gradle")

	da, err := lang.DependencyAnalyzer(".")
	assert.NoError(t, err)
	assert.NotNil(t, da, "Kotlin supports dependency analysis")
}
