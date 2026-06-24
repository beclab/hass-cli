---
name: ha-services
version: 0.1.0
description: "Call Home Assistant services to control devices and trigger actions with hass-cli. Use to turn lights on/off, set climate, open covers, play media, run any domain.service. Covers discovering services (service list/describe) and calling them (service call) with --arguments and --data, including targeting by entity/device/area."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service --help"
---

# ha-services

Drive devices by calling services. **Prerequisite:** [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> This is the workhorse for ALL device control. There are no per-domain
> commands (no `hass-cli light ...`); everything goes through `service call`
> against the service schema.

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

- Find the right service first with `service describe <domain.service>` to learn
  required fields and selectors.
- `entity_id`, `area_id`, `device_id`, `label_id` are the standard targets.
- Toggle/turn_on/turn_off live under each domain and under `homeassistant.*`
  (generic). When unsure of the domain, `homeassistant.turn_on` often works.
