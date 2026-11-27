---
name: ha-states
version: "0.0.0"
description: "Read what something is doing now or did earlier: 'is the light on', 'what is the temperature', 'is the door locked', 'how many lights are on', 'when did it change', what happened today; history, logbook and live state changes."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-states

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Answering questions about state, now and in the past. Changing a device is a service call, never a state write. Long-term trends (energy per month, average temperature last week) are [`ha-statistics`](../ha-statistics/SKILL.md); who is home is [`ha-presence`](../ha-presence/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Is the light on", "what is the temperature" | this file | `state list --search <name>`, the STATE column |
| One device's details: brightness, colour, battery, unit | this file | `state get <entity_id>` |
| "How many lights are on", "which windows are open" | this file | `state list --domain ...`, or one `template` |
| "When did the door last open", "what happened this morning" | this file | `system logbook` or `system history` |
| Watch changes live while something is tested | this file | `event watch state_changed`, bounded |

## Reading now

`state list` with `--search` and `--domain` answers most questions in one call: the table shows entity, state and name. `state get` shows one entity as key/value rows: state, `last_changed`, then every attribute.

| State | Means |
|---|---|
| `on` / `off`, `open` / `closed`, `locked` / `unlocked`, `home` / `not_home` | what it says; a `binary_sensor` door or window is `on` when open |
| a number | a sensor reading; the unit is `attributes.unit_of_measurement` |
| `unavailable` | Home Assistant cannot reach the device right now; say so, do not report a value |
| `unknown` | reachable but no value yet (just restarted, never reported) |

- `last_changed` is when the state value last changed; an attribute-only update (a new brightness while on) does not move it.
- Times are UTC. Convert to the house's zone (`hass-cli config get` shows `time_zone`) before saying them.
- A friendly name in the user's words can belong to several entities (a light and its plug, a sensor's temperature and humidity). Answer from the one whose domain fits the question.

## Counting and summarising

For a count or a list across many entities, let Home Assistant compute it in one call instead of reading every row:

```bash
hass-cli template "{{ states.light | selectattr('state','eq','on') | map(attribute='name') | list }}"
hass-cli template "{{ states.binary_sensor | selectattr('attributes.device_class','eq','window') | selectattr('state','eq','on') | list | count }}"
hass-cli template "{{ area_entities('客厅') | select('match','light[.]') | select('is_state','on') | list }}"
```

`area_entities` takes an area name or id and includes entities that inherit the area from their device. A template that fails prints Home Assistant's error; fix the expression rather than retrying it unchanged.

## Earlier

```bash
hass-cli system logbook --entity binary_sensor.front_door --start 2026-11-27T00:00:00+08:00
hass-cli system history --entities sensor.living_temperature,light.kitchen --start 2026-11-27T06:00:00+08:00 -o json
```

- The logbook is the readable story: "turned on", "opened", "triggered by automation X". Use it for "when", "who" and "what happened".
- History is every state value with its time; use it for "what was the temperature at 7".
- Both cover one day from `--start`, or the last day without it. For an older window, run once per day.
- Always pass `--entities` to `history`: without it Home Assistant returns every entity in the house.

## Watching

`hass-cli event watch state_changed` streams every change as NDJSON until interrupted, so it never returns by itself. Bound it, for example `timeout 30 hass-cli event watch state_changed`, and filter the output for the entity being tested.

## Verb index

| Verb | Use | Read when triggered |
|---|---|---|
| `state list [--domain d1,d2] [--search words]` | find entities and read their state | [reading now](#reading-now) |
| `state get <entity_id>` | one entity, every attribute | `-o json` when a script will read the attributes |
| `template "<jinja>"` or `--file` | counts, filters, area lookups in one call | [counting](#counting-and-summarising) |
| `system logbook`, `system history` | the past day from `--start` | [earlier](#earlier) |
| `event watch [event_type]` | live stream, never ends by itself | [watching](#watching) |
| `state set <entity_id> --state v` | overwrite the state machine only | [safety](#safety-and-escalation) first |

## Safety and escalation

- `state set` overwrites the entity's state in Home Assistant's state machine; it does **not** drive a device. The light stays as it is, and the integration overwrites the fake value at its next update. To change a device use `service call` ([the front door](../ha-shared/SKILL.md#control-a-device)). Use `state set` only when the user asks to fake a state for testing, and say that it is fake.
- Report what the state says, including `unavailable`; do not guess a value or "fix" an unavailable device by writing its state.
- Reading is always safe, but an answer about locks, doors and alarms should name the entity it came from so the user can tell which door was meant.
