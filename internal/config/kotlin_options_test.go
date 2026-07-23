package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault_KotlinDetectorOptions(t *testing.T) {
	cfg := Default()
	assert.True(t, cfg.Analysis.DetectNonNullAssertions, "non-null assertion detection should default to enabled")
	assert.True(t, cfg.Analysis.DetectRunBlocking, "runBlocking detection should default to enabled")
}

func TestMerge_KotlinDetectorOptions(t *testing.T) {
	cfg := Default()
	cfg.Merge(map[string]interface{}{
		"detect_non_null_assertions": false,
		"detect_run_blocking":        false,
	})
	assert.False(t, cfg.Analysis.DetectNonNullAssertions)
	assert.False(t, cfg.Analysis.DetectRunBlocking)
}
