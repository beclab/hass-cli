package cmd

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/beclab/hass-cli/internal/client"
)

func TestSplitArguments(t *testing.T) {
	cases := map[string][]string{
		"entity_id=light.kitchen":                                {"entity_id=light.kitchen"},
		"entity_id=light.kitchen,brightness_pct=40":              {"entity_id=light.kitchen", "brightness_pct=40"},
		"entity_id=light.aqara,color_temp=400":                   {"entity_id=light.aqara", "color_temp=400"},
		"entity_id=light.a,brightness=255,rgb_color=[255,220,0]": {"entity_id=light.a", "brightness=255", "rgb_color=[255,220,0]"},
		"entity_id=light.a,light.b":                              {"entity_id=light.a,light.b"},
		`message=hello, world,title=x`:                           {"message=hello, world", "title=x"},
		`data={"a":1,"b":2},entity_id=light.a`:                   {`data={"a":1,"b":2}`, "entity_id=light.a"},
		`message="a,b=c",title=x`:                                {`message="a,b=c"`, "title=x"},
	}
	for in, want := range cases {
		if got := splitArguments(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitArguments(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEntityIDsIn(t *testing.T) {
	got := entityIDsIn(map[string]any{"entity_id": "light.a, light.b", "target": map[string]any{"entity_id": []any{"switch.c"}}})
	if want := []string{"light.a", "light.b", "switch.c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entityIDsIn = %q, want %q", got, want)
	}
}

func TestExplainCallError(t *testing.T) {
	err := explainCallError(&client.HTTPError{Status: 400, Body: "400: Bad Request"}, "light.turn_on")
	if !strings.Contains(err.Error(), "service describe light.turn_on") || !strings.Contains(err.Error(), "--data") {
		t.Fatalf("400 not explained: %v", err)
	}
	var httpErr *client.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatal("the HTTP error is no longer wrapped")
	}
	other := errors.New("boom")
	if explainCallError(other, "light.turn_on") != other {
		t.Fatal("non-400 errors must pass through")
	}
}
