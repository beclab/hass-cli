---
name: ha-states
version: 0.3.0
description: "Read entity states and recent history: 'is the light on', 'what is the temperature', list the entities of a domain."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli state --help"
---

# ha-states

Read entity state. Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Commands

```bash
hass-cli state list                                  # every entity
hass-cli state list --domain light                   # one domain (or light,switch)
hass-cli state list --search kitchen                 # entity_id or friendly name, every word
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
