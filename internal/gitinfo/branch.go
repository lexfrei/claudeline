// Package gitinfo reads minimal git metadata from the working tree without
// shelling out — only the files needed to identify the current branch.
package gitinfo

import (
	"os"
	"path/filepath"
	"strings"
)

const headRefPrefix = "ref: refs/heads/"

// CurrentBranch returns the branch name at HEAD for the given working directory.
// Returns an empty string when:
//
//   - cwd is empty or not inside a git repository
//   - HEAD is detached (no symbolic ref)
//   - any required file is unreadable
//
// Supports both regular repos (.git is a directory) and linked worktrees
// (.git is a file pointing to gitdir).
func CurrentBranch(cwd string) string {
	if cwd == "" {
		return ""
	}

	headPath, ok := resolveHeadPath(cwd)
	if !ok {
		return ""
	}

	content, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, headRefPrefix) {
		return ""
	}

	return strings.TrimPrefix(line, headRefPrefix)
}

func resolveHeadPath(cwd string) (string, bool) {
	gitDir, _, ok := resolveGitDir(cwd)
	if !ok {
		return "", false
	}

	return filepath.Join(gitDir, "HEAD"), true
}

// resolveGitDir returns the git directory that backs cwd and whether cwd is a
// linked worktree. A regular repo has a .git directory and reports linked=false;
// a linked worktree has a .git *file* containing "gitdir: <path>" and reports
// linked=true. ok is false when cwd is not inside a git repository or the
// pointer file is unreadable or malformed.
func resolveGitDir(cwd string) (gitDir string, linked, ok bool) {
	gitPath := filepath.Join(cwd, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		return "", false, false
	}

	if info.IsDir() {
		return gitPath, false, true
	}

	// Linked worktree: .git is a file containing "gitdir: <path>".
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", false, false
	}

	line := strings.TrimSpace(string(content))

	const gitdirPrefix = "gitdir: "
	if !strings.HasPrefix(line, gitdirPrefix) {
		return "", false, false
	}

	gitDir = strings.TrimPrefix(line, gitdirPrefix)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(cwd, gitDir)
	}

	return gitDir, true, true
}
