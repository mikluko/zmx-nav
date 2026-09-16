package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
)

// The groupings pick offers. They differ only in the leading column and the
// sort, because fzf has no unselectable line with which to draw a header row:
// putting the group in a column keeps every line selectable and keeps the
// group itself matchable.
const (
	modeFlat = "flat"
	modeDir  = "dir"
	modeRepo = "repo"
)

var modes = []string{modeFlat, modeDir, modeRepo}

func validMode(mode string, list []string) bool {
	for _, m := range list {
		if m == mode {
			return true
		}
	}
	return false
}

// prompt renders the picker's prompt for mode.
//
// fzf keeps no state a binding can read, so the prompt is where the current
// grouping lives and promptMode is how tab reads it back.
func prompt(mode string) string { return "zmx(" + mode + ")> " }

// promptMode returns the grouping a prompt names, or the first of list when
// the prompt is not one of ours.
func promptMode(s string, list []string) string {
	_, rest, ok := strings.Cut(s, "(")
	if !ok {
		return list[0]
	}
	mode, _, ok := strings.Cut(rest, ")")
	if !ok || !validMode(mode, list) {
		return list[0]
	}
	return mode
}

// cycleMode returns the grouping one step from current in list, wrapping at
// both ends.
func cycleMode(current string, forward bool, list []string) string {
	i := 0
	for j, m := range list {
		if m == current {
			i = j
			break
		}
	}
	step := 1
	if !forward {
		step = len(list) - 1
	}
	return list[(i+step)%len(list)]
}

// renderPick returns one `name\tdisplay` line per session, ordered for mode.
//
// The name leads the line so `zmx history {1}` can preview it and the choice
// can be read back; fzf is told to search and show the display only.
func renderPick(found []Session, mode, root string) []string {
	if len(found) == 0 {
		return nil
	}
	type row struct {
		name  string
		cells []string
	}
	var rows []row

	switch mode {
	case modeDir:
		sorted := append([]Session(nil), found...)
		sort.Slice(sorted, func(i, j int) bool {
			a, b := shorten(sorted[i].Dir), shorten(sorted[j].Dir)
			if a != b {
				return a < b
			}
			return sorted[i].Name < sorted[j].Name
		})
		for _, s := range sorted {
			rows = append(rows, row{s.Name, []string{shorten(s.Dir), s.Name, "c:" + s.Clients}})
		}

	case modeRepo:
		type grouped struct {
			repo string
			tree string
			s    Session
		}
		var all []grouped
		for _, s := range found {
			repo := mainWorktree(s.Dir)
			if repo == "" {
				all = append(all, grouped{"~", "", s})
				continue
			}
			tree := "."
			if filepath.Clean(s.Dir) != filepath.Clean(repo) {
				tree = label(s.Dir, filepath.Base(repo))
			}
			all = append(all, grouped{repoLabel(repo, root), tree, s})
		}
		// Sessions outside any repository sort last, under "~".
		sort.SliceStable(all, func(i, j int) bool {
			a, b := all[i], all[j]
			if (a.repo == "~") != (b.repo == "~") {
				return b.repo == "~"
			}
			if a.repo != b.repo {
				return a.repo < b.repo
			}
			if a.tree != b.tree {
				return a.tree < b.tree
			}
			return a.s.Name < b.s.Name
		})
		for _, g := range all {
			rows = append(rows, row{g.s.Name, []string{g.repo, g.tree, g.s.Name, "c:" + g.s.Clients}})
		}

	default:
		sorted := append([]Session(nil), found...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
		for _, s := range sorted {
			rows = append(rows, row{s.Name, []string{s.Name, "c:" + s.Clients, shorten(s.Dir)}})
		}
	}

	cells := make([][]string, len(rows))
	for i, r := range rows {
		cells[i] = r.cells
	}
	display := align(cells)

	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = r.name + "\t" + display[i]
	}
	return lines
}

// align pads every column to its widest cell, leaving the last ragged.
func align(rows [][]string) []string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for _, cells := range rows {
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}
	w.Flush()
	out := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	for i := range out {
		out[i] = strings.TrimRight(out[i], " ")
	}
	return out
}

// quote wraps s so a shell reads it as one word, whatever it holds.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// runCycle prints the actions that move command's picker one grouping along.
//
// A binding is fixed for the life of the picker, so tab cannot name the
// grouping it moves to. It asks here instead, and the grouping it moves from
// comes back out of $FZF_PROMPT.
func runCycle(forward bool, command string, list []string) error {
	self, err := os.Executable()
	if err != nil {
		self = "zmx-nav"
	}
	next := cycleMode(promptMode(os.Getenv("FZF_PROMPT"), list), forward, list)
	fmt.Printf("change-prompt(%s)+reload(%s %s --render %s)", prompt(next), quote(self), command, next)
	return nil
}

// runPick presents the session picker and attaches to the choice.
func runPick(mode, root string, render bool) error {
	found, err := sessions()
	if err != nil {
		return err
	}
	lines := renderPick(found, mode, root)

	if render {
		if len(lines) > 0 {
			fmt.Println(strings.Join(lines, "\n"))
		}
		return nil
	}
	if len(lines) == 0 {
		return fmt.Errorf("no sessions; `zmx-nav new` starts one")
	}

	self, err := os.Executable()
	if err != nil {
		self = "zmx-nav"
	}
	args := []string{
		"--delimiter=\t",
		"--with-nth=2",
		"--height=80%",
		"--no-sort",
		"--prompt=" + prompt(mode),
		"--header=enter attach | tab/shift-tab grouping",
		"--preview=zmx history {1}",
		"--preview-window=right:60%:follow",
		"--bind", "tab:transform:" + quote(self) + " pick --cycle next",
		"--bind", "btab:transform:" + quote(self) + " pick --cycle prev",
	}

	_, chosen, err := choose(lines, args)
	if err != nil {
		return err
	}
	name, _, _ := strings.Cut(chosen, "\t")
	return attach(name, "")
}
