package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/beclab/hass-cli/internal/catalog"
	"github.com/beclab/hass-cli/internal/client"
)

// splitArguments turns one --arguments value into its key=value pairs. Agents
// often write `entity_id=light.kitchen,brightness_pct=40` in one flag; read as a
// single pair, the whole tail became the entity id and Home Assistant answered a
// bare "400: Bad Request". A comma splits only where the next segment starts a
// new `key=`, outside brackets and quotes, so `entity_id=light.a,light.b` and
// `rgb_color=[255,200,0]` keep their commas.
func splitArguments(arg string) []string {
	var parts []string
	depth, quote, start := 0, rune(0), 0
	runes := []rune(arg)
	for i, r := range runes {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '[' || r == '{':
			depth++
		case r == ']' || r == '}':
			if depth > 0 {
				depth--
			}
		case r == ',' && depth == 0 && startsKey(string(runes[i+1:])):
			parts = append(parts, string(runes[start:i]))
			start = i + 1
		}
	}
	return append(parts, string(runes[start:]))
}

var argumentKey = regexp.MustCompile(`^\s*[A-Za-z_][A-Za-z0-9_]*=`)

func startsKey(rest string) bool { return argumentKey.MatchString(rest) }

var entityIDPattern = regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`)

// entityIDsIn collects the entity ids a service call targets, from entity_id at
// the top level or under target, as a string (comma-separated allowed) or a list.
func entityIDsIn(data map[string]any) []string {
	var ids []string
	add := func(v any) {
		switch x := v.(type) {
		case string:
			for _, id := range strings.Split(x, ",") {
				if id = strings.TrimSpace(id); id != "" {
					ids = append(ids, id)
				}
			}
		case []any:
			for _, item := range x {
				if s, ok := item.(string); ok {
					ids = append(ids, strings.TrimSpace(s))
				}
			}
		}
	}
	add(data["entity_id"])
	if target, ok := data["target"].(map[string]any); ok {
		add(target["entity_id"])
	}
	return ids
}

// checkEntities fails before the call when an entity id is malformed or does not
// exist. Home Assistant accepts a call on a missing entity and changes nothing,
// which reads like success ("No entity changed state"), and rejects a malformed
// one with no reason at all. "all" is Home Assistant's own every-entity target.
func checkEntities(ctx context.Context, c *client.Client, ids []string) error {
	var check []string
	for _, id := range ids {
		if id == "all" {
			continue
		}
		if !entityIDPattern.MatchString(id) {
			return fmt.Errorf("%q is not an entity id (want domain.object_id, e.g. light.kitchen). Give each key its own --arguments, or pass lists and objects with --data '{...}'", id)
		}
		check = append(check, id)
	}
	if len(check) == 0 {
		return nil
	}
	raw, err := c.States(ctx)
	if err != nil {
		return nil // the call itself will report a connection problem
	}
	var states []struct {
		EntityID   string         `json:"entity_id"`
		Attributes map[string]any `json:"attributes"`
	}
	if json.Unmarshal(raw, &states) != nil {
		return nil
	}
	known := make(map[string]bool, len(states))
	for _, st := range states {
		known[st.EntityID] = true
	}
	for _, id := range check {
		if known[id] {
			continue
		}
		msg := fmt.Sprintf("entity %s not found", id)
		if near := similarEntities(id, states); len(near) > 0 {
			msg += "; did you mean " + strings.Join(near, ", ") + "?"
		} else {
			msg += fmt.Sprintf("; list candidates with: hass-cli state list --domain %s", strings.SplitN(id, ".", 2)[0])
		}
		return errors.New(msg)
	}
	return nil
}

// similarEntities ranks entities of the same domain by words shared with the
// requested object id or friendly name, best three first.
func similarEntities(id string, states []struct {
	EntityID   string         `json:"entity_id"`
	Attributes map[string]any `json:"attributes"`
}) []string {
	domain, object, _ := strings.Cut(id, ".")
	words := strings.FieldsFunc(object, func(r rune) bool { return r == '_' })
	type scored struct {
		label string
		score int
	}
	var hits []scored
	for _, st := range states {
		d, o, _ := strings.Cut(st.EntityID, ".")
		if d != domain {
			continue
		}
		name, _ := st.Attributes["friendly_name"].(string)
		hay := strings.ToLower(o + " " + name)
		score := 0
		for _, w := range words {
			if w != "" && strings.Contains(hay, strings.ToLower(w)) {
				score++
			}
		}
		label := st.EntityID
		if name != "" {
			label += " (" + name + ")"
		}
		hits = append(hits, scored{label, score})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	var out []string
	for i := 0; i < len(hits) && i < 3; i++ {
		out = append(out, hits[i].label)
	}
	return out
}

// explainCallError adds what Home Assistant leaves out: a 400 on a service call
// comes back as "400: Bad Request" with no field named.
func explainCallError(err error, service string) error {
	var httpErr *client.HTTPError
	if errors.As(err, &httpErr) && httpErr.Status == 400 {
		return fmt.Errorf("%w\nHome Assistant rejected the service data. Check the field names and values with `hass-cli service describe %s`; colours, lists and objects go in --data, e.g. --data '{\"entity_id\":\"light.kitchen\",\"rgb_color\":[255,200,0]}'", err, service)
	}
	return err
}

// targetKeys are the ways a service call can name what it acts on.
var targetKeys = []string{"entity_id", "device_id", "area_id", "floor_id", "label_id", "target"}

// checkTarget fails before the call when the service acts on targets and the
// data names none: Home Assistant answers that with a bare "400: Bad Request".
// Whether a service takes a target comes from the live service catalog.
func checkTarget(ctx context.Context, c *client.Client, domain, service string, data map[string]any) error {
	for _, key := range targetKeys {
		if _, ok := data[key]; ok {
			return nil
		}
	}
	raw, err := c.Services(ctx)
	if err != nil {
		return nil
	}
	cat, err := catalog.Parse(raw)
	if err != nil {
		return nil
	}
	svc, ok := cat.Lookup(domain, service)
	if !ok || len(svc.Target) == 0 || string(svc.Target) == "null" {
		return nil
	}
	return fmt.Errorf("%s.%s needs a target: add --entity %s.<object_id> (find it with: hass-cli state list --domain %s --search <name>)", domain, service, domain, domain)
}
