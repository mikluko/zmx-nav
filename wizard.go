package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// The wizard is one list at a time, descended with the right arrow and left
// with the left one. Nothing is bound to a chord: a pane is reached mid-task,
// from inside something else, and a key that has to be recalled is a key that
// is not pressed.
//
//	pick    flat        every session, by name
//	        worktree    sessions under the repository they belong to
//	create  session     a repository or a worktree git already records
//	        worktree    a worktree that does not exist yet
var tree = []menuItem{
	{name: "pick", hint: "attach to a session that is running", kids: []menuItem{
		{name: "flat", hint: "every session, by name"},
		{name: "worktree", hint: "grouped by repository, worktrees under theirs"},
	}},
	{name: "create", hint: "start a session that is not", kids: []menuItem{
		{name: "session", hint: "in a repository, or a worktree git records"},
		{name: "worktree", hint: "in a worktree made for it now"},
	}},
}

// menuItem is one line of a menu, and the menu below it where it has one.
type menuItem struct {
	name string
	hint string
	kids []menuItem
}

// errBack reports a step the user left by the left arrow or escape.
var errBack = errors.New("back")

// winsize is what TIOCGWINSZ fills in.
type winsize struct{ rows, cols, xpixel, ypixel uint16 }

// termSize returns the terminal's columns and rows, or zeroes where there is
// no terminal to ask.
func termSize() (cols, rows int) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return 0, 0
	}
	defer tty.Close()

	var ws winsize
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		tty.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	); errno != 0 {
		return 0, 0
	}
	return int(ws.cols), int(ws.rows)
}

// previewLayout returns the --preview-window for a terminal this size, or ""
// where the preview would leave the list too little to be read.
//
// The split follows the shape of the space rather than a preference: a wide
// terminal has room beside the list, a tall narrow one only below it. fzf
// carries one alternative of its own, which is what answers a resize made
// while the picker is up.
func previewLayout(cols, rows int) string {
	switch {
	case cols < 80 || rows < 15:
		return ""
	case cols >= 120:
		return "right,55%,border-left,<100(up,50%,border-bottom)"
	default:
		return "up,50%,border-bottom"
	}
}

// pickerArgs returns the fzf arguments every step of the wizard shares.
//
// The query sits at the top because that is where the eye already is: the list
// grows down from it, and a filter typed below the thing it filters reads
// backwards.
func pickerArgs(prompt, header string) []string {
	return []string{
		"--delimiter=\t",
		"--height=100%",
		"--layout=reverse",
		"--no-sort",
		"--no-mouse",
		"--prompt=" + prompt,
		"--header=" + header,
		"--expect=left,right",
	}
}

// withPreview appends a preview to args, where the terminal has room for one.
func withPreview(args []string, command string) []string {
	layout := previewLayout(termSize())
	if layout == "" {
		return args
	}
	return append(args, "--preview="+command, "--preview-window="+layout)
}

// step runs one step of the wizard over lines, showing field two onwards.
//
// The key comes back with the choice, since right and left are what move
// between steps and fzf reports them only as keys it was told to expect.
func step(lines []string, prompt, header, preview string) (key, chosen string, err error) {
	args := append(pickerArgs(prompt, header), "--with-nth=2..")
	if preview != "" {
		args = withPreview(args, preview)
	}
	return choose(lines, args)
}

// menuLines renders a menu as `name\tname  hint`.
func menuLines(items []menuItem) []string {
	cells := make([][]string, len(items))
	for i, item := range items {
		cells[i] = []string{item.name, item.hint}
	}
	display := align(cells)

	lines := make([]string, len(items))
	for i, item := range items {
		lines[i] = item.name + "\t" + display[i]
	}
	return lines
}

// chooseMenu presents items and returns the one chosen.
func chooseMenu(items []menuItem, prompt string) (menuItem, error) {
	key, chosen, err := step(menuLines(items), prompt, "↑↓ move   → enter   ← back", "")
	if err != nil {
		return menuItem{}, err
	}
	if key == "left" {
		return menuItem{}, errBack
	}
	name, _, _ := strings.Cut(chosen, "\t")
	for _, item := range items {
		if item.name == name {
			return item, nil
		}
	}
	return menuItem{}, fmt.Errorf("unreadable choice %q", chosen)
}

// wizard walks the menus and returns the session to attach to and the
// directory to create it in, which is empty for a session already running.
func wizard(root string) (name, dir string, err error) {
	var path []menuItem
	for {
		items := tree
		if len(path) > 0 {
			items = path[len(path)-1].kids
		}

		prompt := "zmx> "
		if len(path) > 0 {
			prompt = "zmx " + path[len(path)-1].name + "> "
		}

		chosen, err := chooseMenu(items, prompt)
		if errors.Is(err, errBack) {
			if len(path) == 0 {
				return "", "", errCancelled
			}
			path = path[:len(path)-1]
			continue
		}
		if err != nil {
			return "", "", err
		}
		if len(chosen.kids) > 0 {
			path = append(path, chosen)
			continue
		}

		name, dir, err = leaf(path[0].name, chosen.name, root)
		if errors.Is(err, errBack) {
			continue
		}
		return name, dir, err
	}
}

// steps are what the menu paths end in, keyed by the path that reaches them.
// A menu item with no entry here is a dead end, which is what the tests read
// this to check.
var steps = map[string]func(root string) (name, dir string, err error){
	"pick/flat":       func(root string) (string, string, error) { return chooseSession(modeFlat, root) },
	"pick/worktree":   func(root string) (string, string, error) { return chooseSession(modeRepo, root) },
	"create/session":  chooseTarget,
	"create/worktree": chooseNewWorktree,
}

// leaf runs the step a menu path ends in.
func leaf(branch, choice, root string) (name, dir string, err error) {
	run, ok := steps[branch+"/"+choice]
	if !ok {
		return "", "", fmt.Errorf("no step for %s/%s", branch, choice)
	}
	return run(root)
}

// chooseSession presents the running sessions in one grouping.
func chooseSession(mode, root string) (name, dir string, err error) {
	live, err := sessions()
	if err != nil {
		return "", "", err
	}
	lines := renderPick(live, mode, root)
	if len(lines) == 0 {
		return "", "", fmt.Errorf("no sessions running; create one")
	}

	key, chosen, err := step(lines, "zmx "+mode+"> ", "↑↓ move   ⏎ attach   ← back", "zmx history {1}")
	if err != nil {
		return "", "", err
	}
	if key == "left" {
		return "", "", errBack
	}
	name, _, _ = strings.Cut(chosen, "\t")
	return name, "", nil
}

// chooseTarget presents the repositories and the worktrees git records.
func chooseTarget(root string) (name, dir string, err error) {
	found := targets(root)
	if len(found) == 0 {
		return "", "", fmt.Errorf("no repositories below %s", shorten(root))
	}

	live, err := sessions()
	if err != nil {
		return "", "", err
	}
	running := make(map[string]bool, len(live))
	for _, s := range live {
		running[s.Name] = true
	}

	key, chosen, err := step(renderNew(found, running), "zmx create> ",
		"↑↓ move   ⏎ start or attach   ← back", "lsd -la --color=always {2}")
	if err != nil {
		return "", "", err
	}
	if key == "left" {
		return "", "", errBack
	}
	parts := strings.SplitN(chosen, "\t", 3)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("unreadable selection %q", chosen)
	}
	return parts[0], filepath.Clean(parts[1]), nil
}

// chooseNewWorktree adds a worktree to a chosen repository and names the
// session for it.
func chooseNewWorktree(root string) (name, dir string, err error) {
	repos := forgeRepos(root)
	if len(repos) == 0 {
		return "", "", fmt.Errorf("no repositories below %s", shorten(root))
	}

	cells := make([][]string, len(repos))
	for i, repo := range repos {
		cells[i] = []string{repoLabel(repo, root), shorten(repo)}
	}
	display := align(cells)
	lines := make([]string, len(repos))
	for i, repo := range repos {
		lines[i] = repo + "\t" + display[i]
	}

	key, chosen, err := step(lines, "zmx worktree> ",
		"↑↓ move   ⏎ pick the repository   ← back", "lsd -la --color=always {1}")
	if err != nil {
		return "", "", err
	}
	if key == "left" {
		return "", "", errBack
	}
	repo, _, _ := strings.Cut(chosen, "\t")

	label, err := promptText("branch> ", "a branch and a worktree, in "+repoLabel(repo, root))
	if err != nil {
		return "", "", err
	}

	tree, err := addWorktree(repo, label)
	if err != nil {
		return "", "", err
	}
	return sessionSafe(sessionSafe(repoLabel(repo, root)) + "@" + label), tree, nil
}

// addWorktree adds a worktree to repo on a new branch, beside the repository
// where the ones already there sit, and returns where it landed.
func addWorktree(repo, label string) (string, error) {
	dir := filepath.Join(repo, ".claude", "worktrees", filepath.Base(repo)+"-"+label)
	out, err := exec.Command("git", "-C", repo, "worktree", "add", "-b", label, dir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add: %s", strings.TrimSpace(string(out)))
	}
	return dir, nil
}
