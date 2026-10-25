package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/beclab/hass-cli/internal/catalog"
	"github.com/beclab/hass-cli/internal/client"
	"github.com/spf13/cobra"
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

// fieldRenames maps field names agents guess to the field Home Assistant takes.
// color_temp (mireds) and kelvin were removed from light.turn_on; a call that
// still sends them gets a bare "400: Bad Request".
var fieldRenames = map[string]string{
	"color_temp":         "color_temp_kelvin",
	"colour_temp":        "color_temp_kelvin",
	"kelvin":             "color_temp_kelvin",
	"color_temperature":  "color_temp_kelvin",
	"brightness_percent": "brightness_pct",
	"color":              "color_name",
	"colour":             "color_name",
	"colour_name":        "color_name",
}

// checkService fails before the call when the data leaves out a target the
// service needs, or names a field the service does not have: Home Assistant
// answers both with a bare "400: Bad Request". The service's targets and fields
// come from the live service catalog; a service that lists no fields is not
// checked for them, since scripts take any variables.
func checkService(ctx context.Context, c *client.Client, domain, service string, data map[string]any) error {
	raw, err := c.Services(ctx)
	if err != nil {
		return nil
	}
	cat, err := catalog.Parse(raw)
	if err != nil {
		return nil
	}
	svc, ok := cat.Lookup(domain, service)
	if !ok {
		return nil
	}
	if err := checkTarget(domain, service, svc, data); err != nil {
		return err
	}
	return checkFields(domain+"."+service, svc, data)
}

// checkTarget fails when the service acts on targets and the data names none.
func checkTarget(domain, service string, svc catalog.Service, data map[string]any) error {
	for _, key := range targetKeys {
		if _, ok := data[key]; ok {
			return nil
		}
	}
	if len(svc.Target) == 0 || string(svc.Target) == "null" {
		return nil
	}
	return fmt.Errorf("%s.%s needs a target: add --entity %s.<object_id> (find it with: hass-cli state list --domain %s --search <name>)", domain, service, domain, domain)
}

// checkFields fails on the first data key that is neither a target nor one of
// the service's fields, naming the field meant where it can tell.
func checkFields(name string, svc catalog.Service, data map[string]any) error {
	fields := fieldNames(svc.Fields)
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if fields[key] || slices.Contains(targetKeys, key) {
			continue
		}
		msg := fmt.Sprintf("%s has no field %q", name, key)
		if meant := meantField(key, fields); meant != "" {
			msg += "; use " + meant
			if meant == "color_temp_kelvin" {
				msg += " (kelvin: about 2700 warm white, 4000 neutral, 6500 cool white)"
				if kelvin, ok := mireds(data[key]); ok {
					msg += fmt.Sprintf(", e.g. --arguments color_temp_kelvin=%d", kelvin)
				}
			}
		}
		return errors.New(msg + fmt.Sprintf(". Fields: hass-cli service describe %s", name))
	}
	return nil
}

// fieldNames lists a service's fields, a section's fields in its place.
func fieldNames(fields map[string]catalog.Field) map[string]bool {
	names := map[string]bool{}
	for name, field := range fields {
		if len(field.Fields) > 0 {
			for inner := range fieldNames(field.Fields) {
				names[inner] = true
			}
			continue
		}
		names[name] = true
	}
	return names
}

// meantField is the field a wrong name stands for: a known rename, the same
// name with dashes, or the one field that contains it or is contained in it.
func meantField(key string, fields map[string]bool) string {
	if renamed, ok := fieldRenames[key]; ok && fields[renamed] {
		return renamed
	}
	if underscored := strings.ReplaceAll(key, "-", "_"); fields[underscored] {
		return underscored
	}
	var near []string
	for field := range fields {
		if strings.Contains(field, key) || strings.Contains(key, field) {
			near = append(near, field)
		}
	}
	if len(near) == 1 {
		return near[0]
	}
	return ""
}

// mireds reads an old color_temp value as kelvin: values below 1000 are mireds.
func mireds(v any) (int, bool) {
	var n float64
	switch x := v.(type) {
	case int64:
		n = float64(x)
	case float64:
		n = x
	default:
		return 0, false
	}
	switch {
	case n >= 1000:
		return int(n), true
	case n > 0:
		return int(1e6/n/100+0.5) * 100, true
	}
	return 0, false
}

// unknownFlagHint answers an unknown --name flag on service call with the
// --arguments form of the field it most likely means.
func unknownFlagHint(err error, args []string) string {
	flag, ok := strings.CutPrefix(err.Error(), "unknown flag: --")
	if !ok {
		return ""
	}
	field := strings.ReplaceAll(flag, "-", "_")
	if renamed, ok := fieldRenames[field]; ok {
		field = renamed
	}
	value := "<value>"
	for i, arg := range args {
		if v, ok := strings.CutPrefix(arg, "--"+flag+"="); ok {
			value = v
		} else if arg == "--"+flag && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			value = args[i+1]
		}
	}
	if field == "color_temp_kelvin" {
		if kelvin, ok := mireds(coerceValue(value)); ok {
			value = fmt.Sprint(kelvin)
		}
	}
	return fmt.Sprintf("\n--%s is not a flag; give a service field with --arguments: --arguments %s=%s", flag, field, value)
}

// callArgs takes the service, then any entity ids the call should act on.
func callArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("service call needs <domain.service>, e.g. hass-cli service call light.turn_on --entity light.kitchen")
	}
	for _, arg := range args[1:] {
		if strings.Contains(arg, "=") {
			return fmt.Errorf("%q is data, not an entity: pass it as --arguments %s", arg, arg)
		}
		if !entityIDPattern.MatchString(arg) {
			return fmt.Errorf("unexpected argument %q: service call takes <domain.service>, then entity ids (domain.object_id)", arg)
		}
	}
	return nil
}
