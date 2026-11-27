package cmd

import (
	"context"
	"strings"

	"github.com/beclab/hass-cli/internal/client"
	"github.com/tidwall/gjson"
)

// workflowDomains run other entities; a call on one of them is not a device
// that something else could be driving.
var workflowDomains = map[string]bool{"script": true, "automation": true, "scene": true}

type driver struct {
	domain, entity, target string
}

// runningDrivers lists the scripts and automations running right now whose
// config names one of targets. A light turned off while a colour-cycle script
// runs is turned back on by the script's next step, and the call's own answer
// ("off") cannot show that. Lookup failures yield nothing: the call itself has
// already succeeded.
func runningDrivers(ctx context.Context, c *client.Client, targets []string) []driver {
	if len(targets) == 0 {
		return nil
	}
	states, err := c.States(ctx)
	if err != nil {
		return nil
	}
	var out []driver
	gjson.ParseBytes(states).ForEach(func(_, st gjson.Result) bool {
		entity := st.Get("entity_id").String()
		domain, object, _ := strings.Cut(entity, ".")
		var id string
		switch {
		case domain == "script" && st.Get("state").String() == "on":
			id = object
		case domain == "automation" && st.Get("attributes.current").Int() > 0:
			id = st.Get("attributes.id").String()
		}
		if id == "" {
			return true
		}
		config, err := c.REST(ctx, "GET", configPath(domain, id), nil)
		if err != nil {
			return true
		}
		for _, target := range targets {
			if strings.Contains(string(config), `"`+target+`"`) {
				out = append(out, driver{domain: domain, entity: entity, target: target})
				break
			}
		}
		return true
	})
	return out
}
