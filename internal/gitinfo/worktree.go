package gitinfo

import "path/filepath"

// worktreesDir is the directory under a repo's git dir that holds linked
// worktree metadata: "<main>/.git/worktrees/<name>".
const worktreesDir = "worktrees"

// LinkedWorktreeName returns the directory name of the linked worktree that cwd
// belongs to, or an empty string when:
//
//   - cwd is empty or not inside a git repository
//   - cwd is the main clone (.git is a directory, not a pointer file)
//   - the .git pointer does not resolve to "<main>/.git/worktrees/<name>"
//
// The name is the final path element of cwd — the worktree's on-disk root, the
// name the user actually sees. The git metadata dir basename is not used: git
// disambiguates colliding metadata dirs (worktrees/side, worktrees/side1) while
// the on-disk directories keep their original name. A non-empty result is only
// possible when cwd itself holds the .git pointer (resolveGitDir reads cwd/.git),
// so cwd is always the worktree root here.
func LinkedWorktreeName(cwd string) string {
	if cwd == "" {
		return ""
	}

	gitDir, linked, ok := resolveGitDir(cwd)
	if !ok || !linked {
		return ""
	}

	// Expect ".../worktrees/<name>"; guard against unexpected layouts so a
	// stray .git pointer is never surfaced as a worktree name.
	if filepath.Base(filepath.Dir(gitDir)) != worktreesDir {
		return ""
	}

	return filepath.Base(cwd)
}
