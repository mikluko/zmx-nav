package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// modeNew is the grouping that offers repositories rather than running
// sessions, so a switcher can reach a session that does not exist yet.
const modeNew = "new"

// switchModes are the groupings the switcher cycles through.
var switchModes = append(append([]string{}, modes...), modeNew)

// widen turns pick's `name\tdisplay` line into the switcher's
// `name\tdir\tdisplay`.
//
// The directory is empty for a running session: zmx records one only as the
// session is created, so naming it for a session already up would say
// something untrue about where it runs.
func widen(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		name, display, _ := strings.Cut(line, "\t")
		out[i] = name + "\t\t" + display
	}
	return out
}

// switchLines returns the picker's lines for mode and the mode they are for,
// which is not always the one asked for: a grouping holding nothing falls back
// to the repositories, since a pane with an empty picker in it has nowhere
// left to go.
func switchLines(live []Session, mode, root string) ([]string, string) {
	if mode != modeNew {
		if lines := widen(renderPick(live, mode, root)); len(lines) > 0 {
			return lines, mode
		}
	}
	running := make(map[string]bool, len(live))
	for _, s := range live {
		running[s.Name] = true
	}
	return renderNew(targets(root), running), modeNew
}

// runSwitchRender prints the picker's lines for mode, which is what the
// picker's own reload runs on tab.
func runSwitchRender(mode, root string) error {
	live, err := sessions()
	if err != nil {
		return err
	}
	if lines, _ := switchLines(live, mode, root); len(lines) > 0 {
		fmt.Println(strings.Join(lines, "\n"))
	}
	return nil
}

// switchArgs returns the fzf arguments for a picker showing mode.
func switchArgs(mode string) []string {
	self, err := os.Executable()
	if err != nil {
		self = "zmx-nav"
	}
	return []string{
		"--delimiter=\t",
		"--with-nth=3",
		"--height=100%",
		"--no-sort",
		"--prompt=" + prompt(mode),
		"--header=enter attach | tab/shift-tab grouping | esc close",
		// A directory in field two means the session has still to be created,
		// which is what tells a listing apart from a session's own scrollback.
		"--preview=[ -n {2} ] && lsd -la --color=always {2} || zmx history {1}",
		"--preview-window=right:60%:follow",
		"--bind", "tab:transform:" + quote(self) + " switch --cycle next",
		"--bind", "btab:transform:" + quote(self) + " switch --cycle prev",
	}
}

// without returns env less the entry for key.
func without(env []string, key string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if k, _, _ := strings.Cut(kv, "="); k != key {
			out = append(out, kv)
		}
	}
	return out
}

// attachChild runs a zmx client attached to name and waits for it, where
// attach replaces this process with one.
//
// The client returns when the session ends or the user detaches, and neither
// is this process's failure: a session's exit status is the session's own.
//
// ZMX_SESSION names the session a process is already inside, and zmx reads an
// attach made with it set as that session switching to the name given. A
// switcher started inside a session would otherwise redirect its host instead
// of attaching here.
func attachChild(name, dir string) error {
	path, err := exec.LookPath("zmx")
	if err != nil {
		return fmt.Errorf("zmx not found in PATH")
	}
	cmd := exec.Command(path, "attach", name)
	if dir != "" {
		cmd.Dir = filepath.Clean(dir)
	}
	cmd.Env = without(os.Environ(), "ZMX_SESSION")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	var exit *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &exit) {
		return err
	}
	return nil
}

// runSwitch makes the pane a slot rather than a session: it attaches to the
// chosen session and offers the picker again once that client is gone.
//
// zmx's client takes ctrl+\ before the PTY sees it, so the switch is reachable
// from inside whatever the session is running, which a shell binding is not.
// Leaving the picker ends the switcher and with it the pane, which costs
// nothing: a pane here is a viewport, and every session it showed outlives it.
func runSwitch(mode, root string) error {
	for {
		live, err := sessions()
		if err != nil {
			return err
		}
		lines, shown := switchLines(live, mode, root)
		if len(lines) == 0 {
			return fmt.Errorf("no sessions, and no repositories below %s", shorten(root))
		}

		chosen, err := runFzf(lines, switchArgs(shown))
		if errors.Is(err, errCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		parts := strings.SplitN(chosen, "\t", 3)
		if len(parts) < 2 {
			return fmt.Errorf("unreadable selection %q", chosen)
		}
		if err := attachChild(parts[0], parts[1]); err != nil {
			return err
		}
	}
}
