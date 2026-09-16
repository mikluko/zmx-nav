package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Target is a place a session can be started, and the name it gets there.
type Target struct {
	Name string
	Dir  string
}

// targets returns every repository below root and every worktree git records
// for one, in repository order.
func targets(root string) []Target {
	var found []Target
	for _, repo := range forgeRepos(root) {
		name := sessionSafe(repoLabel(repo, root))
		found = append(found, Target{Name: name, Dir: repo})
		for _, w := range worktreesOf(repo) {
			found = append(found, Target{Name: sessionSafe(name + "@" + w.Label), Dir: w.Dir})
		}
	}
	return found
}

// renderNew returns one `name\tdir\tdisplay` line per target.
//
// The directory rides the line so the preview can list it and the chosen
// session can be created there without resolving the name back to a path.
func renderNew(found []Target, running map[string]bool) []string {
	cells := make([][]string, len(found))
	for i, t := range found {
		state := ""
		if running[t.Name] {
			state = "running"
		}
		cells[i] = []string{t.Name, state, shorten(t.Dir)}
	}
	display := align(cells)

	lines := make([]string, len(found))
	for i, t := range found {
		lines[i] = strings.Join([]string{t.Name, t.Dir, display[i]}, "\t")
	}
	return lines
}

// runNew presents the repository picker and starts or attaches to a session.
func runNew(root string) error {
	found := targets(root)
	if len(found) == 0 {
		return fmt.Errorf("no repositories below %s", shorten(root))
	}

	live, err := sessions()
	if err != nil {
		return err
	}
	running := make(map[string]bool, len(live))
	for _, s := range live {
		running[s.Name] = true
	}

	_, chosen, err := choose(renderNew(found, running), []string{
		"--delimiter=\t",
		"--with-nth=3",
		"--height=80%",
		"--no-sort",
		"--prompt=zmx new> ",
		"--header=enter start or attach",
		"--preview=lsd -la --color=always {2}",
		"--preview-window=right:55%",
	})
	if err != nil {
		return err
	}
	parts := strings.SplitN(chosen, "\t", 3)
	if len(parts) < 2 {
		return fmt.Errorf("unreadable selection %q", chosen)
	}
	return attach(parts[0], filepath.Clean(parts[1]))
}
