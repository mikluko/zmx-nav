package main

import (
	"strings"
	"testing"
)

// dirs returns the middle field of each switcher line, which is empty for a
// session already running and set for one that has to be created.
func dirs(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		parts := strings.SplitN(l, "\t", 3)
		if len(parts) > 1 {
			out[i] = parts[1]
		}
	}
	return out
}

func TestWidenKeepsTheNameAndLeavesTheDirectoryEmpty(t *testing.T) {
	root, found := scene(t)
	lines := widen(renderPick(found, modeFlat, root))

	for i, want := range names(renderPick(found, modeFlat, root)) {
		if got := names(lines)[i]; got != want {
			t.Errorf("line %d names %q, want %q", i, got, want)
		}
	}
	for i, dir := range dirs(lines) {
		if dir != "" {
			t.Errorf("line %d carries directory %q; a running session is attached where it runs", i, dir)
		}
	}
	for _, l := range lines {
		if n := strings.Count(l, "\t"); n != 2 {
			t.Errorf("line %q has %d tabs, want 2", l, n)
		}
	}
}

// Every grouping renders the same three fields, or fzf's --with-nth shows the
// wrong column for one of them.
func TestSwitchLinesAgreeOnTheirShape(t *testing.T) {
	root, found := scene(t)
	for _, mode := range switchModes {
		lines, shown := switchLines(found, mode, root)
		if len(lines) == 0 {
			t.Fatalf("%s rendered nothing", mode)
		}
		if shown != mode {
			t.Errorf("%s rendered as %q", mode, shown)
		}
		for _, l := range lines {
			if n := strings.Count(l, "\t"); n != 2 {
				t.Errorf("%s line %q has %d tabs, want 2", mode, l, n)
			}
		}
	}
}

// A pane whose picker is empty has nowhere to go, so a grouping with no
// sessions in it gives way to the repositories.
func TestSwitchLinesFallBackToRepositoriesWithNoSessions(t *testing.T) {
	root, _ := scene(t)
	lines, shown := switchLines(nil, modeRepo, root)

	if shown != modeNew {
		t.Errorf("with no sessions the picker showed %q, want %q", shown, modeNew)
	}
	if len(lines) == 0 {
		t.Fatal("fell back to a grouping that is also empty")
	}
	for i, dir := range dirs(lines) {
		if dir == "" {
			t.Errorf("target %d carries no directory; it has to be created somewhere", i)
		}
	}
}

func TestSwitchModesExtendPickModes(t *testing.T) {
	for i, m := range modes {
		if switchModes[i] != m {
			t.Errorf("switch grouping %d is %q, want %q", i, switchModes[i], m)
		}
	}
	if last := switchModes[len(switchModes)-1]; last != modeNew {
		t.Errorf("switch ends on %q, want %q", last, modeNew)
	}
	if validMode(modeNew, modes) {
		t.Error("pick offers the repository grouping, which it cannot render")
	}
}

func TestWithoutDropsOnlyTheNamedVariable(t *testing.T) {
	env := []string{"PATH=/bin", "ZMX_SESSION=supa-1", "ZMX_DIR=/tmp/zmx", "ZMX_SESSION_PREFIX=p"}
	got := without(env, "ZMX_SESSION")

	if len(got) != 3 {
		t.Fatalf("kept %d of %d entries: %q", len(got), len(env), got)
	}
	for _, kv := range got {
		if kv == "ZMX_SESSION=supa-1" {
			t.Error("ZMX_SESSION survived; the child would redirect its host session")
		}
	}
	if got[2] != "ZMX_SESSION_PREFIX=p" {
		t.Errorf("a variable sharing the prefix was dropped: %q", got)
	}
}
