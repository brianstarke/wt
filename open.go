package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [branch]",
	Short: "open a worktree as a herdr workspace with configured tabs",
	Long: `Pick a worktree (or pass a branch), create a herdr workspace for it,
and open tabs from your config file (` + configPath() + `).

Default tabs: zsh (shell in worktree), omp (agent), nvim (editor).`,
	Args: cobra.MaximumNArgs(1),
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

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("config %s: %w", configPath(), err)
		}

		repo := filepath.Base(root)
		name := wt.Branch
		if name == "" {
			name = filepath.Base(wt.Path)
		}
		label := strings.NewReplacer("{repo}", repo, "{branch}", name).
			Replace(cfg.Open.WorkspaceLabel)

		focus := true
		if cfg.Open.Focus != nil {
			focus = *cfg.Open.Focus
		}

		fmt.Fprintf(os.Stderr, "%s %s\n",
			infoStyle.Render("→ workspace:"),
			lipgloss.NewStyle().Foreground(accent2).Bold(true).Render(label))

		ws, err := herdrCreateWorkspace(wt.Path, label, focus)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%s %s %s\n",
			okStyle.Render("✓ workspace"),
			dimStyle.Render(ws.Workspace.ID),
			pathStyle.Render(wt.Path))

		for i, tab := range cfg.Open.Tabs {
			var paneID string
			if i == 0 {
				// reuse the workspace's root tab
				if err := herdrRenameTab(ws.Tab.ID, tab.Label); err != nil {
					return fmt.Errorf("tab %q: %w", tab.Label, err)
				}
				paneID = ws.RootPane.ID
			} else {
				t, err := herdrCreateTab(ws.Workspace.ID, wt.Path, tab.Label)
				if err != nil {
					return fmt.Errorf("tab %q: %w", tab.Label, err)
				}
				paneID = t.RootPane.ID
			}
			fmt.Fprintf(os.Stderr, "  %s %s\n",
				okStyle.Render("✓ tab"),
				branchStyle.Render(tab.Label))

			if len(tab.Command) > 0 {
				if err := herdrPaneRun(paneID, tab.Command); err != nil {
					return fmt.Errorf("tab %q: run %v: %w", tab.Label, tab.Command, err)
				}
				fmt.Fprintf(os.Stderr, "    %s %s\n",
					dimStyle.Render("↳"),
					headStyle.Render(strings.Join(tab.Command, " ")))
			}
		}
		return nil
	},
}
