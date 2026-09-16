package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	fzf "github.com/junegunn/fzf/src"
)

// errCancelled reports a picker the user dismissed, or a query that matched
// nothing. Neither is worth a message.
var errCancelled = errors.New("cancelled")

// runFzf presents lines and returns what fzf wrote and the code it ended on.
//
// fzf runs in this process rather than as a command: the picker then shares
// the terminal it was started in, and a choice comes back as a value rather
// than as bytes on a pipe that have to be read back in the right order.
func runFzf(lines []string, args []string) (int, []string, error) {
	opts, err := fzf.ParseOptions(true, args)
	if err != nil {
		return 0, nil, err
	}

	in := make(chan string, len(lines)+1)
	for _, line := range lines {
		in <- line
	}
	close(in)
	opts.Input = in

	// Buffered past anything fzf writes: --expect adds a key line and
	// --print-query a query line, and a reader started beside Run would have
	// to outlive a panic in it to drain them.
	out := make(chan string, len(lines)+8)
	opts.Output = out

	code, err := fzf.Run(opts)
	close(out)
	if err != nil {
		return code, nil, err
	}

	var written []string
	for line := range out {
		written = append(written, line)
	}
	return code, written, nil
}

// choose runs one step over lines and returns the key pressed, empty for
// enter, and the line it was pressed on.
func choose(lines []string, args []string) (key, chosen string, err error) {
	code, written, err := runFzf(lines, args)
	if err != nil {
		return "", "", err
	}
	switch code {
	case fzf.ExitOk:
	case fzf.ExitInterrupt, fzf.ExitNoMatch:
		return "", "", errCancelled
	default:
		return "", "", fmt.Errorf("picker exited %d", code)
	}
	if len(written) < 2 {
		return "", "", errCancelled
	}
	return written[0], strings.Trim(written[len(written)-1], "\n"), nil
}

// promptText reads a line typed into an empty picker.
//
// Enter on a query matching nothing is how fzf reports typed text, so the
// no-match exit is the answer here rather than a failure; escape is not.
func promptText(prompt, header string) (string, error) {
	code, written, err := runFzf(nil, []string{
		"--height=100%",
		"--layout=reverse",
		"--no-info",
		"--print-query",
		"--prompt=" + prompt,
		"--header=" + header,
	})
	if err != nil {
		return "", err
	}
	if code == fzf.ExitInterrupt {
		return "", errBack
	}
	if len(written) == 0 {
		return "", errBack
	}
	text := strings.TrimSpace(written[0])
	if text == "" {
		return "", errBack
	}
	return text, nil
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
