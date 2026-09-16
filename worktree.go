package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Worktree is a linked worktree of some repository.
type Worktree struct {
	Label string
	Dir   string
}

// mainWorktree returns the main worktree of the repository containing dir,
// or "" when dir is in none.
//
// A linked worktree's `.git` is a file naming a gitdir under the main
// worktree's own `.git/worktrees/`, which is what makes the main worktree
// reachable by reading rather than by running git.
func mainWorktree(dir string) string {
	if dir == "" {
		return ""
	}
	current, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		marker := filepath.Join(current, ".git")
		info, err := os.Lstat(marker)
		switch {
		case err != nil:
			// keep walking up
		case info.IsDir():
			return current
		default:
			gitdir := readGitdir(marker)
			if gitdir == "" {
				return current
			}
			if repo, ok := repoOfGitdir(gitdir); ok {
				return repo
			}
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// repoOfGitdir maps `<main>/.git/worktrees/<id>` back to `<main>`.
func repoOfGitdir(gitdir string) (string, bool) {
	parent := filepath.Dir(gitdir)      // <main>/.git/worktrees
	grandparent := filepath.Dir(parent) // <main>/.git
	if filepath.Base(parent) != "worktrees" || filepath.Base(grandparent) != ".git" {
		return "", false
	}
	return filepath.Dir(grandparent), true
}

// readGitdir returns the path a `.git` file points at, or "".
func readGitdir(marker string) string {
	data, err := os.ReadFile(marker)
	if err != nil {
		return ""
	}
	key, value, ok := strings.Cut(strings.TrimSpace(string(data)), ":")
	if !ok || strings.TrimSpace(key) != "gitdir" {
		return ""
	}
	return strings.TrimSpace(value)
}

// worktreesOf returns every linked worktree of repo.
//
// Read from `.git/worktrees/<id>/gitdir`, which git keeps pointing at the
// worktree's own `.git`, so worktrees living outside the repository are found
// the same way as the ones beside it. A record whose directory is gone is
// skipped: git leaves those until someone prunes them.
func worktreesOf(repo string) []Worktree {
	records := filepath.Join(repo, ".git", "worktrees")
	entries, err := os.ReadDir(records)
	if err != nil {
		return nil
	}
	name := filepath.Base(repo)
	var found []Worktree
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(records, entry.Name(), "gitdir"))
		if err != nil {
			continue
		}
		target := strings.TrimSpace(string(data))
		if target == "" {
			continue
		}
		tree := target
		if filepath.Base(tree) == ".git" {
			tree = filepath.Dir(tree)
		}
		if info, err := os.Stat(tree); err != nil || !info.IsDir() {
			continue
		}
		found = append(found, Worktree{Label: label(tree, name), Dir: tree})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Label < found[j].Label })
	return found
}

// label names a worktree by its directory, less a repository prefix.
//
// `up2-monitoring/.claude/worktrees/up2-monitoring-rebase-517` is named
// `rebase-517`: repeating the repository in the suffix says nothing.
func label(tree, repoName string) string {
	base := filepath.Base(tree)
	prefix := repoName + "-"
	if strings.HasPrefix(base, prefix) && len(base) > len(prefix) {
		return strings.TrimPrefix(base, prefix)
	}
	return base
}

// forgeRepos returns every main worktree two levels below root, as `org/repo`
// paths.
//
// Two levels only: a repository nested inside another is that repository's
// business, and the `.git` directories below it are worktree plumbing rather
// than checkouts to open.
func forgeRepos(root string) []string {
	orgs, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var found []string
	for _, org := range orgs {
		if !org.IsDir() || strings.HasPrefix(org.Name(), ".") {
			continue
		}
		repos, err := os.ReadDir(filepath.Join(root, org.Name()))
		if err != nil {
			continue
		}
		for _, repo := range repos {
			if !repo.IsDir() {
				continue
			}
			dir := filepath.Join(root, org.Name(), repo.Name())
			if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
				found = append(found, dir)
			}
		}
	}
	sort.Strings(found)
	return found
}

// repoLabel names a repository as a reader sees it: `org/repo` below root,
// else a path with the home directory shortened.
//
// A bare directory name collides when the same repository name appears below
// two orgs, which is what the org qualifier is for.
func repoLabel(repo, root string) string {
	if rel, err := filepath.Rel(root, repo); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return shorten(repo)
}

// shorten replaces the home directory with `~`.
func shorten(path string) string {
	if path == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || !strings.HasPrefix(path, home) {
		return path
	}
	return "~" + strings.TrimPrefix(path, home)
}
