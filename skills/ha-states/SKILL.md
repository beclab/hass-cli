---
name: ha-states
version: 0.1.0
description: "Read Home Assistant entity states and recent history with hass-cli. Use to answer 'is the light on', 'what's the temperature', 'list all sensors', 'what changed recently', or to inspect any entity's current state and attributes. Covers state list/get, table shaping (--columns/--sort-by), and history/logbook reads."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli state --help"
---

# ha-states

Read entity state. **Prerequisite:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) for connection/auth.

## Commands

```bash
hass-cli state list                                  # every entity
hass-cli state list -o json | jq '.[] | select(.entity_id|startswith("light."))'
hass-cli state get sensor.kitchen_temperature
hass-cli state get light.kitchen -o yaml             # full attributes
```

Shape tables. `state list` already defaults to
`ENTITY=entity_id,STATE=state,NAME=attributes.friendly_name` in table mode;
pass `--columns` to override:

```bash
hass-cli state list --columns ENTITY=entity_id,STATE=state --sort-by entity_id
```

## History and logbook

```bash
hass-cli system history --start 2026-01-01T00:00:00+00:00 --entities light.kitchen,sensor.temp
hass-cli system logbook --entity automation.demo
```

## Notes

- `state set` overwrites the state object in the state machine; it does NOT
  drive a device. To actually change a device, use `service call` (see
  [`../ha-services/SKILL.md`](../ha-services/SKILL.md)).
- Filtering/aggregation is client-side: pipe `-o json` to `jq`.
