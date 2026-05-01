package todo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type taskSnapshot struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type tabSnapshot struct {
	ID     int            `json:"id"`
	Name   string         `json:"name"`
	NextID int            `json:"next_id"`
	Tasks  []taskSnapshot `json:"tasks"`
}

type workspaceSnapshot struct {
	Version   int           `json:"version"`
	NextTabID int           `json:"next_tab_id"`
	ActiveIdx int           `json:"active_idx"`
	Tabs      []tabSnapshot `json:"tabs"`
}

// DefaultStatePath returns the platform-appropriate path for the state file.
// On Linux this is $XDG_CONFIG_HOME/todo/state.json (~/.config/todo/state.json).
// The file lives outside the repository so it is never accidentally committed.
func DefaultStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "todo", "state.json"), nil
}

// Load reads the workspace from path. If the file does not exist a fresh
// workspace is returned with no error, so first-run works without setup.
func Load(path string) (*Workspace, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewWorkspace(), nil
	}
	if err != nil {
		return nil, err
	}
	var snap workspaceSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return workspaceFromSnapshot(snap), nil
}

// Save writes the workspace to path atomically (write to tmp, then rename).
func Save(path string, ws *Workspace) error {
	snap := ws.toSnapshot()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
