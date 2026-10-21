package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_RoundTripsPreset(t *testing.T) {
	backends := map[string]func(t *testing.T) Storage{
		"file": func(t *testing.T) Storage {
			s, err := NewFileStorage(t.TempDir())
			require.NoError(t, err)
			return s
		},
		"sqlite": func(t *testing.T) Storage {
			s, err := NewSQLiteStorage(filepath.Join(t.TempDir(), "history.db"))
			require.NoError(t, err)
			return s
		},
	}

	for name, open := range backends {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			for _, preset := range []string{"strict", ""} {
				s := open(t)
				report := createTestReport("/test/project", time.Now())
				report.Result.Preset = preset
				require.NoError(t, s.Save(ctx, report))

				latest, err := s.GetLatest(ctx, "/test/project")
				require.NoError(t, err)
				require.NotNil(t, latest)
				assert.Equal(t, preset, latest.Result.Preset)
				require.NoError(t, s.Close())
			}
		})
	}
}
