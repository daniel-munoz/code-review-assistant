package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// TopLevel returns the absolute, symlink-resolved root of the work tree
// containing dir. It returns an error wrapping ErrNotARepo when dir is not
// inside a git work tree (or git is unavailable).
func TopLevel(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%w: %s is not inside a work tree", ErrNotARepo, dir)
	}
	top := strings.TrimSpace(out)
	if resolved, err := filepath.EvalSymlinks(top); err == nil {
		top = resolved
	}
	return filepath.FromSlash(top), nil
}
