package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "git worktree manager with a slick TUI",
	Long: banner() + `
Worktrees for a repo "foo" live as siblings of it, grouped in one folder:
foo-worktrees/<branch>/ next to foo/.

Commands that "cd" (cd/root) print the target path to stdout so shell
integration can cd for you. Run 'wt init zsh' for the wrapper.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, errStyle.Render("✗ "+err.Error()))
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		newCmd,
		lsCmd,
		cdCmd,
		rmCmd,
		rmfCmd,
		rmlCmd,
		pruneCmd,
		rootPathCmd,
		openCmd,
		configCmd,
		initCmd,
	)
}
