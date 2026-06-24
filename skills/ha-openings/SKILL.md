---
name: ha-openings
version: 0.1.0
description: "Control Home Assistant covers, locks, valves, garage doors, and gates with hass-cli: open/close/stop covers, set position and tilt, lock/unlock, open/close valves. Use for 'open the blinds', 'close the garage', 'set shades to 50%', 'lock the front door' requests. All control goes through service call."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-openings

Covers (blinds/shades/garage/gate), locks, valves. **Prerequisites:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md), [`../ha-services/SKILL.md`](../ha-services/SKILL.md).

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
