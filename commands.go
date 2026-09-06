package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

func mustRepoRoot() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return root, nil
}

// --- wt new ----------------------------------------------------------------

var newCmd = &cobra.Command{
	Use:   "new <branch> [base]",
	Short: "create (or attach to) a worktree",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]
		var base string
		if len(args) > 1 {
			base = args[1]
		}
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		wdir := worktreesDir(root)
		if err := os.MkdirAll(wdir, 0o755); err != nil {
			return err
		}
		wtpath := filepath.Join(wdir, slug(branch))

		if _, err := os.Stat(wtpath); err == nil {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				infoStyle.Render("→ already exists:"),
				pathStyle.Render(wtpath))
			fmt.Println(wtpath)
			return nil
		}

		switch {
		case localBranchExists(root, branch):
			fmt.Fprintf(os.Stderr, "%s %s\n",
				infoStyle.Render("→ attaching existing branch"),
				branchStyle.Render(branch))
			if _, err := git("-C", root, "worktree", "add", wtpath, branch); err != nil {
				return err
			}
		case base != "":
			fmt.Fprintf(os.Stderr, "%s %s %s %s\n",
				infoStyle.Render("→ new branch"),
				branchStyle.Render(branch),
				dimStyle.Render("from"),
				branchStyle.Render(base))
			if _, err := git("-C", root, "worktree", "add", "-b", branch, wtpath, base); err != nil {
				return err
			}
		default:
			if ref := remoteBranchRef(root, branch); ref != "" {
				fmt.Fprintf(os.Stderr, "%s %s\n",
					infoStyle.Render("→ tracking"),
					branchStyle.Render(ref))
				if _, err := git("-C", root, "worktree", "add", "-b", branch, wtpath, ref); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(os.Stderr, "%s %s %s\n",
					infoStyle.Render("→ new branch"),
					branchStyle.Render(branch),
					dimStyle.Render("from HEAD"))
				if _, err := git("-C", root, "worktree", "add", "-b", branch, wtpath); err != nil {
					return err
				}
			}
		}

		if err := runHook(root, wtpath, branch); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", warnStyle.Render("⚠ hook failed: "+err.Error()))
		}

		fmt.Fprintf(os.Stderr, "%s %s\n",
			okStyle.Render("✓ ready:"),
			pathStyle.Render(wtpath))
		fmt.Println(wtpath)
		return nil
	},
}

// --- wt ls -----------------------------------------------------------------

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "list worktrees for the current repo",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		wts, err := listWorktrees(root)
		if err != nil {
			return err
		}

		cwd, _ := os.Getwd()
		if abs, err := filepath.Abs(cwd); err == nil {
			cwd = abs
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(accent)).
			StyleFunc(func(row, col int) lipgloss.Style {
				switch {
				case row == table.HeaderRow:
					return headerStyle.Padding(0, 1)
				case col == 0:
					return lipgloss.NewStyle().Padding(0, 1)
				default:
					return lipgloss.NewStyle().Padding(0, 1)
				}
			}).
			Headers("", "BRANCH", "HEAD", "PATH")

		for _, w := range wts {
			marker := " "
			if real, err := filepath.EvalSymlinks(w.Path); err == nil {
				if cwdReal, err := filepath.EvalSymlinks(cwd); err == nil &&
					(cwdReal == real || strings.HasPrefix(cwdReal, real+string(os.PathSeparator))) {
					marker = currentStyle.Render("●")
				}
			}
			head := w.Head
			if len(head) > 8 {
				head = head[:8]
			}
			name := w.Name()
			if isMainWorktree(root, w.Path) {
				name += dimStyle.Render(" (main)")
			}
			t.Row(marker, branchStyle.Render(name), headStyle.Render(head), pathStyle.Render(w.Path))
		}

		fmt.Println(banner())
		fmt.Println(t)
		return nil
	},
}

// --- wt cd / root ----------------------------------------------------------

func resolveWorktree(root, branch string) (*Worktree, error) {
	wdir := worktreesDir(root)
	if branch != "" {
		target := filepath.Join(wdir, slug(branch))
		if st, err := os.Stat(target); err != nil || !st.IsDir() {
			return nil, fmt.Errorf("no worktree for %q (looked in %s)", branch, target)
		}
		wts, _ := listWorktrees(root)
		for _, w := range wts {
			if w.Path == target {
				return &w, nil
			}
		}
		return &Worktree{Path: target}, nil
	}
	wts, err := listWorktrees(root)
	if err != nil {
		return nil, err
	}
	wt := pickWorktree("jump to worktree", wts, wdir)
	if wt == nil {
		return nil, fmt.Errorf("no worktree selected")
	}
	return wt, nil
}

var cdCmd = &cobra.Command{
	Use:   "cd [branch]",
	Short: "print a worktree path (picker if no branch) — use with wt init zsh",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		branch := ""
		if len(args) > 0 {
			branch = args[0]
		}
		wt, err := resolveWorktree(root, branch)
		if err != nil {
			return err
		}
		fmt.Println(wt.Path)
		return nil
	},
}

var rootPathCmd = &cobra.Command{
	Use:     "root",
	Aliases: []string{"main"},
	Short:   "print the main repo checkout path",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		fmt.Println(root)
		return nil
	},
}

// --- wt rm / rmf / rml -----------------------------------------------------

func removeWorktree(root, wdir string, wt *Worktree, force bool) error {
	if isMainWorktree(root, wt.Path) {
		return fmt.Errorf("refusing to remove the main worktree")
	}
	branch := wt.Branch

	args := []string{"-C", root, "worktree", "remove", wt.Path}
	if force {
		args = append(args, "--force")
	}
	if _, err := git(args...); err != nil {
		if !force {
			return fmt.Errorf("%s has uncommitted changes — rerun with -f to discard them", branch)
		}
		return err
	}
	fmt.Fprintf(os.Stderr, "%s %s\n",
		okStyle.Render("✓ removed"),
		pathStyle.Render(wt.Path))

	if branch != "" && localBranchExists(root, branch) {
		if confirm(fmt.Sprintf("delete local branch \"%s\" too?", branchStyle.Render(branch))) {
			if _, err := git("-C", root, "branch", "-d", branch); err != nil {
				_, _ = git("-C", root, "branch", "-D", branch)
			}
			fmt.Fprintf(os.Stderr, "%s %s\n",
				okStyle.Render("✓ deleted branch"),
				branchStyle.Render(branch))
		}
	}
	// drop sibling dir when empty
	_ = os.Remove(wdir)
	return nil
}

var rmForce bool

var rmCmd = &cobra.Command{
	Use:     "rm [branch]",
	Aliases: []string{"remove"},
	Short:   "remove a worktree (picker if no branch)",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		wdir := worktreesDir(root)

		var wt *Worktree
		if len(args) > 0 {
			wt, err = resolveWorktree(root, args[0])
			if err != nil {
				return err
			}
		} else {
			wts, err := listWorktrees(root)
			if err != nil {
				return err
			}
			var nonMain []Worktree
			for _, w := range wts {
				if !isMainWorktree(root, w.Path) {
					nonMain = append(nonMain, w)
				}
			}
			wt = pickWorktree("remove worktree", nonMain, wdir)
			if wt == nil {
				return fmt.Errorf("no worktree selected")
			}
		}

		// back out if standing inside the target
		if cwd, err := os.Getwd(); err == nil {
			if real, err := filepath.EvalSymlinks(wt.Path); err == nil {
				if cwdReal, err := filepath.EvalSymlinks(cwd); err == nil &&
					(cwdReal == real || strings.HasPrefix(cwdReal, real+string(os.PathSeparator))) {
					fmt.Fprintf(os.Stderr, "%s\n",
						warnStyle.Render("⚠ you are inside the worktree being removed — shell stays put; cd out first"))
				}
			}
		}

		return removeWorktree(root, wdir, wt, rmForce)
	},
}

var rmfCmd = &cobra.Command{
	Use:   "rmf [branch]",
	Short: "force-remove a worktree (picker if no branch)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rmForce = true
		return rmCmd.RunE(cmd, args)
	},
}

var rmlForce bool

var rmlCmd = &cobra.Command{
	Use:   "rml",
	Short: "remove multiple worktrees (multi-select picker)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		wdir := worktreesDir(root)
		wts, err := listWorktrees(root)
		if err != nil {
			return err
		}
		var nonMain []Worktree
		for _, w := range wts {
			if !isMainWorktree(root, w.Path) {
				nonMain = append(nonMain, w)
			}
		}
		picked := multiPickWorktrees("remove worktrees (space to toggle, enter to confirm)", nonMain, wdir)
		if len(picked) == 0 {
			return fmt.Errorf("no worktrees selected")
		}
		force := rmlForce
		if !force {
			force = confirm(fmt.Sprintf("force-remove all %d selected worktree(s) (discards uncommitted changes)?", len(picked)))
		}
		failed := false
		for i := range picked {
			if err := removeWorktree(root, wdir, &picked[i], force); err != nil {
				fmt.Fprintln(os.Stderr, errStyle.Render("✗ "+err.Error()))
				failed = true
			}
		}
		if failed {
			return fmt.Errorf("some worktrees could not be removed")
		}
		return nil
	},
}

// --- wt prune ---------------------------------------------------------------

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "drop stale worktree metadata + empty sibling dir",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := mustRepoRoot()
		if err != nil {
			return err
		}
		out, err := git("-C", root, "worktree", "prune", "-v")
		if err != nil {
			return err
		}
		if out != "" {
			fmt.Println(out)
		}
		wdir := worktreesDir(root)
		if err := os.Remove(wdir); err == nil {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				okStyle.Render("✓ removed empty"),
				pathStyle.Render(wdir))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", okStyle.Render("✓ nothing stale"))
		}
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "discard uncommitted changes")
	rmlCmd.Flags().BoolVarP(&rmlForce, "force", "f", false, "skip the force-all prompt")
}
