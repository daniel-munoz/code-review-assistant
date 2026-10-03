package reporter

import (
	"fmt"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
)

// formatScope renders a one-line description of a diff-based run's scope,
// e.g. "--branch=main (merge-base a1b2c3d) · 4 files analyzed".
// It returns "" for full-project runs.
func formatScope(s *analyzer.Scope) string {
	if s == nil {
		return ""
	}
	line := "--" + s.Mode
	if s.Ref != "" {
		line += "=" + s.Ref
	}
	if s.Base != "" {
		base := s.Base
		if len(base) > 7 {
			base = base[:7]
		}
		if s.Mode == "branch" {
			line += fmt.Sprintf(" (merge-base %s)", base)
		} else {
			line += fmt.Sprintf(" (%s)", base)
		}
	}
	switch n := len(s.Files); n {
	case 0:
		return line + " · no matching files changed"
	case 1:
		return line + " · 1 file analyzed"
	default:
		return line + fmt.Sprintf(" · %d files analyzed", n)
	}
}
