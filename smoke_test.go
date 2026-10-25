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

	"github.com/beclab/hass-cli/cmd"
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
						"target":      map[string]any{"entity": []any{map[string]any{"domain": []string{"light"}}}},
						"fields": map[string]any{
							"brightness_pct":    map[string]any{"description": "Brightness %"},
							"color_temp_kelvin": map[string]any{"description": "Colour temperature in kelvin"},
							"advanced_fields":   map[string]any{"collapsed": true, "fields": map[string]any{"color_name": map[string]any{"example": "red"}}},
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
				{"entity_id": "light.kitchen", "state": "off", "attributes": map[string]any{"friendly_name": "Kitchen Ceiling"}},
				{"entity_id": "light.living_room", "state": "on", "attributes": map[string]any{"friendly_name": "客厅灯"}},
				{"entity_id": "switch.kitchen_kettle", "state": "off", "attributes": map[string]any{"friendly_name": "Kettle"}},
				{"entity_id": "automation.demo", "state": "on"},
			})
		case strings.HasPrefix(r.URL.Path, "/api/states/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/states/")
			if id == "light.kitchen" {
				writeJSON(w, map[string]any{"entity_id": id, "state": "on", "attributes": map[string]any{"friendly_name": "Kitchen Ceiling", "brightness": 255, "supported_color_modes": []string{"color_temp", "xy"}}})
				return
			}
			writeJSON(w, map[string]any{"entity_id": id, "state": "on"})
		case r.URL.Path == "/api/services/light/turn_on":
			// Home Assistant answers with the states the call changed: none when the light
			// was already on.
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["entity_id"] == "light.kitchen" {
				writeJSON(w, []map[string]any{{"entity_id": "light.kitchen", "state": "on", "attributes": map[string]any{"friendly_name": "Kitchen Ceiling", "brightness": 255}}})
				return
			}
			writeJSON(w, []any{})
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
		{"state-list-domain", []string{"state", "list", "--domain", "switch"}, "switch.kitchen_kettle"},
		{"state-list-search-name", []string{"state", "list", "--search", "客厅"}, "light.living_room"},
		{"service-call-changed", []string{"service", "call", "light.turn_on", "--arguments", "entity_id=light.kitchen"}, "light.kitchen"},
		{"service-call-unchanged", []string{"service", "call", "light.turn_on", "--arguments", "entity_id=light.living_room"}, "No entity changed state"},
		{"state-list-yaml", []string{"-o", "yaml", "state", "list"}, "entity_id: light.kitchen"},
		{"state-list-ndjson", []string{"-o", "ndjson", "state", "list"}, "{\"entity_id\":\"sun.sun\""},
		{"service-call-entity_id-flag", []string{"service", "call", "light.turn_on", "--entity_id=light.kitchen"}, "light.kitchen"},
		{"service-call-entity-id-flag", []string{"service", "call", "light.turn_on", "--entity-id", "light.kitchen"}, "light.kitchen"},
		{"service-call-positional-entity", []string{"service", "call", "light.turn_on", "light.kitchen", "--arguments", "color_temp_kelvin=6500"}, "light.kitchen"},
		{"service-call-section-field", []string{"service", "call", "light.turn_on", "--entity", "light.kitchen", "--arguments", "color_name=white"}, "light.kitchen"},
		{"service-call-comma-pairs", []string{"service", "call", "light.turn_on", "--arguments", "entity_id=light.kitchen,brightness_pct=40"}, "light.kitchen"},
		{"service-describe-table", []string{"service", "describe", "light.turn_on"}, "brightness_pct"},
		{"service-describe-section-fields", []string{"service", "describe", "light.turn_on"}, "color_name"},
		{"service-list-domain", []string{"service", "list", "light"}, "light.turn_on"},
		{"state-get-table-attributes", []string{"state", "get", "light.kitchen"}, "attributes.supported_color_modes  [\"color_temp\",\"xy\"]"},
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

// TestProfileFlow exercises login → list → current → show → remove end to end
// against the mock, using a temp config dir (and data dir for the file-backed
// keychain). The trailing remove purges the keychain service it created.
func TestProfileFlow(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("HASS_CLI_DATA_DIR", t.TempDir())

	out := run(t, "profile", "login", "home", "--server", srv.URL, "--token", "test-token")
	if !strings.Contains(out, "logged in") || !strings.Contains(out, "Mock") {
		t.Fatalf("login output unexpected: %s", out)
	}
	if !strings.Contains(out, "Next steps") {
		t.Fatalf("login should print next steps: %s", out)
	}

	out = run(t, "profile", "list")
	if !strings.Contains(out, "home") || !strings.Contains(out, srv.URL) {
		t.Fatalf("list missing profile: %s", out)
	}

	out = run(t, "profile", "current")
	if !strings.Contains(out, "home") {
		t.Fatalf("current unexpected: %s", out)
	}

	out = run(t, "profile", "show", "home")
	if !strings.Contains(out, "server:") || strings.Contains(out, "test-token") {
		t.Fatalf("show should mask token: %s", out)
	}

	out = run(t, "profile", "remove", "home")
	if !strings.Contains(out, "removed profile") {
		t.Fatalf("remove unexpected: %s", out)
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

// TestStateFilters pins what --domain and --search leave out, which the substring
// checks above cannot show.
func TestStateFilters(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")
	ids := func(args ...string) []string {
		var states []map[string]any
		if err := json.Unmarshal([]byte(run(t, append([]string{"-o", "json", "state", "list"}, args...)...)), &states); err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, st := range states {
			out = append(out, st["entity_id"].(string))
		}
		return out
	}
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--domain", "light"}, "light.kitchen,light.living_room"},
		{[]string{"--domain", "light,switch", "--search", "kitchen"}, "light.kitchen,switch.kitchen_kettle"},
		{[]string{"--search", "KITCHEN ceiling"}, "light.kitchen"},
		{[]string{"--search", "客厅"}, "light.living_room"},
		{[]string{"--search", "garage"}, ""},
	}
	for _, tc := range cases {
		if got := strings.Join(ids(tc.args...), ","); got != tc.want {
			t.Errorf("state list %v = %q, want %q", tc.args, got, tc.want)
		}
	}
	// The table of a changed call is the compact one, not the whole state object.
	if out := run(t, "service", "call", "light.turn_on", "--arguments", "entity_id=light.kitchen"); strings.Contains(out, "brightness") {
		t.Errorf("service call table shows attributes: %s", out)
	}
}

// TestServiceCallEntityChecks checks that a call on a malformed or missing entity
// fails before reaching Home Assistant, naming candidates for a missing one.
func TestServiceCallEntityChecks(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	cases := map[string][]string{
		"did you mean light.kitchen (Kitchen Ceiling)":                   {"service", "call", "light.turn_on", "--arguments", "entity_id=light.kitchen_lamp"},
		"is not an entity id":                                            {"service", "call", "light.turn_on", "--data", `{"entity_id":"Kitchen light"}`},
		"light.turn_on needs a target: add --entity":                     {"service", "call", "light.turn_on"},
		"service call takes the target and data as":                      {"service", "call", "light.turn_on", "--brightness", "40"},
		"hass-cli service call light.turn_off":                           {"light.turn_off", "light.kitchen"},
		"did you mean state":                                             {"stat", "list"},
		`light.turn_on has no field "color_temp"; use color_temp_kelvin`: {"service", "call", "light.turn_on", "--entity", "light.kitchen", "--arguments", "color_temp=6500"},
		"e.g. --arguments color_temp_kelvin=2500":                        {"service", "call", "light.turn_on", "--data", `{"entity_id":"light.kitchen","color_temp":400}`},
		`light.turn_on has no field "colour"; use color_name`:            {"service", "call", "light.turn_on", "--entity", "light.kitchen", "--arguments", "colour=white"},
		"is data, not an entity: pass it as --arguments color_temp=6500": {"service", "call", "light.turn_on", "light.kitchen", "color_temp=6500"},
		`unexpected argument "kitchen"`:                                  {"service", "call", "light.turn_on", "kitchen"},
		"entity light.kitchen_lamp not found":                            {"service", "call", "light.turn_on", "light.kitchen_lamp"},
	}
	for want, args := range cases {
		t.Run(want, func(t *testing.T) {
			root := cmd.NewRootCommand()
			root.SetArgs(args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want error containing %q, got %v", want, err)
			}
		})
	}
}
