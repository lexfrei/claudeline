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
	gitPath := filepath.Join(cwd, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		return "", false
	}

	if info.IsDir() {
		return filepath.Join(gitPath, "HEAD"), true
	}

	// Linked worktree: .git is a file containing "gitdir: <path>".
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", false
	}

	line := strings.TrimSpace(string(content))

	const gitdirPrefix = "gitdir: "
	if !strings.HasPrefix(line, gitdirPrefix) {
		return "", false
	}

	gitdir := strings.TrimPrefix(line, gitdirPrefix)
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(cwd, gitdir)
	}

	return filepath.Join(gitdir, "HEAD"), true
}
