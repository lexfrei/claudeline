package gitinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentBranchEmptyCwd(t *testing.T) {
	t.Parallel()

	if got := CurrentBranch(""); got != "" {
		t.Errorf("expected empty branch for empty cwd, got %q", got)
	}
}

func TestCurrentBranchNotAGitRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if got := CurrentBranch(dir); got != "" {
		t.Errorf("expected empty branch outside a repo, got %q", got)
	}
}

func TestCurrentBranchRegularRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")

	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feat/foo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CurrentBranch(dir); got != "feat/foo" {
		t.Errorf("expected feat/foo, got %q", got)
	}
}

func TestCurrentBranchDetachedHead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")

	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Detached HEAD points at a commit SHA, not a ref.
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("3a7c2f1e0d8b9f4a5c6e7d8b9f4a5c6e7d8b9f4a\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CurrentBranch(dir); got != "" {
		t.Errorf("expected empty branch on detached HEAD, got %q", got)
	}
}

func TestCurrentBranchLinkedWorktreeAbsoluteGitdir(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(worktreeGitdir, "HEAD"), []byte("ref: refs/heads/feat/side\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+worktreeGitdir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CurrentBranch(worktreeDir); got != "feat/side" {
		t.Errorf("expected feat/side, got %q", got)
	}
}

func TestCurrentBranchLinkedWorktreeRelativeGitdir(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(worktreeGitdir, "HEAD"), []byte("ref: refs/heads/feat/rel\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	relGitdir, err := filepath.Rel(worktreeDir, worktreeGitdir)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+relGitdir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CurrentBranch(worktreeDir); got != "feat/rel" {
		t.Errorf("expected feat/rel, got %q", got)
	}
}

func TestCurrentBranchMalformedGitFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("not a gitdir pointer\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CurrentBranch(dir); got != "" {
		t.Errorf("expected empty branch for malformed .git file, got %q", got)
	}
}
