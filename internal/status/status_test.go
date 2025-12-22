package status

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReporter(t *testing.T) {
	t.Run("returns SilentReporter when QuietMode is true", func(t *testing.T) {
		cfg := &config.OutputConfig{
			QuietMode: true,
		}

		reporter := NewReporter(cfg)

		assert.IsType(t, &SilentReporter{}, reporter, "should return SilentReporter in quiet mode")
	})

	t.Run("returns TTYReporter when ShowStatus is true", func(t *testing.T) {
		cfg := &config.OutputConfig{
			ShowStatus: true,
		}

		reporter := NewReporter(cfg)

		assert.IsType(t, &TTYReporter{}, reporter, "should return TTYReporter when forced")
	})

	t.Run("returns SilentReporter when OutputFile is set", func(t *testing.T) {
		cfg := &config.OutputConfig{
			OutputFile: "/tmp/output.txt",
		}

		reporter := NewReporter(cfg)

		assert.IsType(t, &SilentReporter{}, reporter, "should return SilentReporter when output is redirected to file")
	})

	t.Run("QuietMode takes precedence over ShowStatus", func(t *testing.T) {
		cfg := &config.OutputConfig{
			QuietMode:  true,
			ShowStatus: true,
		}

		reporter := NewReporter(cfg)

		assert.IsType(t, &SilentReporter{}, reporter, "QuietMode should take precedence")
	})

	t.Run("auto-detects based on TTY when no flags set", func(t *testing.T) {
		cfg := &config.OutputConfig{
			QuietMode:  false,
			ShowStatus: false,
			OutputFile: "",
		}

		reporter := NewReporter(cfg)

		// Result depends on whether stderr is a TTY, but should be one of the two types
		_, isTTY := reporter.(*TTYReporter)
		_, isSilent := reporter.(*SilentReporter)
		assert.True(t, isTTY || isSilent, "should return either TTYReporter or SilentReporter")
	})
}

func TestSilentReporter(t *testing.T) {
	reporter := NewSilentReporter()

	t.Run("all methods are no-ops", func(t *testing.T) {
		// These should not panic
		reporter.Start()
		reporter.Update("test message")
		reporter.UpdateProgress("prefix", 1, 10, "detail")
		reporter.Clear()
		reporter.Stop()
	})

	t.Run("is not nil", func(t *testing.T) {
		require.NotNil(t, reporter, "SilentReporter should not be nil")
	})
}

func TestTTYReporter(t *testing.T) {
	reporter := NewTTYReporter()

	t.Run("initializes with zero lastMessageLen", func(t *testing.T) {
		require.NotNil(t, reporter, "TTYReporter should not be nil")
		assert.Equal(t, 0, reporter.lastMessageLen, "should start with zero message length")
	})

	t.Run("Start and Stop do not panic", func(t *testing.T) {
		reporter.Start()
		reporter.Stop()
	})

	t.Run("Update sets lastMessageLen", func(t *testing.T) {
		reporter := NewTTYReporter()
		message := "test message"

		reporter.Update(message)

		assert.Equal(t, len(message), reporter.lastMessageLen, "should track message length")
	})

	t.Run("UpdateProgress formats correctly", func(t *testing.T) {
		reporter := NewTTYReporter()

		reporter.UpdateProgress("[TEST] Doing work", 5, 10, "item.go")

		// Should have tracked the formatted message length
		assert.Greater(t, reporter.lastMessageLen, 0, "should have non-zero message length")
	})

	t.Run("Clear resets lastMessageLen", func(t *testing.T) {
		reporter := NewTTYReporter()

		reporter.Update("some message")
		assert.Greater(t, reporter.lastMessageLen, 0, "should have message length after Update")

		reporter.Clear()
		assert.Equal(t, 0, reporter.lastMessageLen, "should reset message length after Clear")
	})

	t.Run("multiple updates track length correctly", func(t *testing.T) {
		reporter := NewTTYReporter()

		reporter.Update("short")
		firstLen := reporter.lastMessageLen

		reporter.Update("much longer message here")
		secondLen := reporter.lastMessageLen

		assert.NotEqual(t, firstLen, secondLen, "should track different message lengths")
		assert.Greater(t, secondLen, firstLen, "longer message should have greater length")
	})
}

func TestTestReporter(t *testing.T) {
	reporter := NewTestReporter()

	t.Run("initializes with empty state", func(t *testing.T) {
		require.NotNil(t, reporter, "TestReporter should not be nil")
		assert.Empty(t, reporter.GetMessages(), "should start with no messages")
		assert.False(t, reporter.WasStarted(), "should not be started initially")
		assert.False(t, reporter.WasStopped(), "should not be stopped initially")
	})

	t.Run("Start sets started flag", func(t *testing.T) {
		reporter := NewTestReporter()

		reporter.Start()

		assert.True(t, reporter.WasStarted(), "should be marked as started")
	})

	t.Run("Stop sets stopped flag", func(t *testing.T) {
		reporter := NewTestReporter()

		reporter.Stop()

		assert.True(t, reporter.WasStopped(), "should be marked as stopped")
	})

	t.Run("Update captures messages", func(t *testing.T) {
		reporter := NewTestReporter()

		reporter.Update("message 1")
		reporter.Update("message 2")

		messages := reporter.GetMessages()
		require.Len(t, messages, 2, "should capture 2 messages")
		assert.Equal(t, "message 1", messages[0])
		assert.Equal(t, "message 2", messages[1])
	})

	t.Run("UpdateProgress captures formatted messages", func(t *testing.T) {
		reporter := NewTestReporter()

		reporter.UpdateProgress("[TEST] Work", 3, 10, "file.go")

		messages := reporter.GetMessages()
		require.Len(t, messages, 1, "should capture 1 message")
		assert.Contains(t, messages[0], "[TEST] Work")
		assert.Contains(t, messages[0], "(3/10)")
		assert.Contains(t, messages[0], "file.go")
	})

	t.Run("Clear captures clear operation", func(t *testing.T) {
		reporter := NewTestReporter()

		reporter.Update("message")
		reporter.Clear()

		messages := reporter.GetMessages()
		require.Len(t, messages, 2, "should capture message and clear")
		assert.Equal(t, "message", messages[0])
		assert.Equal(t, "[CLEAR]", messages[1])
	})

	t.Run("captures full workflow", func(t *testing.T) {
		reporter := NewTestReporter()

		reporter.Start()
		reporter.Update("starting")
		reporter.UpdateProgress("[PARSE] Files", 1, 5, "test.go")
		reporter.UpdateProgress("[PARSE] Files", 5, 5, "final.go")
		reporter.Clear()
		reporter.Stop()

		assert.True(t, reporter.WasStarted())
		assert.True(t, reporter.WasStopped())

		messages := reporter.GetMessages()
		require.Len(t, messages, 4, "should capture all messages")
		assert.Equal(t, "starting", messages[0])
		assert.Contains(t, messages[1], "(1/5)")
		assert.Contains(t, messages[2], "(5/5)")
		assert.Equal(t, "[CLEAR]", messages[3])
	})
}

func TestReporterInterface(t *testing.T) {
	t.Run("SilentReporter implements Reporter", func(t *testing.T) {
		var _ Reporter = (*SilentReporter)(nil)
	})

	t.Run("TTYReporter implements Reporter", func(t *testing.T) {
		var _ Reporter = (*TTYReporter)(nil)
	})

	t.Run("TestReporter implements Reporter", func(t *testing.T) {
		var _ Reporter = (*TestReporter)(nil)
	})
}
