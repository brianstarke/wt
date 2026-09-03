package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree is one entry of `git worktree list --porcelain`.
type Worktree struct {
	Path   string
	Head   string
	Branch string // short name, empty if detached
	Bare   bool
}

func (w Worktree) Name() string {
	if w.Branch != "" {
		return w.Branch
	}
	if len(w.Head) >= 7 {
		return w.Head[:7]
	}
	return w.Head
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// repoRoot resolves the main checkout root off the shared .git dir,
// so it works from inside any linked worktree.
func repoRoot() (string, error) {
	common, err := git("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return filepath.Dir(common), nil
}

func worktreesDir(root string) string {
	return filepath.Join(filepath.Dir(root), filepath.Base(root)+"-worktrees")
}

func slug(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

func listWorktrees(root string) ([]Worktree, error) {
	out, err := git("-C", root, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var wts []Worktree
	var cur *Worktree
	flush := func() {
		if cur != nil {
			wts = append(wts, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur = &Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case cur == nil:
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			cur.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			cur.Bare = true
		}
	}
	flush()
	return wts, nil
}

func localBranchExists(root, branch string) bool {
	_, err := git("-C", root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// remoteBranchRef finds a remote-tracking ref whose tail matches branch,
// preferring origin when several remotes have it.
func remoteBranchRef(root, branch string) string {
	out, err := git("-C", root, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	if err != nil {
		return ""
	}
	var matches []string
	for _, ref := range strings.Split(out, "\n") {
		if i := strings.Index(ref, "/"); i >= 0 && ref[i+1:] == branch {
			matches = append(matches, ref)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	preferred := "origin/" + branch
	for _, m := range matches {
		if m == preferred {
			return m
		}
	}
	return matches[0]
}

// isMainWorktree reports whether path is the main checkout.
func isMainWorktree(root, path string) bool {
	r, err1 := filepath.EvalSymlinks(root)
	p, err2 := filepath.EvalSymlinks(path)
	if err1 != nil || err2 != nil {
		return false
	}
	return r == p
}

func runHook(root, wtpath, branch string) error {
	hook := filepath.Join(root, ".worktree-hook")
	st, err := os.Stat(hook)
	if err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
		return nil
	}
	fmt.Fprintln(os.Stderr, infoStyle.Render("⚡ running .worktree-hook"))
	cmd := exec.Command(hook, branch)
	cmd.Dir = wtpath
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
