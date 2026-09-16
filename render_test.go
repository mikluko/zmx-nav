package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// names returns the hidden first field of each rendered line, which is what
// fzf hands back and what the caller attaches to.
func names(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i], _, _ = strings.Cut(l, "\t")
	}
	return out
}

// displays returns what fzf shows and searches.
func displays(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		_, out[i], _ = strings.Cut(l, "\t")
	}
	return out
}

// scene builds a root holding one repository, one of its worktrees, and
// returns sessions sitting in those plus one outside any repository.
func scene(t *testing.T) (root string, found []Session) {
	t.Helper()
	root = t.TempDir()
	repo := repoAt(t, filepath.Join(root, "mikluko", "octant"))
	tree := worktreeAt(t, repo, "octant-918-arrival",
		filepath.Join(root, "mikluko", "octant", "tmp", "worktrees", "octant-918-arrival"))
	other := repoAt(t, filepath.Join(root, "uptime-com", "api"))
	loose := t.TempDir()

	return root, []Session{
		{Name: "mikluko.octant", Clients: "1", Dir: repo},
		{Name: "mikluko.octant@918-arrival", Clients: "0", Dir: tree},
		{Name: "uptime-com.api", Clients: "0", Dir: other},
		{Name: "scratch", Clients: "2", Dir: loose},
	}
}

func TestRenderPickRepoGroupsWorktreesUnderTheirRepository(t *testing.T) {
	root, found := scene(t)
	lines := renderPick(found, modeRepo, root)

	want := []string{
		"mikluko.octant",
		"mikluko.octant@918-arrival",
		"uptime-com.api",
		"scratch", // outside any repository, so last
	}
	got := names(lines)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}

	// The repository leads the display, and the worktree label follows it.
	if !strings.HasPrefix(displays(lines)[0], "mikluko/octant") {
		t.Errorf("repo column missing: %q", displays(lines)[0])
	}
	if !strings.Contains(displays(lines)[1], "918-arrival") {
		t.Errorf("worktree label missing: %q", displays(lines)[1])
	}
	// The main worktree is marked ".", not by repeating the repository.
	if !strings.Contains(displays(lines)[0], " .  ") {
		t.Errorf("main worktree not marked: %q", displays(lines)[0])
	}
}

func TestRenderPickDirSortsByDirectory(t *testing.T) {
	root, found := scene(t)
	got := displays(renderPick(found, modeDir, root))
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("not sorted by directory: %q before %q", got[i-1], got[i])
		}
	}
}

func TestRenderPickFlatSortsByName(t *testing.T) {
	root, found := scene(t)
	got := names(renderPick(found, modeFlat, root))
	want := []string{"mikluko.octant", "mikluko.octant@918-arrival", "scratch", "uptime-com.api"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// Every grouping must keep the name in field one, because that is what the
// preview reads as {1} and what the caller attaches to.
func TestRenderPickKeepsTheNameInFieldOne(t *testing.T) {
	root, found := scene(t)
	for _, mode := range modes {
		for _, line := range renderPick(found, mode, root) {
			name, display, ok := strings.Cut(line, "\t")
			if !ok {
				t.Fatalf("%s: line has no tab: %q", mode, line)
			}
			if strings.Contains(display, "\t") {
				t.Errorf("%s: display carries a tab, fzf would split it: %q", mode, display)
			}
			var known bool
			for _, s := range found {
				known = known || s.Name == name
			}
			if !known {
				t.Errorf("%s: field one %q names no session", mode, name)
			}
		}
	}
}

func TestRenderPickOnNoSessions(t *testing.T) {
	if got := renderPick(nil, modeRepo, t.TempDir()); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestTargetsListsRepositoriesAndTheirWorktrees(t *testing.T) {
	root := t.TempDir()
	repo := repoAt(t, filepath.Join(root, "mikluko", "octant"))
	worktreeAt(t, repo, "octant-918", filepath.Join(root, "elsewhere", "octant-918"))
	repoAt(t, filepath.Join(root, "uptime-com", "api"))

	got := targets(root)
	want := []Target{
		{Name: "mikluko.octant", Dir: repo},
		{Name: "mikluko.octant@918", Dir: filepath.Join(root, "elsewhere", "octant-918")},
		{Name: "uptime-com.api", Dir: filepath.Join(root, "uptime-com", "api")},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d targets, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("target %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// Every name targets offers must be one zmx can actually create.
func TestTargetsNamesCarryNoSeparator(t *testing.T) {
	root := t.TempDir()
	repoAt(t, filepath.Join(root, "uptime-com", "up2-monitoring"))
	for _, target := range targets(root) {
		if strings.Contains(target.Name, string(os.PathSeparator)) {
			t.Errorf("%q carries a separator, so zmx would create nothing", target.Name)
		}
	}
}

func TestRenderNewMarksRunningSessionsAndCarriesTheDirectory(t *testing.T) {
	root := t.TempDir()
	repoAt(t, filepath.Join(root, "mikluko", "octant"))
	repoAt(t, filepath.Join(root, "uptime-com", "api"))

	found := targets(root)
	lines := renderNew(found, map[string]bool{"mikluko.octant": true})
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	name, rest, _ := strings.Cut(lines[0], "\t")
	dir, display, _ := strings.Cut(rest, "\t")
	if name != "mikluko.octant" {
		t.Errorf("name %q", name)
	}
	if dir != filepath.Join(root, "mikluko", "octant") {
		t.Errorf("dir %q", dir)
	}
	if !strings.Contains(display, "running") {
		t.Errorf("display does not mark it running: %q", display)
	}
	if strings.Contains(displays(lines[1:])[0], "running") {
		t.Errorf("second target wrongly marked running")
	}
}

func TestAlignPadsToTheWidestCell(t *testing.T) {
	got := align([][]string{
		{"a", "one"},
		{"bbbb", "two"},
	})
	if len(got) != 2 {
		t.Fatalf("got %d lines", len(got))
	}
	if strings.Contains(got[0], "\t") || strings.Contains(got[1], "\t") {
		t.Errorf("tabs survived alignment: %q", got)
	}
	// The second column must start at the same offset on both lines.
	if strings.Index(got[0], "one") != strings.Index(got[1], "two") {
		t.Errorf("columns not aligned: %q / %q", got[0], got[1])
	}
}

func TestCycleModeWrapsInBothDirections(t *testing.T) {
	for _, list := range [][]string{modes, switchModes} {
		mode := list[0]
		for range list {
			mode = cycleMode(mode, true, list)
		}
		if mode != list[0] {
			t.Errorf("forward through every grouping landed on %q, want %q", mode, list[0])
		}
		for _, m := range list {
			if back := cycleMode(cycleMode(m, true, list), false, list); back != m {
				t.Errorf("%q forward then back landed on %q", m, back)
			}
		}
	}
}

// tab reads the grouping back out of the prompt, so the two must agree.
func TestPromptModeReadsBackPrompt(t *testing.T) {
	for _, list := range [][]string{modes, switchModes} {
		for _, m := range list {
			if got := promptMode(prompt(m), list); got != m {
				t.Errorf("prompt %q read back as %q", prompt(m), got)
			}
		}
		if got := promptMode("> ", list); got != list[0] {
			t.Errorf("a foreign prompt read back as %q, want %q", got, list[0])
		}
	}
}

// A grouping pick cannot render must not come back out of a prompt it reads,
// which is what would happen if the two shared one list.
func TestPromptModeKeepsTheListsApart(t *testing.T) {
	if got := promptMode(prompt(modeNew), modes); got != modes[0] {
		t.Errorf("pick read %q back as %q, want %q", prompt(modeNew), got, modes[0])
	}
}
