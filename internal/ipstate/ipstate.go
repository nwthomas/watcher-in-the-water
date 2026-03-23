package ipstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// State is persisted JSON for deploy-safe comparison of the last known public IP.
type State struct {
	PublicIP  string    `json:"public_ip"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Load reads state from path. If the file is missing, returns empty State and nil error.
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

// Save writes state atomically (temp file in same directory + rename).
func Save(path string, s State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
