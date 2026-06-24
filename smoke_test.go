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
	return mockHAWith(t, []string{"api", "frontend", "hassio"})
}

// mockHAWith is mockHA with a configurable component list, so tests can model
// a non-Supervisor (no "hassio") install.
func mockHAWith(t *testing.T, components []string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/" || r.URL.Path == "/api":
			writeJSON(w, map[string]any{"message": "API running."})
		case r.URL.Path == "/api/config":
			writeJSON(w, map[string]any{
				"version":       "test",
				"location_name": "Mock",
				"components":    components,
			})
		case r.URL.Path == "/api/config/config_entries/flow_handlers":
			writeJSON(w, []string{"random", "group", "template"})
		case r.URL.Path == "/api/services":
			writeJSON(w, []map[string]any{{
				"domain": "light",
				"services": map[string]any{
					"turn_on": map[string]any{
						"name":        "Turn on",
						"description": "Turn a light on",
						"fields": map[string]any{
							"brightness_pct": map[string]any{"description": "Brightness %"},
						},
					},
				},
			}})
		case r.URL.Path == "/api/error_log":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("no errors\n"))
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

			// system_health/info is subscription-style: ack with an empty
			// result, then stream an "initial" snapshot and a "finish" event.
			if msg["type"] == "system_health/info" {
				ack, _ := json.Marshal(map[string]any{"id": id, "type": "result", "success": true})
				_ = c.Write(ctx, websocket.MessageText, ack)
				initial, _ := json.Marshal(map[string]any{
					"id": id, "type": "event",
					"event": map[string]any{
						"type": "initial",
						"data": map[string]any{
							"homeassistant": map[string]any{"info": map[string]any{"version": "test"}},
						},
					},
				})
				_ = c.Write(ctx, websocket.MessageText, initial)
				finish, _ := json.Marshal(map[string]any{
					"id": id, "type": "event", "event": map[string]any{"type": "finish"},
				})
				_ = c.Write(ctx, websocket.MessageText, finish)
				continue
			}

			var result any
			switch msg["type"] {
			case "get_config":
				result = map[string]any{"version": "test"}
			case "config/area_registry/list":
				result = []map[string]any{{"area_id": "kitchen", "name": "Kitchen"}}
			case "input_boolean/list":
				result = []map[string]any{{"id": "guest_mode", "name": "Guest Mode"}}
			case "input_boolean/create":
				result = map[string]any{"id": "guest_mode", "name": msg["name"]}
			case "config_entries/get":
				result = []map[string]any{{"entry_id": "abc123", "domain": "sun", "title": "Sun", "state": "loaded"}}
			case "backup/info":
				result = map[string]any{"state": "idle", "backups": []any{}}
			case "repairs/list_issues":
				result = map[string]any{"issues": []any{}}
			case "hardware/info":
				result = map[string]any{"hardware": []any{}}
			case "analytics":
				result = map[string]any{"preferences": map[string]any{"base": false}}
			case "labs/list":
				result = map[string]any{"features": []any{map[string]any{"domain": "frontend", "preview_feature": "winter_mode", "enabled": false}}}
			case "lovelace/dashboards/list":
				result = []map[string]any{{"id": "ops_room", "url_path": "ops-room", "title": "Ops Room", "mode": "storage"}}
			case "lovelace/resources":
				result = []any{}
			case "assist_pipeline/pipeline/list":
				result = map[string]any{
					"pipelines":          []map[string]any{{"id": "p1", "name": "Home Assistant", "language": "en"}},
					"preferred_pipeline": "p1",
				}
			case "assist_pipeline/language/list":
				result = map[string]any{"languages": []string{"en", "de"}}
			case "config/category_registry/list":
				result = []map[string]any{{"category_id": "c1", "name": "Lighting", "icon": nil}}
			case "energy/get_prefs":
				result = map[string]any{"energy_sources": []any{}, "device_consumption": []any{}}
			case "recorder/info":
				result = map[string]any{"recording": true, "backlog": 0}
			case "recorder/list_statistic_ids":
				result = []map[string]any{{"statistic_id": "sensor.energy", "has_sum": true}}
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
		{"helper-list", []string{"-o", "json", "helper", "input_boolean", "list"}, "Guest Mode"},
		{"integration-list", []string{"-o", "json", "integration", "list"}, "entry_id"},
		{"flow-handlers", []string{"-o", "json", "integration", "flow", "handlers"}, "random"},
		{"backup-list", []string{"-o", "json", "backup", "list"}, "idle"},
		{"system-hardware", []string{"-o", "json", "system", "hardware"}, "hardware"},
		{"system-analytics", []string{"-o", "json", "system", "analytics"}, "preferences"},
		{"system-labs", []string{"-o", "json", "system", "labs"}, "winter_mode"},
		{"lovelace-dashboards", []string{"-o", "json", "lovelace", "dashboard", "list"}, "ops-room"},
		{"assist-pipelines", []string{"-o", "json", "assist", "pipeline", "list"}, "preferred_pipeline"},
		{"assist-languages", []string{"-o", "json", "assist", "languages"}, "languages"},
		{"category-list", []string{"-o", "json", "registry", "category", "list", "--scope", "automation"}, "Lighting"},
		{"energy-prefs", []string{"-o", "json", "energy", "prefs", "get"}, "energy_sources"},
		{"stats-info", []string{"-o", "json", "statistics", "info"}, "recording"},
		{"stats-list", []string{"-o", "json", "statistics", "list"}, "statistic_id"},
		{"workflow-list", []string{"-o", "json", "workflow", "automation", "list"}, "automation.demo"},
		{"service-describe", []string{"-o", "json", "service", "describe", "light.turn_on"}, "brightness_pct"},
		{"system-health", []string{"-o", "json", "system", "health"}, "homeassistant"},
		{"system-repairs", []string{"-o", "json", "system", "repairs"}, "issues"},
		{"system-errorlog", []string{"system", "errorlog"}, "no errors"},
		{"addon-list", []string{"-o", "json", "addon", "list"}, "addons"},
		{"supervisor-info", []string{"-o", "json", "supervisor", "info"}, ""},
		{"skill-list-desc", []string{"skill", "list"}, "Home Assistant"},
		{"state-list-table", []string{"state", "list"}, "ENTITY"},
		{"state-list-yaml", []string{"-o", "yaml", "state", "list"}, "entity_id: light.kitchen"},
		{"state-list-ndjson", []string{"-o", "ndjson", "state", "list"}, "{\"entity_id\":\"sun.sun\""},
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

// TestValidation checks that write commands reject an empty --data locally.
func TestValidation(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	cases := [][]string{
		{"helper", "input_boolean", "update", "x"},
		{"integration", "update", "x"},
		{"registry", "area", "create"},
		{"system", "analytics", "set"},
		{"lovelace", "dashboard", "create"},
		{"assist", "pipeline", "create"},
		{"statistics", "period"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			root := cmd.NewRootCommand()
			root.SetArgs(args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			if err := root.Execute(); err == nil {
				t.Fatalf("expected error for %v, got nil", args)
			}
		})
	}
}

// TestNoSupervisor checks that add-on/supervisor commands fail fast with a clear
// message on a non-Supervised install (no "hassio" component).
func TestNoSupervisor(t *testing.T) {
	srv := mockHAWith(t, []string{"api", "frontend"})
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	for _, args := range [][]string{{"addon", "list"}, {"supervisor", "info"}} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			root := cmd.NewRootCommand()
			root.SetArgs(args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			err := root.Execute()
			if err == nil {
				t.Fatalf("expected error for %v on non-Supervisor install", args)
			}
			if !strings.Contains(err.Error(), "no Supervisor") {
				t.Fatalf("want 'no Supervisor' message, got: %v", err)
			}
		})
	}
}
