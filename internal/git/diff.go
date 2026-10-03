package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Mode selects which change set diff-based analysis looks at.
type Mode string

const (
	// ModeStaged compares the index with HEAD (pre-commit).
	ModeStaged Mode = "staged"
	// ModeSince compares a ref with the working tree.
	ModeSince Mode = "since"
	// ModeBranch compares merge-base(ref, HEAD) with the working tree (PR / CI).
	ModeBranch Mode = "branch"
)

// Spec describes the requested change set.
type Spec struct {
	Mode Mode
	Ref  string // Unused for ModeStaged
}

// Flag returns the CLI flag that selects this mode, for error messages.
func (s Spec) Flag() string { return "--" + string(s.Mode) }

// ChangeSet is the result of ChangedFiles.
type ChangeSet struct {
	// Files changed (added, copied, modified, renamed), relative to the
	// directory passed to ChangedFiles, sorted. Deleted files are omitted.
	Files []string
	// Base is the resolved commit the working tree is compared against
	// (ModeSince: the ref; ModeBranch: the merge-base; ModeStaged: empty).
	Base string
	// PartiallyStaged lists staged files that also have unstaged changes
	// (ModeStaged only).
	PartiallyStaged []string
}

// Sentinel errors for diff-mode failures; match with errors.Is.
var (
	ErrGitNotFound = errors.New("diff mode requires git, which was not found in PATH")
	ErrNotARepo    = errors.New("not a git repository")
	ErrUnknownRef  = errors.New("unknown ref")
	ErrNoMergeBase = errors.New("no merge-base")
)

// ChangedFiles lists the files changed according to spec, relative to dir.
//
// All git commands run inside dir with --relative, so when dir is a
// subdirectory of the repository, only changes under it are returned.
// Untracked files never appear (git diff does not list them).
func ChangedFiles(dir string, spec Spec) (*ChangeSet, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrGitNotFound
	}
	if out, err := runGit(dir, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		return nil, fmt.Errorf("%w: %s requires a git repository; %s is not inside a work tree", ErrNotARepo, spec.Flag(), dir)
	}

	switch spec.Mode {
	case ModeStaged:
		staged, err := diffNames(dir, "--cached")
		if err != nil {
			return nil, err
		}
		unstaged, err := diffNames(dir)
		if err != nil {
			return nil, err
		}
		return &ChangeSet{Files: staged, PartiallyStaged: intersect(staged, unstaged)}, nil

	case ModeSince:
		base, err := resolveCommit(dir, spec)
		if err != nil {
			return nil, err
		}
		files, err := diffNames(dir, base)
		if err != nil {
			return nil, err
		}
		return &ChangeSet{Files: files, Base: base}, nil

	case ModeBranch:
		ref, err := resolveCommit(dir, spec)
		if err != nil {
			return nil, err
		}
		out, err := runGit(dir, "merge-base", ref, "HEAD")
		if err != nil {
			return nil, fmt.Errorf("%w: no common ancestor between HEAD and %s; in CI, fetch full history (e.g. actions/checkout with fetch-depth: 0)", ErrNoMergeBase, spec.Ref)
		}
		base := strings.TrimSpace(out)
		files, err := diffNames(dir, base)
		if err != nil {
			return nil, err
		}
		return &ChangeSet{Files: files, Base: base}, nil

	default:
		return nil, fmt.Errorf("unknown diff mode %q", spec.Mode)
	}
}

// resolveCommit resolves spec.Ref to a full commit SHA.
func resolveCommit(dir string, spec Spec) (string, error) {
	out, err := runGit(dir, "rev-parse", "--verify", "--quiet", spec.Ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("%w %q for %s", ErrUnknownRef, spec.Ref, spec.Flag())
	}
	return strings.TrimSpace(out), nil
}

// diffNames runs `git diff --name-only` with the standard diff-mode flags plus
// args, and returns the NUL-separated paths in OS form, sorted.
func diffNames(dir string, args ...string) ([]string, error) {
	full := append([]string{"diff", "--name-only", "--relative", "-z", "--diff-filter=ACMR", "--no-color", "--no-ext-diff"}, args...)
	full = append(full, "--")
	out, err := runGit(dir, full...)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			files = append(files, filepath.FromSlash(p))
		}
	}
	sort.Strings(files)
	return files, nil
}

// intersect returns the elements of a that are also in b (a's order).
func intersect(a, b []string) []string {
	in := make(map[string]bool, len(b))
	for _, x := range b {
		in[x] = true
	}
	var out []string
	for _, x := range a {
		if in[x] {
			out = append(out, x)
		}
	}
	return out
}

// gitEnv returns env for git subprocesses. Git runs hooks with GIT_DIR set
// (in a linked worktree: <repo>/.git/worktrees/<name>) and GIT_WORK_TREE
// unset; git then treats the command's directory as the work tree root, so
// --relative paths from a subdirectory target come back wrong. When GIT_DIR
// is set without GIT_WORK_TREE, drop it (and GIT_PREFIX) so git discovers
// the repository from dir. GIT_INDEX_FILE is kept, so `git commit -a` hooks
// still see their temporary index. An explicit GIT_DIR + GIT_WORK_TREE pair
// is kept as is.
func gitEnv(env []string) []string {
	hasDir, hasWorkTree := false, false
	for _, kv := range env {
		hasDir = hasDir || strings.HasPrefix(kv, "GIT_DIR=")
		hasWorkTree = hasWorkTree || strings.HasPrefix(kv, "GIT_WORK_TREE=")
	}
	if !hasDir || hasWorkTree {
		return env
	}
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_DIR=") || strings.HasPrefix(kv, "GIT_PREFIX=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// runGit runs git with args in dir and returns stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(os.Environ())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}
