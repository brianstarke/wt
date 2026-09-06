package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "show config path, creating a default config if missing",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, created, err := writeDefaultConfig()
		if err != nil {
			return err
		}
		if created {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				okStyle.Render("✓ created"),
				pathStyle.Render(p))
		}
		fmt.Println(p)
		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init zsh",
	Short: "print shell integration for zsh (enables real cd on new/cd/root)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "zsh" {
			return fmt.Errorf("only zsh is supported for now")
		}
		fmt.Print(`# wt shell integration — eval "$(wt init zsh)"
# Wraps the wt binary so ` + "`wt cd`" + `, ` + "`wt root`" + ` actually change directory.
wt() {
  case "$1" in
    cd|root|main)
      local dest
      dest=$(command wt "$@") || return
      # empty output = picker aborted, stay put
      if [ -n "$dest" ] && [ -d "$dest" ]; then
        cd "$dest"
      fi
      ;;
    *)
      command wt "$@"
      ;;
  esac
}
`)
		return nil
	},
}
