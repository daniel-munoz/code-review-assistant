package parser

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sortedPaths(ms []*FileMetrics) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.FilePath)
	}
	sort.Strings(out)
	return out
}

func TestParseFiles_MatchesParseDirectory(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample")
	files := []string{filepath.Join(root, "util.go"), filepath.Join(root, "complex.go")}

	for _, workers := range []int{1, 4} {
		p := NewParser(workers)
		got, errs := ParseFiles(p, files, workers, status.NewSilentReporter())
		require.Empty(t, errs)

		all, _ := p.ParseDirectory(root, nil, []string{".go"}, status.NewSilentReporter())
		want := map[string]*FileMetrics{}
		for _, m := range all {
			want[filepath.Clean(m.FilePath)] = m
		}

		assert.Equal(t, []string{filepath.Join(root, "complex.go"), filepath.Join(root, "util.go")}, sortedPaths(got), "workers=%d", workers)
		for _, m := range got {
			w := want[filepath.Clean(m.FilePath)]
			require.NotNil(t, w)
			assert.Equal(t, w.TotalLines, m.TotalLines)
			assert.Equal(t, len(w.Functions), len(m.Functions))
		}
	}
}

func TestParseFiles_ErrorDoesNotStopOthers(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample")
	files := []string{filepath.Join(root, "util.go"), filepath.Join(root, "missing.go")}

	got, errs := ParseFiles(NewParser(2), files, 2, status.NewSilentReporter())

	assert.Len(t, got, 1)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "error parsing")
	assert.Contains(t, errs[0].Error(), "missing.go")
}

func TestParseFiles_Empty(t *testing.T) {
	got, errs := ParseFiles(NewParser(0), nil, 0, status.NewSilentReporter())
	assert.Empty(t, got)
	assert.Empty(t, errs)
}
