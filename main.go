// Command zmx-nav selects and starts zmx sessions.
//
// zmx persists terminal sessions and deliberately provides no windows, tabs
// or splits. What it also provides no opinion about is which session you
// want, which is the gap this fills: `pick` moves between running sessions,
// `new` starts one in a repository or one of its worktrees, and `switch`
// holds a pane open over one session after another.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// version is stamped by the release build; `devel` means built from source.
var version = "devel"

const usage = `zmx-nav selects and starts zmx sessions.

Usage:
  zmx-nav pick [--mode flat|dir|repo]   Attach to a running session
  zmx-nav new                           Start one in a repository or worktree
  zmx-nav switch [--mode ...]           Hold this pane over one session at a time
  zmx-nav version                       Print the version

Options:
  --root DIR    Where repositories live (default ~/Forge, or $ZMX_NAV_ROOT)

In the picker: tab cycles the grouping, shift-tab goes back.
In a session under switch: ctrl+\ detaches and brings the picker back.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, errCancelled) {
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "zmx-nav: "+err.Error())
		os.Exit(1)
	}
}

func run(argv []string) error {
	if len(argv) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return errors.New("no command given")
	}

	command, rest := argv[0], argv[1:]
	switch command {
	case "pick":
		fs := flag.NewFlagSet("pick", flag.ContinueOnError)
		mode := fs.String("mode", modeRepo, "grouping: flat, dir or repo")
		render := fs.String("render", "", "print the lines for a grouping and exit; used by the picker's reload")
		cycle := fs.String("cycle", "", "print the actions for the next or previous grouping; used by the picker's tab")
		root := fs.String("root", defaultRoot(), "where repositories live")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		switch *cycle {
		case "":
		case "next":
			return runCycle(true, "pick", modes)
		case "prev":
			return runCycle(false, "pick", modes)
		default:
			return fmt.Errorf("unknown direction %q", *cycle)
		}
		if *render != "" {
			if !validMode(*render, modes) {
				return fmt.Errorf("unknown grouping %q", *render)
			}
			return runPick(*render, *root, true)
		}
		if !validMode(*mode, modes) {
			return fmt.Errorf("unknown grouping %q", *mode)
		}
		return runPick(*mode, *root, false)

	case "switch":
		fs := flag.NewFlagSet("switch", flag.ContinueOnError)
		root := fs.String("root", defaultRoot(), "where repositories live")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		return runSwitch(*root)

	case "new":
		fs := flag.NewFlagSet("new", flag.ContinueOnError)
		root := fs.String("root", defaultRoot(), "where repositories live")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		return runNew(*root)

	case "version", "--version", "-v":
		fmt.Println(version)
		return nil

	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	}

	fmt.Fprint(os.Stderr, usage)
	return fmt.Errorf("unknown command %q", command)
}

// defaultRoot is where repositories are looked for: $ZMX_NAV_ROOT, else
// ~/Forge.
func defaultRoot() string {
	if root := os.Getenv("ZMX_NAV_ROOT"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Forge")
}
