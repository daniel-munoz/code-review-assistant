package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractInfo_GitRepository(t *testing.T) {
	// Create a temporary Git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run(), "failed to init git repo")

	// Configure git user (required for commits)
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create a file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Extract Git info
	info := ExtractInfo(tmpDir)

	// Verify commit hash exists and is valid (40 char hex string)
	assert.NotEmpty(t, info.Commit, "commit hash should not be empty")
	assert.Len(t, info.Commit, 40, "commit hash should be 40 characters")
	assert.Regexp(t, "^[0-9a-f]{40}$", info.Commit, "commit hash should be hexadecimal")

	// Verify branch name (should be master or main depending on git version)
	assert.NotEmpty(t, info.Branch, "branch name should not be empty")
	assert.Contains(t, []string{"master", "main"}, info.Branch, "branch should be master or main")
}

func TestExtractInfo_NonGitDirectory(t *testing.T) {
	// Create a temporary non-Git directory
	tmpDir := t.TempDir()

	// Extract Git info (should return empty values, not error)
	info := ExtractInfo(tmpDir)

	// Verify both fields are empty
	assert.Empty(t, info.Commit, "commit should be empty for non-git directory")
	assert.Empty(t, info.Branch, "branch should be empty for non-git directory")
}

func TestExtractInfo_NonExistentDirectory(t *testing.T) {
	// Try to extract info from a directory that doesn't exist
	nonExistent := "/tmp/this-directory-does-not-exist-12345"

	// Should return empty values, not crash
	info := ExtractInfo(nonExistent)

	assert.Empty(t, info.Commit, "commit should be empty for non-existent directory")
	assert.Empty(t, info.Branch, "branch should be empty for non-existent directory")
}

func TestExtractInfo_WithBranch(t *testing.T) {
	// Create a temporary Git repository with a custom branch
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create and checkout a new branch
	cmd = exec.Command("git", "checkout", "-b", "feature-branch")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Extract Git info
	info := ExtractInfo(tmpDir)

	// Verify branch name is the feature branch
	assert.Equal(t, "feature-branch", info.Branch, "should detect custom branch")
	assert.NotEmpty(t, info.Commit, "should have commit hash")
}

func TestGetGitCommit(t *testing.T) {
	// Create a temporary Git repository
	tmpDir := t.TempDir()

	// Initialize and create a commit
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Test commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Test getGitCommit
	commit := getGitCommit(tmpDir)

	assert.NotEmpty(t, commit)
	assert.Len(t, commit, 40)
	assert.Regexp(t, "^[0-9a-f]{40}$", commit)
}

func TestGetGitCommit_NoGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Test getGitCommit on non-git directory
	commit := getGitCommit(tmpDir)

	assert.Empty(t, commit, "should return empty string for non-git directory")
}

func TestGetGitBranch(t *testing.T) {
	// Create a temporary Git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Test commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Test getGitBranch
	branch := getGitBranch(tmpDir)

	assert.NotEmpty(t, branch)
	assert.Contains(t, []string{"master", "main"}, branch)
}

func TestGetGitBranch_NoGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Test getGitBranch on non-git directory
	branch := getGitBranch(tmpDir)

	assert.Empty(t, branch, "should return empty string for non-git directory")
}

func TestExtractInfo_NoWhitespace(t *testing.T) {
	// Create a temporary Git repository
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	info := ExtractInfo(tmpDir)

	// Verify no leading/trailing whitespace
	assert.Equal(t, strings.TrimSpace(info.Commit), info.Commit)
	assert.Equal(t, strings.TrimSpace(info.Branch), info.Branch)

	// Verify no newlines
	assert.NotContains(t, info.Commit, "\n")
	assert.NotContains(t, info.Branch, "\n")
}
