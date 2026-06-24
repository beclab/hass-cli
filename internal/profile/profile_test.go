package profile

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/beclab/hass-cli/internal/keychain/keychainfake"
)

func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func TestIndexUpsertCurrentSwitch(t *testing.T) {
	idx := &Index{}
	idx.Upsert(Entry{Name: "home", Server: "http://home:8123"})
	idx.Upsert(Entry{Name: "work", Server: "http://work:8123"})
	idx.CurrentProfile = "home" // login sets this explicitly

	if cur := idx.Current(); cur == nil || cur.Name != "home" {
		t.Fatalf("expected first profile as default current, got %v", cur)
	}

	if _, err := idx.SetCurrent("work"); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
	if idx.CurrentProfile != "work" || idx.PreviousProfile != "home" {
		t.Fatalf("unexpected current/previous: %q/%q", idx.CurrentProfile, idx.PreviousProfile)
	}

	// "-" switches back to home.
	if _, err := idx.SetCurrent("-"); err != nil {
		t.Fatalf("SetCurrent(-): %v", err)
	}
	if idx.CurrentProfile != "home" {
		t.Fatalf("expected switch back to home, got %q", idx.CurrentProfile)
	}
}

func TestIndexRemoveFallback(t *testing.T) {
	idx := &Index{}
	idx.Upsert(Entry{Name: "a"})
	idx.Upsert(Entry{Name: "b"})
	_, _ = idx.SetCurrent("b")

	removed, ok := idx.Remove("b")
	if !ok || removed.Name != "b" {
		t.Fatalf("remove b: ok=%v removed=%v", ok, removed)
	}
	// Current should fall back (previous was "a").
	if idx.CurrentProfile != "a" {
		t.Fatalf("expected fallback to a, got %q", idx.CurrentProfile)
	}

	if _, ok := idx.Remove("missing"); ok {
		t.Fatalf("removing a missing profile should report ok=false")
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	t.Setenv("HASS_CLI_CONFIG_DIR", t.TempDir())

	idx := &Index{}
	idx.Upsert(Entry{Name: "home", Server: "http://home:8123", HAVersion: "2025.1"})
	idx.CurrentProfile = "home"
	if err := Save(idx); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.CurrentProfile != "home" || len(got.Profiles) != 1 || got.Profiles[0].Server != "http://home:8123" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	t.Setenv("HASS_CLI_CONFIG_DIR", t.TempDir())
	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Profiles) != 0 {
		t.Fatalf("expected empty index, got %+v", got)
	}
}

func TestTokenStoreAndStatus(t *testing.T) {
	store := NewTokenStoreWith(keychainfake.New())

	if _, err := store.Get("home"); err != ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
	if got := Status(store, "home", time.Now()); got != "never" {
		t.Fatalf("status before login: want never, got %q", got)
	}

	// A token with no exp claim → "logged-in".
	noExp := "aaa." + b64("{}") + ".sig"
	if err := store.Set("home", noExp); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := Status(store, "home", time.Now()); got != "logged-in" {
		t.Fatalf("status no-exp: want logged-in, got %q", got)
	}

	// An expired token.
	expired := "aaa." + b64(`{"exp":1000000000}`) + ".sig"
	_ = store.Set("home", expired)
	if got := Status(store, "home", time.Now()); got != "expired" {
		t.Fatalf("status expired: want expired, got %q", got)
	}

	_ = store.Delete("home")
	if got := Status(store, "home", time.Now()); got != "never" {
		t.Fatalf("status after delete: want never, got %q", got)
	}
}
