# zmx-nav

Session navigation for [zmx](https://zmx.sh): pick a running session, or start
one in a repository or one of its worktrees.

zmx persists terminal sessions and, by design, provides no windows, tabs or
splits. What it also has no opinion about is *which* session you want. This
fills that gap and nothing else.

```sh
zmx-nav pick    # attach to a running session
zmx-nav new     # start one in a repository or worktree
zmx-nav switch  # hold this pane over one session after another
```

Both present [fzf](https://github.com/junegunn/fzf) and `exec` into
`zmx attach`.

## Install

```sh
brew install mikluko/tap/zmx-nav
```

Or from source:

```sh
go install github.com/mikluko/zmx-nav@latest
```

Needs `zmx` and `fzf` 0.46 or newer on `PATH`. `zmx-nav new` previews with
[`lsd`](https://github.com/lsd-rs/lsd).

## pick

Three groupings over the same sessions, cycled in place with `tab` without
leaving the picker. `shift-tab` goes back, and the prompt names the one in
view:

| Grouping | |
| --- | --- |
| flat | by session name |
| dir | by working directory |
| repo | by repository, worktrees under the repository they belong to |

```
mikluko/dotfiles   .            mikluko.dotfiles            c:0
mikluko/octant     918-arrival  mikluko.octant@918-arrival  c:0
mikluko/slopguard  heldout      mikluko.slopguard@heldout   c:0
~                               scratch                     c:0
```

The grouping is a leading column plus a sort, not a header row, because fzf
has no unselectable line. Every line stays selectable, and the group stays
matchable by typing it.

A binding is fixed for the life of the picker, so `tab` cannot name the
grouping it moves to. It reads the current one back out of the prompt and
prints the actions that reload the next.

The preview is `zmx history`, so a session shows its own scrollback.

Run from a prompt inside a session, this moves the pane rather than nesting a
client: zmx reads an attach made with `ZMX_SESSION` set as that session
switching, and leaves the session behind it running with no clients.

## new

Every repository two levels below the root, and every worktree git records for
one. A worktree is offered wherever it physically sits: beside the repository
under `.claude/worktrees/`, or far away under `~/.claude-squad/worktrees/`.

Sessions are named `org.repo`, and a worktree adds `@label`:

```
~/Forge/mikluko/dotfiles                             mikluko.dotfiles
~/Forge/uptime-com/up2-monitoring                    uptime-com.up2-monitoring
.../up2-monitoring/.claude/worktrees/up2-mon-rb-517  uptime-com.up2-monitoring@rb-517
```

The org qualifier is there because a repository name repeats across orgs, and
a bare name would send the second one into the first one's session. The
separator is a dot rather than a slash: **zmx names each session's unix socket
after it, so `zmx attach org/repo` creates nothing and says nothing.**

A name already running is attached rather than created, so `new` doubles as a
jump to a repository whose session is up.

## switch

`pick` and `new` hand the terminal over and are done. `switch` keeps the pane:
it attaches, and offers the picker again once that client is gone.

```sh
zmx-nav switch
```

The key is zmx's own. **ctrl+\\** detaches the current client, and the client
takes that byte before the PTY sees it, so the switch is reachable from inside
whatever the session is running — an editor, a build, a full-screen agent —
where a shell binding is not. Esc in the picker ends the switcher, and with it
the pane: what a pane holds here is a viewport, and every session it showed
outlives it.

Tab cycles a fourth grouping, the repositories from `new`, so a pane can reach
a session that does not exist yet. A grouping holding nothing gives way to it,
which is what the first pane of the day sees.

As a terminal's command this makes every pane a slot:

```
# Ghostty
command = /bin/zsh -lc "exec zmx-nav switch"
keybind = cmd+alt+s=text:\x1c
```

The keybind is optional and sends nothing but the detach byte, so the switch
can be a cmd chord instead of ctrl+\\. The login shell is what puts `zmx` and
`fzf` on `PATH`: Ghostty runs `command` through `/bin/sh -c`, which inherits
the GUI session's `PATH` rather than a shell's.

## Configuration

| | |
| --- | --- |
| `--root DIR` | Where repositories live. |
| `ZMX_NAV_ROOT` | The same, as an environment variable. |

The default is `~/Forge`. The layout expected below it is `<org>/<repo>`.

## Binding it to a key

zmx leaves window management to the terminal, so the natural home for these is
a terminal keybinding. In Ghostty, send a byte string the shell receives from
nothing else:

```
keybind = cmd+alt+d=csi:27;7;100~
keybind = cmd+alt+n=csi:27;7;110~
```

and in `.zshrc`, run the command on a fresh line, keeping whatever was
half-typed:

```zsh
function zmx-pick-widget() { zle push-input; BUFFER="zmx-nav pick"; zle accept-line }
function zmx-new-widget() { zle push-input; BUFFER="zmx-nav new"; zle accept-line }
zle -N zmx-pick-widget
zle -N zmx-new-widget
bindkey '\e[27;7;100~' zmx-pick-widget
bindkey '\e[27;7;110~' zmx-new-widget
```

## Why it reads git rather than running it

Worktrees come from `.git/worktrees/<id>/gitdir`, the record git writes, not
from `git worktree list`. At 84 repositories that is 84 directory reads
instead of 84 subprocesses, and it finds worktrees outside the repository the
same way as the ones beside it.

## License

MIT.
