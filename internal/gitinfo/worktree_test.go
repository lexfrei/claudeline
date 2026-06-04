package gitinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkedWorktreeNameEmptyCwd(t *testing.T) {
	t.Parallel()

	if got := LinkedWorktreeName(""); got != "" {
		t.Errorf("expected empty worktree name for empty cwd, got %q", got)
	}
}

func TestLinkedWorktreeNameNotAGitRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if got := LinkedWorktreeName(dir); got != "" {
		t.Errorf("expected empty worktree name outside a repo, got %q", got)
	}
}

func TestLinkedWorktreeNameMainClone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")

	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Main clone: .git is a directory, so there is no linked worktree name.
	if got := LinkedWorktreeName(dir); got != "" {
		t.Errorf("expected empty worktree name for main clone, got %q", got)
	}
}

func TestLinkedWorktreeNameAbsoluteGitdir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mainGit := filepath.Join(root, "main", ".git")
	worktreeGitdir := filepath.Join(mainGit, "worktrees", "side")
	worktreeDir := filepath.Join(root, "side")

	if err := os.MkdirAll(worktreeGitdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+worktreeGitdir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := LinkedWorktreeName(worktreeDir); got != "side" {
		t.Errorf("expected side, got %q", got)
	}
}

func TestLinkedWorktreeNameRelativeGitdir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mainGit := filepath.Join(root, "main", ".git")
	worktreeGitdir := filepath.Join(mainGit, "worktrees", "rel-side")
	worktreeDir := filepath.Join(root, "rel-side")

	if err := os.MkdirAll(worktreeGitdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	relGitdir, err := filepath.Rel(worktreeDir, worktreeGitdir)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+relGitdir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := LinkedWorktreeName(worktreeDir); got != "rel-side" {
		t.Errorf("expected rel-side, got %q", got)
	}
}

func TestLinkedWorktreeNameUsesCwdNotGitdirBasename(t *testing.T) {
	t.Parallel()

	// Two worktrees can share an on-disk directory basename ("side") while git
	// disambiguates their metadata dirs (worktrees/side, worktrees/side1). The
	// displayed name must follow the on-disk directory the user sees, not the
	// disambiguated metadata basename.
	root := t.TempDir()
	worktreeGitdir := filepath.Join(root, "main", ".git", "worktrees", "side1")
	worktreeDir := filepath.Join(root, "other", "side")

	if err := os.MkdirAll(worktreeGitdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+worktreeGitdir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := LinkedWorktreeName(worktreeDir); got != "side" {
		t.Errorf("expected side (on-disk dir name), got %q", got)
	}
}

func TestLinkedWorktreeNameMalformedGitFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("not a gitdir pointer\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := LinkedWorktreeName(dir); got != "" {
		t.Errorf("expected empty worktree name for malformed .git file, got %q", got)
	}
}

func TestLinkedWorktreeNameGitdirNotUnderWorktrees(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// A .git file pointing somewhere that is not a ".../worktrees/<name>"
	// layout must not be mistaken for a linked worktree.
	otherGitdir := filepath.Join(root, "somewhere", "custom")
	worktreeDir := filepath.Join(root, "wt")

	if err := os.MkdirAll(otherGitdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+otherGitdir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := LinkedWorktreeName(worktreeDir); got != "" {
		t.Errorf("expected empty worktree name when gitdir is not under worktrees/, got %q", got)
	}
}
