package main

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Tab describes one herdr tab opened by `wt open`.
type Tab struct {
	Label   string   `toml:"label"`
	Command []string `toml:"command"` // empty = plain shell
}

// OpenConfig controls `wt open` behavior.
type OpenConfig struct {
	// WorkspaceLabel supports {repo} and {branch} placeholders.
	WorkspaceLabel string `toml:"workspace_label"`
	Focus          *bool  `toml:"focus"` // focus new workspace, default true
	Tabs           []Tab  `toml:"tabs"`
}

type Config struct {
	Open OpenConfig `toml:"open"`
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "wt", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wt", "config.toml")
}

func defaultConfig() Config {
	return Config{Open: OpenConfig{
		WorkspaceLabel: "{repo} ❯ {branch}",
		Tabs: []Tab{
			{Label: "zsh"},
			{Label: "omp", Command: []string{"omp"}},
			{Label: "nvim", Command: []string{"nvim", "."}},
		},
	}}
}

// loadConfig reads the config file, layering it over defaults. Missing file
// returns pure defaults.
func loadConfig() (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg, nil // missing config is fine
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if len(cfg.Open.Tabs) == 0 {
		cfg.Open.Tabs = defaultConfig().Open.Tabs
	}
	if cfg.Open.WorkspaceLabel == "" {
		cfg.Open.WorkspaceLabel = defaultConfig().Open.WorkspaceLabel
	}
	return cfg, nil
}

// writeDefaultConfig creates the config file with defaults if absent.
// Returns path and whether it was created.
func writeDefaultConfig() (string, bool, error) {
	p := configPath()
	if _, err := os.Stat(p); err == nil {
		return p, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return p, false, err
	}
	const sample = `# wt configuration
# https://github.com/starke/wt

[open]
# Label for the herdr workspace. Placeholders: {repo}, {branch}.
workspace_label = "{repo} ❯ {branch}"
# Focus the new workspace after creating it.
focus = true

# Tabs created inside the workspace, in order.
# The first tab reuses the workspace's root tab.
# command = [] or omitted opens a plain shell in the worktree directory.

[[open.tabs]]
label = "zsh"

[[open.tabs]]
label = "omp"
command = ["omp"]

[[open.tabs]]
label = "nvim"
command = ["nvim", "."]
`
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		return p, false, err
	}
	return p, true, nil
}
