# wt

A Go CLI for managing [git worktrees](https://git-scm.com/docs/git-worktree)
without thinking about paths — with a slick TUI
([bubbletea](https://github.com/charmbracelet/bubbletea) pickers,
[lipgloss](https://github.com/charmbracelet/lipgloss) tables) and first-class
integration with [herdr](https://herdr.dev), the terminal workspace manager
for AI coding agents.

Inspired by [tauantcamargo/wt](https://github.com/tauantcamargo/wt), ported to
Go and extended.

Worktrees for a repo `foo` are created as siblings of it, grouped in one
folder: `foo-worktrees/<branch>/` next to `foo/`. That keeps every worktree
out of the repo itself (nothing to `.gitignore`) and out of your way.

```
~/dev/
├── foo/                       # main checkout
└── foo-worktrees/
    ├── feature-login/         # `wt new feature/login`
    └── hotfix/                # `wt new hotfix main`
```

## Install

```sh
go install github.com/brianstarke/wt@latest
```

Requires git. [herdr](https://herdr.dev) is needed for `wt open` only.

### Shell integration (zsh)

`wt cd` and `wt root` print a path to stdout. To make them actually
change your shell's directory, add to `~/.zshrc`:

```sh
eval "$(wt init zsh)"
```

## Usage

```
wt new <branch> [base]   create (or attach to) a worktree, print its path
wt ls                    list worktrees for the current repo (pretty table)
wt cd [branch]           print a worktree path (interactive picker if no branch)
wt open [branch]         open a worktree as a herdr workspace with configured tabs
wt rm [branch] [-f]      remove a worktree (picker if no branch)
wt rmf [branch]          force-remove a worktree
wt rml [-f]              remove multiple worktrees (multi-select picker)
wt prune                 drop stale worktree metadata + empty sibling dir
wt root                  print the main repo checkout path
wt config                print config path, creating a default config if missing
```

`wt new` resolution order: an existing local branch is attached as-is;
otherwise, with `[base]` given, a new branch is created from it; otherwise, if
`origin/<branch>` exists, a new local branch is created tracking it; otherwise
a new branch is created off `HEAD`.

Every command works from inside any worktree, not just the main checkout —
paths are always resolved off the repo's shared `.git` dir.

`wt rm` refuses to remove the main worktree and offers to delete the local
branch once the worktree is gone.

## wt open + herdr

`wt open` picks a worktree, creates a herdr workspace for it (labeled
`{repo} ❯ {branch}`), and opens tabs from your config file
(`~/.config/wt/config.toml`, created on first run of `wt config`).

Default tabs:

```toml
[open]
workspace_label = "{repo} ❯ {branch}"
focus = true

[[open.tabs]]
label = "zsh"                 # plain shell in the worktree

[[open.tabs]]
label = "omp"
command = ["omp"]             # your AI coding agent

[[open.tabs]]
label = "nvim"
command = ["nvim", "."]       # editor
```

Add, remove, or reorder `[[open.tabs]]` entries to taste. `command` is
optional — omit it for a plain shell.

## Hook

Drop an executable `.worktree-hook` at the repo root and `wt new` runs it
right after creating a worktree — cwd is the new worktree, `$1` is the branch
name. Useful for copying `.env`, symlinking `node_modules`, or `direnv allow`:

```sh
#!/bin/sh
cp "$(git rev-parse --path-format=absolute --git-common-dir)/../.env" .
npm install
```

## License

MIT
