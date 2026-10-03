package git

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopLevel_FromSubdirectory(t *testing.T) {
	dir := newRepo(t, map[string]string{"sub/deep/a.go": "a"})
	want, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)

	got, err := TopLevel(filepath.Join(dir, "sub", "deep"))
	require.NoError(t, err)
	assert.Equal(t, want, got, "symlink-resolved repository root")
}

func TestTopLevel_NotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	_, err := TopLevel(t.TempDir())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotARepo))
}
