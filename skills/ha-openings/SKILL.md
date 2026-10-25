---
name: ha-openings
version: 0.2.0
description: "Open, close or position covers, blinds, garage doors and gates; lock and unlock locks; open and close valves."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-openings

Covers (blinds/shades/garage/gate), locks, valves. Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Quick control (most requests)

Two commands turn an existing device on or off or set it. Do not load other
skills, run `service describe`, or check the state afterwards.

```bash
# 1. Find it by name: a few rows, not the whole house
hass-cli state list --domain cover --search bedroom

# 2. Call the service on the entity_id from step 1
hass-cli service call cover.close_cover --arguments entity_id=cover.bedroom
hass-cli service call cover.open_cover --arguments entity_id=cover.bedroom
hass-cli service call lock.lock --arguments entity_id=lock.front_door
```

The call prints the entities it changed: that is the confirmation. `No entity
changed state` means it was already that way. No row: search one word, another
language, or the domain alone. Several equally likely rows: ask which one.

## Covers (`cover.*`)

```bash
hass-cli service call cover.open_cover  --arguments entity_id=cover.living_blinds
hass-cli service call cover.close_cover --arguments entity_id=cover.living_blinds
hass-cli service call cover.stop_cover  --arguments entity_id=cover.living_blinds

# Position 0 (closed) .. 100 (open)
hass-cli service call cover.set_cover_position --data '{"entity_id":"cover.living_blinds","position":50}'

# Tilt (venetian blinds)
hass-cli service call cover.set_cover_tilt_position --data '{"entity_id":"cover.living_blinds","tilt_position":30}'
```

Check support with `hass-cli state get cover.x -o json` →
`attributes.supported_features` / `current_position`.

## Locks (`lock.*`)

```bash
hass-cli service call lock.lock   --arguments entity_id=lock.front_door
hass-cli service call lock.unlock --arguments entity_id=lock.front_door
# Some locks require a code:
hass-cli service call lock.unlock --data '{"entity_id":"lock.front_door","code":"1234"}'
```

## Valves (`valve.*`)

```bash
hass-cli service call valve.open_valve  --arguments entity_id=valve.water_main
hass-cli service call valve.close_valve --arguments entity_id=valve.water_main
hass-cli service call valve.set_valve_position --data '{"entity_id":"valve.water_main","position":25}'
```

Garage doors and gates are usually `cover.*` entities with device classes
`garage`/`gate`; use the cover services above.
