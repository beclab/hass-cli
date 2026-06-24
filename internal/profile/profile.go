// Package profile owns the on-disk hass-cli profile index
// (~/.config/hass-cli/profiles.json) and the keychain-backed token store.
// Tokens are NOT kept in the index file; they live in the OS keychain via
// internal/keychain, keyed by profile name.
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// indexFilename is the profile index this package owns.
const indexFilename = "profiles.json"

const (
	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
)

// Entry is a single profile: a target Home Assistant instance plus its
// connection settings. The token is stored separately in the keychain
// (account = Name).
type Entry struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Insecure bool   `json:"insecure,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`

	// InstanceName and HAVersion are best-effort cached from /api/config on
	// login for display in `profile list`. Never authoritative.
	InstanceName string `json:"instanceName,omitempty"`
	HAVersion    string `json:"haVersion,omitempty"`
}

// Index is the on-disk schema of profiles.json.
type Index struct {
	CurrentProfile  string  `json:"current,omitempty"`
	PreviousProfile string  `json:"previous,omitempty"`
	Profiles        []Entry `json:"profiles,omitempty"`
}

// ConfigDir resolves the hass-cli config directory, honoring
// $HASS_CLI_CONFIG_DIR and falling back to os.UserConfigDir()/hass-cli. The
// directory is not created here.
func ConfigDir() (string, error) {
	if dir := os.Getenv("HASS_CLI_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "hass-cli"), nil
}

// IndexFile returns the absolute path to profiles.json without creating it.
func IndexFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, indexFilename), nil
}

// Find looks up a profile by name. Returns nil if no match.
func (i *Index) Find(name string) *Entry {
	if name == "" {
		return nil
	}
	for idx := range i.Profiles {
		if i.Profiles[idx].Name == name {
			return &i.Profiles[idx]
		}
	}
	return nil
}

// Current returns the active profile, or nil when there isn't one.
func (i *Index) Current() *Entry {
	if i.CurrentProfile == "" {
		if len(i.Profiles) == 0 {
			return nil
		}
		return &i.Profiles[0]
	}
	return i.Find(i.CurrentProfile)
}

// Upsert inserts or replaces a profile by name, preserving slot order.
func (i *Index) Upsert(e Entry) *Entry {
	for idx := range i.Profiles {
		if i.Profiles[idx].Name == e.Name {
			i.Profiles[idx] = e
			return &i.Profiles[idx]
		}
	}
	i.Profiles = append(i.Profiles, e)
	return &i.Profiles[len(i.Profiles)-1]
}

// SetCurrent flips Current/Previous, resolving "-" to the previous profile
// (a la `cd -`). Returns the newly-current profile.
func (i *Index) SetCurrent(name string) (*Entry, error) {
	if name == "-" {
		if i.PreviousProfile == "" {
			return nil, errors.New("no previous profile to switch back to")
		}
		name = i.PreviousProfile
	}
	target := i.Find(name)
	if target == nil {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	if i.CurrentProfile != target.Name {
		i.PreviousProfile = i.CurrentProfile
		i.CurrentProfile = target.Name
	}
	return target, nil
}

// Remove deletes a profile by name. If the removed profile was current, the
// current pointer falls back to Previous (when still valid) or the first
// remaining profile. Returns the removed profile and whether anything matched.
func (i *Index) Remove(name string) (*Entry, bool) {
	idx := -1
	for n := range i.Profiles {
		if i.Profiles[n].Name == name {
			idx = n
			break
		}
	}
	if idx == -1 {
		return nil, false
	}
	removed := i.Profiles[idx]
	i.Profiles = append(i.Profiles[:idx], i.Profiles[idx+1:]...)

	wasCurrent := i.CurrentProfile == removed.Name
	if i.PreviousProfile == removed.Name {
		i.PreviousProfile = ""
	}
	if wasCurrent {
		switch {
		case i.PreviousProfile != "" && i.Find(i.PreviousProfile) != nil:
			i.CurrentProfile = i.PreviousProfile
			i.PreviousProfile = ""
		case len(i.Profiles) > 0:
			i.CurrentProfile = i.Profiles[0].Name
		default:
			i.CurrentProfile = ""
		}
	}
	return &removed, true
}

// Load reads profiles.json. A missing file yields an empty Index (not an
// error) so first-run UX works.
func Load() (*Index, error) {
	path, err := IndexFile()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Index{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return &Index{}, nil
	}
	idx := &Index{}
	if err := json.Unmarshal(data, idx); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return idx, nil
}

// Save writes profiles.json atomically with 0600 perms, creating the parent
// directory if needed.
func Save(idx *Index) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	path := filepath.Join(dir, indexFilename)
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}
	return atomicWrite(path, data, filePerm)
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}
