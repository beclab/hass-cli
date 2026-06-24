package config

import (
	"os"
	"path/filepath"
	"testing"
)

// clearEnv unsets the connection env vars so tests are deterministic.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"HASS_SERVER", "HASS_TOKEN", "HASS_SUPERVISOR_TOKEN", "HASS_INSECURE", "HASS_CLI_CONFIG_DIR"} {
		t.Setenv(k, "")
	}
}

func writeProfile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolvePrecedence(t *testing.T) {
	clearEnv(t)
	dir := writeProfile(t, `
default: home
profiles:
  home:
    server: http://profile:8123
    token: profile-token
    timeout: 30
`)
	t.Setenv("HASS_CLI_CONFIG_DIR", dir)

	// Profile only.
	cfg, err := Resolve("", "", "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "http://profile:8123" || cfg.Token != "profile-token" || cfg.TimeoutSeconds != 30 {
		t.Fatalf("profile not applied: %+v", cfg)
	}

	// Env overrides profile.
	t.Setenv("HASS_SERVER", "http://env:8123")
	t.Setenv("HASS_TOKEN", "env-token")
	cfg, err = Resolve("", "", "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "http://env:8123" || cfg.Token != "env-token" {
		t.Fatalf("env did not override profile: %+v", cfg)
	}

	// Flags override env.
	cfg, err = Resolve("", "http://flag:8123", "flag-token", false, 5)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "http://flag:8123" || cfg.Token != "flag-token" || cfg.TimeoutSeconds != 5 {
		t.Fatalf("flags did not override env: %+v", cfg)
	}
}

func TestResolveDefaultTimeout(t *testing.T) {
	clearEnv(t)
	t.Setenv("HASS_CLI_CONFIG_DIR", t.TempDir()) // no config.yaml -> no profile
	cfg, err := Resolve("", "http://h:8123", "tok", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TimeoutSeconds != 10 {
		t.Errorf("want default timeout 10, got %d", cfg.TimeoutSeconds)
	}
}

func TestResolveInsecureEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("HASS_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("HASS_INSECURE", "true")
	cfg, err := Resolve("", "https://h:8123", "tok", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Insecure {
		t.Error("HASS_INSECURE=true should set Insecure")
	}
}

func TestResolveMissingProfile(t *testing.T) {
	clearEnv(t)
	dir := writeProfile(t, `
default: home
profiles:
  home:
    server: http://h:8123
    token: t
`)
	t.Setenv("HASS_CLI_CONFIG_DIR", dir)
	if _, err := Resolve("nonexistent", "", "", false, 0); err == nil {
		t.Error("expected error for missing profile, got nil")
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"ok", Config{Server: "http://h:8123", Token: "t"}, false},
		{"https ok", Config{Server: "https://h:8123", Token: "t"}, false},
		{"no server", Config{Token: "t"}, true},
		{"no token", Config{Server: "http://h:8123"}, true},
		{"no scheme", Config{Server: "h:8123", Token: "t"}, true},
		{"wrong scheme", Config{Server: "ftp://h:8123", Token: "t"}, true},
		{"no host", Config{Server: "http://", Token: "t"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestURLDerivation(t *testing.T) {
	cases := []struct {
		server string
		rest   string
		ws     string
	}{
		{"http://h:8123", "http://h:8123/api", "ws://h:8123/api/websocket"},
		{"https://h:8123/", "https://h:8123/api", "wss://h:8123/api/websocket"},
	}
	for _, tc := range cases {
		c := &Config{Server: tc.server}
		if got := c.RESTBaseURL(); got != tc.rest {
			t.Errorf("RESTBaseURL(%q) = %q, want %q", tc.server, got, tc.rest)
		}
		if got := c.WebSocketURL(); got != tc.ws {
			t.Errorf("WebSocketURL(%q) = %q, want %q", tc.server, got, tc.ws)
		}
	}
}
