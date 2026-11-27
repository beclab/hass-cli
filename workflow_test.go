package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beclab/hass-cli/cmd"
)

// runAll executes the root command and returns stdout, stderr and the error.
func runAll(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	old := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	var stderr bytes.Buffer
	root := cmd.NewRootCommand()
	root.SetArgs(args)
	root.SetErr(&stderr)
	err := root.Execute()
	_ = wp.Close()
	os.Stdout = old
	var stdout bytes.Buffer
	_, _ = stdout.ReadFrom(rp)
	return stdout.String(), stderr.String(), err
}

func requested(prefix string) []string {
	var out []string
	for _, r := range haRequests.all() {
		if strings.HasPrefix(r, prefix) {
			out = append(out, r)
		}
	}
	return out
}

// TestWorkflowRunAndStop pins the services each verb calls: Home Assistant has
// no script.trigger, and stopping an automation's run must leave it enabled.
func TestWorkflowRunAndStop(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"workflow", "script", "trigger", "script.color_cycle"}, []string{"POST /api/services/script/turn_on"}},
		{[]string{"workflow", "automation", "trigger", "automation.demo"}, []string{"POST /api/services/automation/trigger"}},
		{[]string{"workflow", "scene", "trigger", "scene.movie"}, []string{"POST /api/services/scene/turn_on"}},
		{[]string{"workflow", "script", "stop", "script.color_cycle"}, []string{"POST /api/services/script/turn_off"}},
		{[]string{"workflow", "automation", "stop", "automation.demo"}, []string{"POST /api/services/automation/turn_off", "POST /api/services/automation/turn_on"}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			haRequests.reset()
			if _, stderr, err := runAll(t, tc.args...); err != nil {
				t.Fatalf("%v: %v\n%s", tc.args, err, stderr)
			}
			got := requested("POST /api/services/")
			if len(got) != len(tc.want) {
				t.Fatalf("calls = %q, want %q", got, tc.want)
			}
			for i, want := range tc.want {
				if !strings.HasPrefix(got[i], want) {
					t.Fatalf("call %d = %q, want %q", i, got[i], want)
				}
			}
		})
	}
	haRequests.reset()
	if _, _, err := runAll(t, "workflow", "automation", "stop", "automation.demo"); err != nil {
		t.Fatal(err)
	}
	if off := requested("POST /api/services/automation/turn_off"); len(off) != 1 || !strings.Contains(off[0], `"stop_actions":true`) {
		t.Fatalf("automation stop must stop the running actions: %q", off)
	}
	if _, _, err := runAll(t, "workflow", "scene", "stop", "scene.movie"); err == nil {
		t.Fatal("a scene has no run to stop")
	}
}

// TestWorkflowSave checks that save takes inline data, refuses an entity id
// where Home Assistant wants the bare id, and names the entity it made.
func TestWorkflowSave(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")
	body := `{"alias":"Colour cycle","sequence":[{"delay":{"seconds":5}}]}`

	haRequests.reset()
	stdout, stderr, err := runAll(t, "workflow", "script", "save", "color_cycle", "--data", body)
	if err != nil {
		t.Fatalf("save: %v\n%s", err, stderr)
	}
	if got := requested("POST /api/config/script/config/color_cycle"); len(got) != 1 || !strings.Contains(got[0], `"sequence"`) {
		t.Fatalf("save sent %q", haRequests.all())
	}
	if !strings.Contains(stdout, "script.color_cycle") {
		t.Fatalf("save should name the entity it created: %s", stdout)
	}

	file := filepath.Join(t.TempDir(), "s.yaml")
	if err := os.WriteFile(file, []byte("alias: x\nsequence: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := runAll(t, "workflow", "script", "save", "x", "--data", "@"+file); err != nil {
		t.Fatalf("save from a YAML file: %v\n%s", err, stderr)
	}

	refusals := map[string][]string{
		"save takes the bare id: color_cycle (the entity becomes script.color_cycle)": {"workflow", "script", "save", "script.color_cycle", "--data", body},
		`"Colour Cycle" is not a script id; use colour_cycle`:                         {"workflow", "script", "save", "Colour Cycle", "--data", body},
		"--data is required": {"workflow", "script", "save", "color_cycle"},
	}
	for want, args := range refusals {
		haRequests.reset()
		_, _, err := runAll(t, args...)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%v: want error containing %q, got %v", args, want, err)
		}
		if got := requested("POST /api/config/"); len(got) != 0 {
			t.Fatalf("%v reached Home Assistant: %q", args, got)
		}
	}
}

// TestServiceCallNamesARunningScript checks that turning a device off says when
// a running script or automation drives it, since that run will undo the call.
func TestServiceCallNamesARunningScript(t *testing.T) {
	srv := mockHA(t)
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	_, stderr, err := runAll(t, "service", "call", "light.turn_off", "--entity", "light.kitchen")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"script.color_cycle", "hass-cli workflow script stop script.color_cycle"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "automation.demo") {
		t.Fatalf("automation.demo drives the kettle, not the light:\n%s", stderr)
	}

	_, stderr, err = runAll(t, "service", "call", "switch.turn_off", "--entity", "switch.kitchen_kettle")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "hass-cli workflow automation stop automation.demo") {
		t.Fatalf("stderr should name the running automation:\n%s", stderr)
	}

	_, stderr, err = runAll(t, "service", "call", "light.turn_on", "--entity", "light.living_room")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("nothing drives the living room light, yet: %s", stderr)
	}
}

// A service that returns data is refused by Home Assistant unless the call
// asks for the response, and the response, not the changed states, is the
// answer.
func TestServiceCallReturnsResponse(t *testing.T) {
	srv := mockHA(t)
	defer srv.Close()
	t.Setenv("HASS_SERVER", srv.URL)
	t.Setenv("HASS_TOKEN", "test-token")

	stdout, _, err := runAll(t, "service", "call", "weather.get_forecasts", "--entity", "weather.home", "--data", `{"type":"daily"}`)
	if err != nil {
		t.Fatalf("a service that requires a response failed: %v", err)
	}
	if !strings.Contains(stdout, "sunny") || !strings.Contains(stdout, "weather.home") {
		t.Fatalf("the forecast is not printed: %q", stdout)
	}

	_, _, err = runAll(t, "service", "call", "light.turn_on", "--entity", "light.kitchen", "--response")
	if err == nil || !strings.Contains(err.Error(), "returns no data") {
		t.Fatalf("--response on a service without data should be refused, got %v", err)
	}
}
