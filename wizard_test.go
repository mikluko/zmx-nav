package main

import (
	"strings"
	"testing"
)

// The layout follows the shape of the space: beside the list where the
// terminal is wide, below it where it is not, and nowhere at all where what is
// left of the list would be unreadable.
func TestPreviewLayoutFollowsTheShapeOfTheTerminal(t *testing.T) {
	for _, c := range []struct {
		cols, rows int
		want       string
	}{
		{200, 50, "right"},
		{120, 40, "right"},
		{119, 40, "up"},
		{80, 24, "up"},
		{79, 40, ""},
		{200, 14, ""},
	} {
		got := previewLayout(c.cols, c.rows)
		switch {
		case c.want == "" && got != "":
			t.Errorf("%dx%d previews as %q, want none", c.cols, c.rows, got)
		case c.want != "" && !strings.HasPrefix(got, c.want):
			t.Errorf("%dx%d previews as %q, want %s", c.cols, c.rows, got, c.want)
		}
	}
}

// fzf carries one alternative of its own, which is what answers a terminal
// resized while the picker is up.
func TestSideBySidePreviewCarriesANarrowAlternative(t *testing.T) {
	got := previewLayout(200, 50)
	if !strings.Contains(got, "<100(up") {
		t.Errorf("wide layout %q has no narrow alternative", got)
	}
}

// Every menu item either opens a menu below it or ends in a step, and the
// wizard walks exactly two levels before it does anything.
func TestEveryMenuItemLeadsSomewhere(t *testing.T) {
	reached := map[string]bool{}
	for _, branch := range tree {
		if len(branch.kids) == 0 {
			t.Errorf("%q is a branch with nothing under it", branch.name)
		}
		for _, kid := range branch.kids {
			if len(kid.kids) > 0 {
				t.Errorf("%s/%s goes a level deeper than the wizard walks", branch.name, kid.name)
			}
			path := branch.name + "/" + kid.name
			if steps[path] == nil {
				t.Errorf("%s has no step", path)
			}
			reached[path] = true
		}
	}
	for path := range steps {
		if !reached[path] {
			t.Errorf("%s is a step no menu reaches", path)
		}
	}
}

func TestMenuLinesHideTheNameInFieldOne(t *testing.T) {
	lines := menuLines(tree)
	if len(lines) != len(tree) {
		t.Fatalf("rendered %d lines for %d items", len(lines), len(tree))
	}
	for i, line := range lines {
		name, display, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("line %q carries no display column", line)
		}
		if name != tree[i].name {
			t.Errorf("line %d names %q, want %q", i, name, tree[i].name)
		}
		if !strings.Contains(display, tree[i].hint) {
			t.Errorf("line %d shows %q, which drops the hint", i, display)
		}
	}
}
