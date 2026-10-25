---
name: ha-services
version: 0.3.0
description: "List, describe and call any Home Assistant service, and fire events, when no domain skill covers the request."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service --help"
---

# ha-services

Drive devices by calling services. Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> Lights, covers, climate and media have their usual calls in their domain
> skills; reach for this one for any other service and to discover a service's
> fields. There are no per-domain commands (no `hass-cli light ...`): everything
> goes through `service call` against the service schema.

## Discover

```bash
hass-cli service list                       # all domains/services
hass-cli service describe light.turn_on     # fields for one service
```

## Call

```bash
# Simple key=value arguments (values are coerced to bool/number/json/string)
hass-cli service call light.turn_on --arguments entity_id=light.kitchen --arguments brightness_pct=60

# Full JSON payload (use for nested data / targets)
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","temperature":21.5}'

# Target by area/device/entity
hass-cli service call light.turn_off --data '{"target":{"area_id":"kitchen"}}'
```

## Patterns

- Common calls (`turn_on`/`turn_off`/`toggle`, and those the domain skills show)
  need no lookup: call them directly. Use `service describe <domain.service>`
  for an unfamiliar service or when a call rejects a field.
- The call prints the entities it changed, which confirms it; `No entity changed
  state` means the target already was that way. Use `-o json` for the raw body.
- `entity_id`, `area_id`, `device_id`, `label_id` are the standard targets.
- Toggle/turn_on/turn_off live under each domain and under `homeassistant.*`
  (generic). When unsure of the domain, `homeassistant.turn_on` often works.
