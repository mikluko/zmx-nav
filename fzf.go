package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// errCancelled reports a picker the user dismissed, or a query that matched
// nothing. Neither is worth a message.
var errCancelled = errors.New("cancelled")

// runFzf runs fzf over lines and returns the chosen line.
//
// fzf's stderr stays attached: a rejected --bind is its only way to say so,
// and capturing it would make a broken picker look like an empty one.
func runFzf(lines []string, args []string) (string, error) {
	if _, err := exec.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("fzf not found in PATH")
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("no TTY for interactive selection")
	}
	tty.Close()

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", errCancelled
	}
	chosen := strings.Trim(string(out), "\n")
	if chosen == "" {
		return "", errCancelled
	}
	return chosen, nil
}

// attach replaces this process with a zmx client attached to name.
//
// dir becomes the session's working directory, but only where the session is
// being created: zmx records it once, when the daemon starts.
func attach(name, dir string) error {
	path, err := exec.LookPath("zmx")
	if err != nil {
		return fmt.Errorf("zmx not found in PATH")
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			return fmt.Errorf("chdir %s: %w", dir, err)
		}
	}
	return syscall.Exec(path, []string{"zmx", "attach", name}, os.Environ())
}
