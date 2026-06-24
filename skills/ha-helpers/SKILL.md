---
name: ha-helpers
version: 0.1.0
description: "Create and manage Home Assistant helper entities with hass-cli: input_boolean, input_button, input_number, input_select, input_text, input_datetime, counter, timer, and schedule. Use to add a toggle/number/dropdown/text helper, a counter or timer, or a weekly schedule, and to list/rename/delete them. Creating a helper is a config operation (WebSocket), distinct from operating it (service call)."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli helper --help"
---

# ha-helpers

Manage helper entities — the user-defined inputs automations read and write.
**Prerequisite:** [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> Two different things, do not confuse them:
> - **Manage the helper** (create/rename/delete the entity itself) → `hass-cli helper ...` (this skill, WebSocket-only).
> - **Operate the helper** (turn on, set value, increment, start timer) → `hass-cli service call ...` (see ha-services).

## Helper types

`input_boolean`, `input_button`, `input_number`, `input_select`, `input_text`,
`input_datetime`, `counter`, `timer`, `schedule`.

## CRUD

`create` takes `--data` (must include `name`). `update`/`delete` take the helper
id positionally; the CLI injects it as `<type>_id`. Prefer `@file.json` for
`--data` to avoid shell quoting issues (especially PowerShell).

```bash
hass-cli helper input_boolean list

# Create (name is required; id is derived from the name by HA)
hass-cli helper input_boolean create --data '{"name":"Guest Mode","icon":"mdi:account"}'

# Rename / change fields (id is positional)
hass-cli helper input_boolean update guest_mode --data '{"name":"Guest Mode On"}'

# Delete
hass-cli helper input_boolean delete guest_mode
```

## Field hints per type

- `input_number`: `name`, `min`, `max`, `step`, `mode` (`box`|`slider`), `unit_of_measurement`, `initial`.
- `input_select`: `name`, `options` (array of strings), `initial`.
- `input_text`: `name`, `min`, `max`, `pattern`, `mode` (`text`|`password`).
- `input_datetime`: `name`, `has_date` (bool), `has_time` (bool).
- `counter`: `name`, `initial`, `minimum`, `maximum`, `step`, `restore`.
- `timer`: `name`, `duration` (e.g. `"00:05:00"`), `restore`.
- `schedule`: `name` plus weekday arrays (`monday`, ...) of `{from,to}` slots.
- `input_button` / `input_boolean`: `name`, `icon` (booleans also take `initial`).

When unsure of a field, create a minimal helper, then inspect it with
`hass-cli helper <type> list -o json`.

## After creating: operate it

A new helper immediately becomes an entity (`<type>.<id>`). Drive it via
services:

```bash
hass-cli service call input_boolean.turn_on --arguments entity_id=input_boolean.guest_mode
hass-cli service call counter.increment   --arguments entity_id=counter.visits
hass-cli service call input_number.set_value --data '{"entity_id":"input_number.temp","value":21}'
hass-cli service call timer.start --data '{"entity_id":"timer.tea","duration":"00:03:00"}'
```

Helpers are the natural state-holders for automations: an automation can read an
`input_boolean` as a condition and a `schedule` as a trigger.
