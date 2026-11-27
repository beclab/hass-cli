---
name: ha-openings
version: "0.0.0"
description: "Curtains, blinds, shades, shutters, garage doors and gates: open, close, stop, half open, tilt; door locks: lock, unlock, unlatch; water and gas valves. 窗帘、百叶窗、车库门、门锁、阀门."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-openings

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Covers (`cover.*`: curtains, blinds, shades, shutters, awnings, garage doors, gates), locks (`lock.*`) and valves (`valve.*`). Whether a door or window is open, read from a contact sensor, is a `binary_sensor`: [`ha-entities`](../ha-entities/SKILL.md). An alarm panel is there too.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Open, close or stop the curtains, blinds, garage door | this file | `cover.open_cover`, `close_cover`, `stop_cover` |
| Halfway, "open it 30%" | this file | `cover.set_cover_position` with `position` |
| Tilt the slats | this file | `cover.set_cover_tilt_position` or `open_cover_tilt` / `close_cover_tilt` |
| Lock or unlock a door | this file | `lock.lock` or `lock.unlock` |
| Shut off the water | this file | `valve.close_valve` |
| Is the garage open, is the door locked | this file | `state get`; the state is the answer |

## Covers

State is `open`, `closed`, `opening` or `closing`. `attributes.current_position` is 0 (closed) to 100 (open); `current_tilt_position` the same for slats. `attributes.device_class` says what it is: `curtain`, `blind`, `shade`, `shutter`, `awning`, `garage`, `gate`, `door`, `window`.

```bash
hass-cli service call cover.set_cover_position --data '{"entity_id":"cover.living_curtain","position":50}'
hass-cli service call cover.set_cover_tilt_position --data '{"entity_id":"cover.office_blind","tilt_position":30}'
hass-cli service call cover.stop_cover --entity cover.living_curtain
```

- The call returns while the cover is still moving: the printed state can be `opening`, which is success.
- Position is how open, not how closed. "Close it 70%" is `position: 30`.
- Not every cover takes a position or a tilt. A cover with no `current_position` attribute only opens, closes and stops: for "halfway" say it cannot, and offer open or closed. Same for tilt without `current_tilt_position`.
- A garage door or gate often reports only `open` / `closed` and may need a few seconds; it may also stay `opening` until its sensor reports.
- "Open all the curtains" in one room: one call with `area_id` (ids from [`ha-registry`](../ha-registry/SKILL.md)) or a list of `entity_id`s; check that the area does not also hold the garage door.

## Locks

State is `locked`, `unlocked`, `locking`, `unlocking`, `jammed` or `open`.

```bash
hass-cli service call lock.lock --entity lock.front_door
hass-cli service call lock.unlock --data '{"entity_id":"lock.front_door","code":"<from the user>"}'
```

- `lock.open` unlatches (pulls the latch so the door swings open) on locks that support it; plain "unlock" is `lock.unlock`.
- When `attributes.code_format` is present the lock needs a code. Ask the user for it each time; never guess one, reuse one from an earlier request, or write it into a script.
- `jammed` means the bolt did not finish. Say so; do not retry in a loop.

## Valves

`valve.open_valve`, `valve.close_valve`, `valve.stop_valve`, and `valve.set_valve_position` (0 closed to 100 open) when `current_position` is reported. Many water shut-offs are `switch.*` instead: if no `valve.*` matches, search without `--domain`, and read the name before switching it.

## Verb index

| Service | For | Read when triggered |
|---|---|---|
| `cover.open_cover`, `close_cover`, `stop_cover`, `toggle` | whole movement | [covers](#covers) |
| `cover.set_cover_position` | a percentage open | only when `current_position` exists |
| `cover.open_cover_tilt`, `close_cover_tilt`, `set_cover_tilt_position`, `stop_cover_tilt` | slats | only when `current_tilt_position` exists |
| `lock.lock`, `lock.unlock`, `lock.open` | locks; `code` when `code_format` is set | [locks](#locks) and the rules below |
| `valve.*` | `open_valve`, `close_valve`, `stop_valve`, `set_valve_position` | [valves](#valves) |
| Anything else on these domains | integration-specific services | `hass-cli service list cover lock valve` shows what this instance has; `service describe` gives the fields |

## Safety and escalation

- Unlocking a door, opening a garage door or gate, and closing a main water or gas valve happen only on the user's request for that device, named clearly. Never as a step of another request ("I'm home" is not "unlock the door"), and never on a guess between two matches: ask.
- Say what changed in words the user can check: "the front door is unlocked", "the garage door is opening".
- A lock code is the user's secret: do not repeat it back, log it, or put it in an automation. Ask each time.
- Closing a garage door or gate with nobody watching can trap a person, a pet or a car: when the request came from away from home (a scheduled task, an automation), say that it is closing.
- An automation that unlocks or opens anything is built only after the user confirms it in words: [the builder](../ha-workflow-automation-builder/SKILL.md).
