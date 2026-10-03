package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitT runs git in dir with a fixed identity, no signing and branch "main",
// independent of the developer's global git config.
func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{
		"-c", "user.name=Test", "-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
}

// newRepo creates a repo on branch main with one commit containing files.
func newRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitT(t, dir, "init", "-q")
	for rel, c := range files {
		writeFile(t, dir, rel, c)
	}
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

func paths(rel ...string) []string {
	out := make([]string, len(rel))
	for i, r := range rel {
		out[i] = filepath.FromSlash(r)
	}
	return out
}

func TestChangedFiles_Staged(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "a", "b.go": "b"})
	writeFile(t, dir, "a.go", "a2")
	writeFile(t, dir, "c.go", "c")
	gitT(t, dir, "add", "a.go", "c.go")
	writeFile(t, dir, "b.go", "b2") // unstaged only

	cs, err := ChangedFiles(dir, Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("a.go", "c.go"), cs.Files)
	assert.Empty(t, cs.PartiallyStaged)
	assert.Empty(t, cs.Base)
}

func TestChangedFiles_PartiallyStaged(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "a"})
	writeFile(t, dir, "a.go", "a2")
	gitT(t, dir, "add", "a.go")
	writeFile(t, dir, "a.go", "a3")

	cs, err := ChangedFiles(dir, Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("a.go"), cs.Files)
	assert.Equal(t, paths("a.go"), cs.PartiallyStaged)
}

func TestChangedFiles_StagedDeleteIgnoredRenameUsesNewPath(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "package a\n\nfunc A() {}\n", "b.go": "b"})
	gitT(t, dir, "rm", "-q", "b.go")
	gitT(t, dir, "mv", "a.go", "renamed.go")

	cs, err := ChangedFiles(dir, Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("renamed.go"), cs.Files)
}

func TestChangedFiles_StagedInFreshRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitT(t, dir, "init", "-q")
	writeFile(t, dir, "main.go", "package main\n")
	gitT(t, dir, "add", "main.go")

	cs, err := ChangedFiles(dir, Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("main.go"), cs.Files)
}

func TestChangedFiles_SinceIncludesUncommitted(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "a", "b.go": "b", "c.go": "c"})
	writeFile(t, dir, "a.go", "a2")
	gitT(t, dir, "commit", "-q", "-am", "change a")
	writeFile(t, dir, "b.go", "b2") // uncommitted

	cs, err := ChangedFiles(dir, Spec{Mode: ModeSince, Ref: "HEAD~1"})
	require.NoError(t, err)
	assert.Equal(t, paths("a.go", "b.go"), cs.Files)
	assert.Equal(t, gitT(t, dir, "rev-parse", "HEAD~1"), cs.Base)
}

func TestChangedFiles_BranchUsesMergeBase(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "a", "b.go": "b"})
	base := gitT(t, dir, "rev-parse", "HEAD")
	gitT(t, dir, "checkout", "-q", "-b", "feature")
	writeFile(t, dir, "b.go", "b2")
	gitT(t, dir, "commit", "-q", "-am", "feature change")
	gitT(t, dir, "checkout", "-q", "main")
	writeFile(t, dir, "a.go", "a2")
	gitT(t, dir, "commit", "-q", "-am", "main moved on")
	gitT(t, dir, "checkout", "-q", "feature")

	cs, err := ChangedFiles(dir, Spec{Mode: ModeBranch, Ref: "main"})
	require.NoError(t, err)
	assert.Equal(t, paths("b.go"), cs.Files, "changes on main after branching must not appear")
	assert.Equal(t, base, cs.Base)
}

func TestChangedFiles_SubdirectoryTarget(t *testing.T) {
	dir := newRepo(t, map[string]string{"top.go": "t", "sub/x.go": "x"})
	writeFile(t, dir, "top.go", "t2")
	writeFile(t, dir, "sub/x.go", "x2")
	gitT(t, dir, "add", "-A")

	cs, err := ChangedFiles(filepath.Join(dir, "sub"), Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("x.go"), cs.Files)
}

func TestChangedFiles_PathWithSpaces(t *testing.T) {
	dir := newRepo(t, map[string]string{"dir with space/my file.go": "a"})
	writeFile(t, dir, "dir with space/my file.go", "a2")
	gitT(t, dir, "add", "-A")

	cs, err := ChangedFiles(dir, Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("dir with space/my file.go"), cs.Files)
}

func TestChangedFiles_NotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	_, err := ChangedFiles(dir, Spec{Mode: ModeStaged})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotARepo))
	assert.Contains(t, err.Error(), "--staged requires a git repository")
}

func TestChangedFiles_UnknownRef(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "a"})
	_, err := ChangedFiles(dir, Spec{Mode: ModeSince, Ref: "nope"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnknownRef))
	assert.Contains(t, err.Error(), `unknown ref "nope" for --since`)
}

func TestChangedFiles_NoMergeBase(t *testing.T) {
	dir := newRepo(t, map[string]string{"a.go": "a"})
	gitT(t, dir, "checkout", "-q", "--orphan", "other")
	gitT(t, dir, "commit", "-q", "-m", "unrelated history")

	_, err := ChangedFiles(dir, Spec{Mode: ModeBranch, Ref: "main"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoMergeBase))
	assert.Contains(t, err.Error(), "fetch-depth: 0")
}

func TestSpecFlag(t *testing.T) {
	assert.Equal(t, "--staged", Spec{Mode: ModeStaged}.Flag())
	assert.Equal(t, "--since", Spec{Mode: ModeSince}.Flag())
	assert.Equal(t, "--branch", Spec{Mode: ModeBranch}.Flag())
}

// Git runs hooks in a linked worktree with GIT_DIR=<repo>/.git/worktrees/<name>
// and no GIT_WORK_TREE. A subdirectory target must still see its staged files.
func TestChangedFiles_HookEnvInLinkedWorktreeSubdir(t *testing.T) {
	main := newRepo(t, map[string]string{"sub/a.go": "a"})
	wt := filepath.Join(t.TempDir(), "wt")
	gitT(t, main, "worktree", "add", "-q", "-b", "feature", wt)
	writeFile(t, wt, "sub/a.go", "a2")
	gitT(t, wt, "add", "sub/a.go")
	gitDir := gitT(t, wt, "rev-parse", "--absolute-git-dir")

	t.Setenv("GIT_DIR", gitDir)
	cs, err := ChangedFiles(filepath.Join(wt, "sub"), Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("a.go"), cs.Files)
}

// `git commit -a` / `git commit <path>` run hooks against a temporary index
// named by GIT_INDEX_FILE, which may be relative to the hook's working directory.
func TestChangedFiles_RelativeIndexFileWithSubdirTarget(t *testing.T) {
	dir := newRepo(t, map[string]string{"sub/a.go": "a", "sub/b.go": "b"})
	gitT(t, dir, "read-tree", "--index-output=.git/alt-index", "HEAD")
	writeFile(t, dir, "sub/b.go", "b2")
	cmd := exec.Command("git", "add", "sub/b.go")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE=.git/alt-index")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)

	t.Chdir(dir)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(".git", "alt-index"))
	cs, err := ChangedFiles(filepath.Join(dir, "sub"), Spec{Mode: ModeStaged})
	require.NoError(t, err)
	assert.Equal(t, paths("b.go"), cs.Files)
}
