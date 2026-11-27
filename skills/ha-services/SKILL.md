---
name: ha-services
version: "0.0.0"
description: "Any Home Assistant service no device skill covers, and what fields a service takes: list and describe services, target a whole room, floor, device or label, get data back from a service (forecast, calendar events), refresh a sensor, fire an event."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-services

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Every change in Home Assistant is a service call (`domain.service`); there are no per-domain commands. The front door holds the basic call. This skill is for finding a service and its fields, for targets beyond one entity, and for services that answer with data. A device family with its own skill (lights, climate, openings, media) starts there.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "What can it do", the service behind a request | this file | `service list <domain>`, then `service describe <domain.service>` |
| Everything in a room, on a floor, on one device, or with a label | this file | one `service call` with `area_id`, `floor_id`, `device_id` or `label_id` |
| A forecast, calendar events, a to-do list: data back from a service | this file | `service call <domain.service>`, plus `--response` when optional |
| Refresh a sensor now, or switch a mixed set of devices | this file | `homeassistant.update_entity`, `homeassistant.turn_off` |
| Fire an event an automation listens for | this file | `event fire <event_type>` |

## Finding a service and its fields

```bash
hass-cli service list vacuum                     # every vacuum.* service, one row each
hass-cli service list light switch -o json       # with fields and targets
hass-cli service describe vacuum.send_command    # FIELD, REQUIRED, DESCRIPTION, EXAMPLE
```

`describe` lists every field the call accepts, including the ones Home Assistant groups under advanced sections. A service missing from `list` does not exist on this instance: its integration is not loaded, so do not call it on a guess.

## Targets

A target names what the service acts on. Put it at the top level of `--data`, next to the other fields (a nested `"target":{...}` is accepted too); each key takes one id or a list.

```bash
hass-cli service call light.turn_off --data '{"area_id":"living_room"}'
hass-cli service call light.turn_on --data '{"floor_id":"upstairs","brightness_pct":40}'
hass-cli service call switch.turn_off --data '{"label_id":"christmas"}'
hass-cli service call media_player.media_pause --data '{"device_id":"8f3c..."}'
```

- Area, floor, label and device ids are not names: get them from [`ha-registry`](../ha-registry/SKILL.md). `客厅` is a name; its id may be `ke_ting` or `living_room`.
- The service still filters by its domain: `light.turn_off` on an area turns off the lights there and leaves its switches and fans alone.
- Several keys add up: `area_id` plus `entity_id` acts on both.
- The CLI checks entity ids before sending, but not area, floor, label or device ids. A wrong one is not an error: the call answers that nothing changed. When that is unexpected, check the id.
- `"entity_id":"all"` means every entity of the domain in the house. Use it only when the user said "all" and meant the whole house.

## `--arguments` values

Each `key=value` becomes a typed JSON value: `true` and `false` become booleans, a number becomes a number, text starting with `[` or `{` that parses becomes a list or object, and anything else stays text. One flag may hold several pairs separated by commas; a comma inside brackets or quotes, or one not followed by `key=`, stays in the value, so `entity_id=light.a,light.b` and `rgb_color=[255,200,0]` work.

A text field whose value looks like a number or a boolean (`input_text.set_value` with `42` or `true`) arrives as the wrong type; send it in `--data` as a JSON string.

## Data back from a service

Some services exist to answer: `weather.get_forecasts`, `calendar.get_events`, `todo.get_items`. `service call` asks for their answer by itself and prints it, keyed by entity. Others answer only when asked, such as a script that returns a value or `conversation.process`; add `--response` to get theirs.

```bash
hass-cli service call weather.get_forecasts --data '{"entity_id":"weather.home","type":"daily"}'
hass-cli service call script.morning_summary --response
```

- The last line of `service describe <domain.service>` says whether the service returns data and whether it needs `--response`. No such line means it returns nothing, and `--response` is refused.
- In table mode only the answer is printed, as YAML; `-o json` prints the whole body, with the changed states next to `service_response`.

## Generic `homeassistant.*` services

| Service | For |
|---|---|
| `homeassistant.turn_on` / `turn_off` / `toggle` | a mixed list across domains (a light, a switch and a fan in one call); each entity gets its own domain's service |
| `homeassistant.update_entity` | refresh a polled sensor now instead of at its next poll |
| `homeassistant.reload_config_entry` | reload one integration by `entry_id` or an entity of it; [`ha-integrations`](../ha-integrations/SKILL.md) owns integrations |
| `homeassistant.restart`, `homeassistant.stop` | never from here: restarts are [`ha-system`](../ha-system/SKILL.md), on explicit request |

Prefer the device's own domain when the call needs more than on or off: its fields are the ones `service describe` lists.

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `service list [domain...]` | domains to narrow to | [finding a service](#finding-a-service-and-its-fields); `-o json` adds the fields |
| `service describe <domain.service>` | one service | before an unfamiliar call, after a rejected field, or to learn whether it returns data |
| `service call <domain.service> [entity_id...]` | `--entity`, `--arguments`, `--data` | [the front door](../ha-shared/SKILL.md#control-a-device), then [targets](#targets) |
| `service call ... --response` | a service whose answer is optional | [data back](#data-back-from-a-service) |
| `event fire <event_type> [--data '{...}']` | event type and data | only an event type the user's automations listen for |

## Safety and escalation

- A target wider than one entity (an area, a floor, a label, `all`) is a change to many devices: use it only when the user named the room, floor or group, and say how many entities changed.
- A service found through `service list` that the user did not ask for is not a shortcut: do not call `*.reload`, `*.delete_*`, `*.reset`, `recorder.purge` or similar on the way to another request.
- `event fire` can trigger automations that unlock, open or announce things. Fire only an event the user named, and say which automations listen to it when that is known.
- Locks, alarms and covers called through a generic service follow [the front door's rules](../ha-shared/SKILL.md#safety-and-escalation) like any other call.
