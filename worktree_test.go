package main

import (
	"os"
	"path/filepath"
	"testing"
)

// repoAt builds a main worktree: a directory with a real .git directory.
func repoAt(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// worktreeAt builds a linked worktree of repo, the way git records one: a
// .git file in the tree pointing at a record under the repo, and a gitdir
// file in that record pointing back.
func worktreeAt(t *testing.T, repo, id, tree string) string {
	t.Helper()
	record := filepath.Join(repo, ".git", "worktrees", id)
	if err := os.MkdirAll(record, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(tree, ".git"), "gitdir: "+record+"\n")
	write(t, filepath.Join(record, "gitdir"), filepath.Join(tree, ".git")+"\n")
	return tree
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMainWorktreeFromTheRepositoryItself(t *testing.T) {
	root := t.TempDir()
	repo := repoAt(t, filepath.Join(root, "org", "thing"))
	if got := mainWorktree(repo); got != repo {
		t.Errorf("got %q, want %q", got, repo)
	}
}

func TestMainWorktreeFromASubdirectory(t *testing.T) {
	root := t.TempDir()
	repo := repoAt(t, filepath.Join(root, "org", "thing"))
	deep := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := mainWorktree(deep); got != repo {
		t.Errorf("got %q, want %q", got, repo)
	}
}

// The case the whole design turns on: a worktree that lives nowhere near its
// repository still resolves to it.
func TestMainWorktreeFromAWorktreeOutsideTheRepository(t *testing.T) {
	root := t.TempDir()
	repo := repoAt(t, filepath.Join(root, "org", "thing"))
	tree := worktreeAt(t, repo, "thing-fix-1", filepath.Join(root, "elsewhere", "squad", "thing-fix-1"))
	if got := mainWorktree(tree); got != repo {
		t.Errorf("got %q, want %q", got, repo)
	}
}

func TestMainWorktreeOutsideAnyRepository(t *testing.T) {
	if got := mainWorktree(t.TempDir()); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := mainWorktree(""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestWorktreesOfSkipsAPrunableRecord(t *testing.T) {
	root := t.TempDir()
	repo := repoAt(t, filepath.Join(root, "org", "thing"))
	worktreeAt(t, repo, "thing-live", filepath.Join(root, "wt", "thing-live"))

	// A record git has not pruned, whose tree is gone.
	gone := filepath.Join(repo, ".git", "worktrees", "thing-gone")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(gone, "gitdir"), filepath.Join(root, "wt", "thing-gone", ".git")+"\n")

	got := worktreesOf(repo)
	if len(got) != 1 {
		t.Fatalf("got %d worktrees, want 1: %+v", len(got), got)
	}
	if got[0].Label != "live" {
		t.Errorf("label %q, want live", got[0].Label)
	}
}

func TestWorktreesOfOnARepositoryWithNone(t *testing.T) {
	repo := repoAt(t, filepath.Join(t.TempDir(), "org", "thing"))
	if got := worktreesOf(repo); len(got) != 0 {
		t.Errorf("got %d, want 0", len(got))
	}
}

func TestLabelDropsTheRepositoryPrefix(t *testing.T) {
	for _, tc := range []struct{ tree, repo, want string }{
		{"/a/b/up2-monitoring-rebase-517", "up2-monitoring", "rebase-517"},
		{"/a/b/slopguard-heldout", "slopguard", "heldout"},
		{"/a/b/uptime-com_18d58f55", "up2-monitoring", "uptime-com_18d58f55"},
		{"/a/b/thing", "thing", "thing"}, // exact match keeps its name
	} {
		if got := label(tc.tree, tc.repo); got != tc.want {
			t.Errorf("label(%q, %q) = %q, want %q", tc.tree, tc.repo, got, tc.want)
		}
	}
}

func TestForgeReposTakesTwoLevelsOnly(t *testing.T) {
	root := t.TempDir()
	repoAt(t, filepath.Join(root, "org", "one"))
	repoAt(t, filepath.Join(root, "org", "two"))
	repoAt(t, filepath.Join(root, "other", "three"))
	// Three levels down: not a checkout to open.
	repoAt(t, filepath.Join(root, "org", "one", "nested"))
	// A bare directory with no .git.
	if err := os.MkdirAll(filepath.Join(root, "org", "notarepo"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := forgeRepos(root)
	if len(got) != 3 {
		t.Fatalf("got %d repos, want 3: %v", len(got), got)
	}
	want := []string{
		filepath.Join(root, "org", "one"),
		filepath.Join(root, "org", "two"),
		filepath.Join(root, "other", "three"),
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("repo %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRepoLabelQualifiesWithTheOrg(t *testing.T) {
	root := "/Users/m/Forge"
	if got := repoLabel("/Users/m/Forge/uptime-com/api", root); got != "uptime-com/api" {
		t.Errorf("got %q", got)
	}
	if got := repoLabel("/somewhere/else", root); got != "/somewhere/else" {
		t.Errorf("got %q", got)
	}
}
