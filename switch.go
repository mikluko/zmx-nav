package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

// runSwitch makes the pane a slot rather than a session: it walks the wizard,
// attaches to what was chosen, and walks it again once that client is gone.
//
// zmx's client takes ctrl+\ before the PTY sees it, so the switch is reachable
// from inside whatever the session is running, which a shell binding is not.
// Leaving the wizard ends the switcher and with it the pane, which costs
// nothing: a pane here is a viewport, and every session it showed outlives it.
func runSwitch(root string) error {
	for {
		name, dir, err := wizard(root)
		if errors.Is(err, errCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := attachChild(name, dir); err != nil {
			return err
		}
	}
}
