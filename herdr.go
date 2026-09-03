package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// herdr is a thin client over the herdr CLI socket API.

type herdrEnvelope struct {
	Error  *struct{ Message string } `json:"error"`
	Result json.RawMessage           `json:"result"`
}

func herdr(args ...string) (json.RawMessage, error) {
	cmd := exec.Command("herdr", args...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("herdr %v: %s", args, errOut.String())
	}
	if len(bytes.TrimSpace(out.Bytes())) == 0 {
		return nil, nil // some commands (pane run) succeed silently
	}
	var env herdrEnvelope
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		return nil, fmt.Errorf("herdr %v: bad response: %s", args, out.String())
	}
	if env.Error != nil {
		return nil, fmt.Errorf("herdr: %s", env.Error.Message)
	}
	return env.Result, nil
}

type herdrWorkspace struct {
	Workspace struct {
		ID string `json:"workspace_id"`
	} `json:"workspace"`
	Tab struct {
		ID string `json:"tab_id"`
	} `json:"tab"`
	RootPane struct {
		ID string `json:"pane_id"`
	} `json:"root_pane"`
}

func herdrCreateWorkspace(cwd, label string, focus bool) (*herdrWorkspace, error) {
	args := []string{"workspace", "create", "--cwd", cwd, "--label", label}
	if focus {
		args = append(args, "--focus")
	} else {
		args = append(args, "--no-focus")
	}
	raw, err := herdr(args...)
	if err != nil {
		return nil, err
	}
	var ws herdrWorkspace
	if err := json.Unmarshal(raw, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

type herdrTab struct {
	Tab struct {
		ID string `json:"tab_id"`
	} `json:"tab"`
	RootPane struct {
		ID string `json:"pane_id"`
	} `json:"root_pane"`
}

func herdrCreateTab(workspaceID, cwd, label string) (*herdrTab, error) {
	raw, err := herdr("tab", "create",
		"--workspace", workspaceID,
		"--cwd", cwd,
		"--label", label,
		"--no-focus")
	if err != nil {
		return nil, err
	}
	var t herdrTab
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func herdrRenameTab(tabID, label string) error {
	_, err := herdr("tab", "rename", tabID, label)
	return err
}

func herdrPaneRun(paneID string, command []string) error {
	args := append([]string{"pane", "run", paneID}, command...)
	_, err := herdr(args...)
	return err
}
