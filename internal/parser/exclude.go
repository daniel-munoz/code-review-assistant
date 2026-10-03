package parser

import (
	"path"
	"path/filepath"
	"strings"
)

// MatchPattern reports whether relPath, relative to the project root, matches
// an exclude pattern. Both are compared segment by segment:
//
//   - "**" matches zero or more whole path segments
//   - any other segment matches exactly one path segment (path.Match: *, ?, [...])
//   - a pattern without "/" matches at any depth ("generated", "*.pb.go")
//   - a pattern with "/" is anchored at the project root ("vendor/**")
//
// The whole path must be consumed, so "**/build/**" matches "build/x.js" but
// not "rebuild/x.js".
func MatchPattern(relPath, pattern string) bool {
	relPath = strings.Trim(strings.ReplaceAll(filepath.ToSlash(relPath), `\`, "/"), "/")
	pattern = strings.Trim(filepath.ToSlash(pattern), "/")
	if relPath == "" || relPath == "." || pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "/") {
		pattern = "**/" + pattern
	}
	return matchSegments(strings.Split(relPath, "/"), strings.Split(pattern, "/"))
}

// matchSegments matches path segments against pattern segments.
func matchSegments(pathSegs, patSegs []string) bool {
	for len(patSegs) > 0 {
		seg := patSegs[0]
		if seg == "**" {
			rest := patSegs[1:]
			for len(rest) > 0 && rest[0] == "**" {
				rest = rest[1:]
			}
			if len(rest) == 0 {
				return true
			}
			for i := 0; i <= len(pathSegs); i++ {
				if matchSegments(pathSegs[i:], rest) {
					return true
				}
			}
			return false
		}
		if len(pathSegs) == 0 {
			return false
		}
		if ok, err := path.Match(seg, pathSegs[0]); err != nil || !ok {
			return false
		}
		pathSegs, patSegs = pathSegs[1:], patSegs[1:]
	}
	return len(pathSegs) == 0
}

// ShouldExclude reports whether path (a file or directory under rootPath)
// matches any exclude pattern. The path and each of its ancestor directories
// are checked, so a file list is excluded exactly as a directory walk that
// prunes excluded directories would exclude it. rootPath itself is never excluded.
func ShouldExclude(path, rootPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	rel, err := filepath.Rel(rootPath, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return false
	}
	segs := strings.Split(rel, "/")
	for i := 1; i <= len(segs); i++ {
		candidate := strings.Join(segs[:i], "/")
		for _, p := range patterns {
			if MatchPattern(candidate, p) {
				return true
			}
		}
	}
	return false
}

// HasMatchingExtension reports whether path ends with one of the extensions
// (each including the leading dot, e.g. ".go").
func HasMatchingExtension(path string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
