package git

import (
	"os/exec"
	"strings"
)

// Info contains Git repository metadata.
//
// This type holds Git information extracted from a repository at the time
// of analysis. Both fields are optional - if the directory is not a Git
// repository or Git is not available, both will be empty strings.
//
// Used to correlate analysis reports with specific code versions and
// improve comparison accuracy across runs.
type Info struct {
	// Current commit hash (full SHA)
	Commit string

	// Current branch name
	Branch string
}

// ExtractInfo attempts to extract Git metadata from a directory.
//
// This function runs git commands to get the current commit hash and branch
// name. It gracefully handles non-Git directories and missing git installations
// by returning empty strings rather than errors.
//
// The function executes:
//   - `git rev-parse HEAD` for the commit hash
//   - `git rev-parse --abbrev-ref HEAD` for the branch name
//
// Both commands are run in the specified directory. If either fails (not a git
// repo, git not installed, etc.), that field will be empty.
//
// Parameters:
//   - projectPath: The directory to extract Git info from
//
// Returns:
//   - Info struct with commit and branch (may be empty if not a Git repo)
//
// Example:
//   info := git.ExtractInfo("/path/to/project")
//   if info.Commit != "" {
//       fmt.Printf("Analyzed commit: %s on branch %s\n", info.Commit, info.Branch)
//   }
func ExtractInfo(projectPath string) Info {
	info := Info{}

	// Try to get commit hash
	if commit := getGitCommit(projectPath); commit != "" {
		info.Commit = commit
	}

	// Try to get branch name
	if branch := getGitBranch(projectPath); branch != "" {
		info.Branch = branch
	}

	return info
}

// getGitCommit extracts the current commit hash.
//
// Runs: git rev-parse HEAD
// Returns the full commit SHA, or empty string if the command fails.
func getGitCommit(projectPath string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getGitBranch extracts the current branch name.
//
// Runs: git rev-parse --abbrev-ref HEAD
// Returns the branch name, or empty string if the command fails.
//
// Note: In detached HEAD state, this returns "HEAD".
func getGitBranch(projectPath string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
