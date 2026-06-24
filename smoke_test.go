package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bytetrade/hass-cli/cmd"
	"github.com/coder/websocket"
)

// mockHA stands up a minimal Home Assistant API (REST + WebSocket) so CLI
// commands can be smoke-tested end to end without a real instance.
func mockHA(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/" || r.URL.Path == "/api":
			writeJSON(w, map[string]any{"message": "API running."})
		case r.URL.Path == "/api/config":
			writeJSON(w, map[string]any{"version": "test", "location_name": "Mock"})
		case r.URL.Path == "/api/states":
			writeJSON(w, []map[string]any{
				{"entity_id": "sun.sun", "state": "above_horizon"},
				{"entity_id": "light.kitchen", "state": "off"},
				{"entity_id": "automation.demo", "state": "on"},
			})
		case strings.HasPrefix(r.URL.Path, "/api/states/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/states/")
			writeJSON(w, map[string]any{"entity_id": id, "state": "on"})
		case strings.HasPrefix(r.URL.Path, "/api/services/"):
			writeJSON(w, []any{})
		default:
			writeJSON(w, map[string]any{"ok": true})
		}
	})

	mux.HandleFunc("/api/websocket", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx := context.Background()
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"auth_required","ha_version":"test"}`))
		if _, _, err := c.Read(ctx); err != nil {
			return
		}
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"auth_ok","ha_version":"test"}`))

		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			var msg map[string]any
			_ = json.Unmarshal(data, &msg)
			id := msg["id"]
			var result any
			switch msg["type"] {
			case "get_config":
				result = map[string]any{"version": "test"}
			case "config/area_registry/list":
				result = []map[string]any{{"area_id": "kitchen", "name": "Kitchen"}}
			case "supervisor/api":
				result = map[string]any{"addons": []any{}}
			default:
				result = map[string]any{}
			}
			resp, _ := json.Marshal(map[string]any{
				"id": id, "type": "result", "success": true, "result": result,
			})
			_ = c.Write(ctx, websocket.MessageText, resp)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// run executes the root command with args and captures stdout.
func run(t *testing.T, args ...string) string {
	t.Helper()
	old := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp

	root := cmd.NewRootCommand()
	root.SetArgs(args)
	err := root.Execute()

	_ = wp.Close()
	os.Stdout = old
	out, _ := io.ReadAll(rp)
	if err != nil {
		t.Fatalf("command %v failed: %v\noutput: %s", args, err, out)
	}
	return string(out)
}

func TestSmoke(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"ping", []string{"-o", "json", "ping"}, "API running."},
		{"config", []string{"-o", "json", "config", "get"}, "Mock"},
		{"state-list", []string{"-o", "json", "state", "list"}, "light.kitchen"},
		{"state-get", []string{"-o", "json", "state", "get", "sun.sun"}, "sun.sun"},
		{"service-call", []string{"-o", "json", "service", "call", "light.turn_on", "--arguments", "entity_id=light.kitchen"}, ""},
		{"raw-api", []string{"-o", "json", "raw", "api", "GET", "states/sun.sun"}, "sun.sun"},
		{"raw-ws", []string{"-o", "json", "raw", "ws", "get_config"}, "version"},
		{"registry-area", []string{"-o", "json", "registry", "area", "list"}, "Kitchen"},
		{"workflow-list", []string{"-o", "json", "workflow", "automation", "list"}, "automation.demo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := run(t, tc.args...)
			if tc.want != "" && !strings.Contains(out, tc.want) {
				t.Fatalf("output missing %q\ngot: %s", tc.want, out)
			}
		})
	}
}
