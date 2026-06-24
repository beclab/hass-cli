// Package config resolves hass-cli connection settings from flags, environment
// variables, and an optional on-disk profile file (in that precedence order).
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the resolved settings used to reach a Home Assistant instance.
type Config struct {
	Server          string `yaml:"server"`
	Token           string `yaml:"token"`
	SupervisorToken string `yaml:"supervisor_token"`
	Insecure        bool   `yaml:"insecure"`
	TimeoutSeconds  int    `yaml:"timeout"`
}

// profileFile is the on-disk shape: a set of named profiles plus a default.
type profileFile struct {
	Default  string            `yaml:"default"`
	Profiles map[string]Config `yaml:"profiles"`
}

// Resolve merges profile file < environment < explicit flag values. Flag values
// are only applied when non-zero so that lower-precedence sources show through.
func Resolve(profileName, server, token string, insecure bool, timeout int) (*Config, error) {
	cfg := &Config{TimeoutSeconds: 10}

	if fileCfg, name, err := loadProfile(profileName); err != nil {
		return nil, err
	} else if fileCfg != nil {
		cfg = fileCfg
		if cfg.TimeoutSeconds == 0 {
			cfg.TimeoutSeconds = 10
		}
		_ = name
	}

	if v := os.Getenv("HASS_SERVER"); v != "" {
		cfg.Server = v
	}
	if v := os.Getenv("HASS_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("HASS_SUPERVISOR_TOKEN"); v != "" {
		cfg.SupervisorToken = v
	}

	if server != "" {
		cfg.Server = server
	}
	if token != "" {
		cfg.Token = token
	}
	if insecure {
		cfg.Insecure = true
	}
	if timeout > 0 {
		cfg.TimeoutSeconds = timeout
	}

	return cfg, nil
}

// Validate ensures the minimum needed to talk to Home Assistant is present.
func (c *Config) Validate() error {
	if c.Server == "" {
		return errors.New("no server configured: set --server, HASS_SERVER, or a profile")
	}
	if c.Token == "" {
		return errors.New("no token configured: set --token, HASS_TOKEN, or a profile")
	}
	if _, err := url.Parse(c.Server); err != nil {
		return fmt.Errorf("invalid server URL %q: %w", c.Server, err)
	}
	return nil
}

// RESTBaseURL returns the normalized REST API base, e.g. https://host:8123/api.
func (c *Config) RESTBaseURL() string {
	base := strings.TrimRight(c.Server, "/")
	return base + "/api"
}

// WebSocketURL derives the ws(s) endpoint from the configured http(s) server.
func (c *Config) WebSocketURL() string {
	base := strings.TrimRight(c.Server, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + "/api/websocket"
}

func configDir() (string, error) {
	if dir := os.Getenv("HASS_CLI_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "hass-cli"), nil
}

// loadProfile reads the named profile (or the file default) if a config file
// exists. A missing file is not an error; it simply yields a nil config.
func loadProfile(name string) (*Config, string, error) {
	dir, err := configDir()
	if err != nil {
		return nil, "", nil
	}
	path := filepath.Join(dir, "config.yaml")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read config %s: %w", path, err)
	}

	var pf profileFile
	if err := yaml.Unmarshal(raw, &pf); err != nil {
		return nil, "", fmt.Errorf("parse config %s: %w", path, err)
	}

	if name == "" {
		name = pf.Default
	}
	if name == "" || len(pf.Profiles) == 0 {
		return nil, "", nil
	}
	cfg, ok := pf.Profiles[name]
	if !ok {
		return nil, "", fmt.Errorf("profile %q not found in %s", name, path)
	}
	return &cfg, name, nil
}
