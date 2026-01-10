package reporter

import (
	"os"
	"path/filepath"
)

// CreateOutputFile creates an output file, including any necessary parent directories.
// If the file already exists, it will be truncated.
func CreateOutputFile(path string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Create or truncate file
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}
