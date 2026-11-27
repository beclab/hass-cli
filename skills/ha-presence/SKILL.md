---
name: ha-presence
version: "0.0.0"
description: "Who is home, is anyone home, where is someone, when did they leave or arrive; people, zones, phones as device trackers, presence groups, and adding a zone or a tracker to a person."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-presence

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Where the people of the house are, as Home Assistant sees it. An automation that should react to someone arriving or leaving is built by [the builder playbook](../ha-workflow-automation-builder/SKILL.md); this skill finds the entities it uses.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Who is home", "is Alice home", "where is Bob" | this file | `state list --domain person` |
| "Is anyone home" | this file | `state get zone.home`: the number of people in it |
| "When did she leave", "when did he get home" | this file | `system logbook --entity person.<id>` |
| "When I get home, turn on..." | [the builder playbook](../ha-workflow-automation-builder/SKILL.md) | an automation on `person.<id>` |

## The model

| Entity | State | Notes |
|---|---|---|
| `person.*` | `home`, `not_home`, a zone's name, or `unknown` | one per person; combines their trackers |
| `device_tracker.*` | the same values | one per phone, router client or tag; `attributes.source_type` is `gps`, `router`, `bluetooth` |
| `zone.*` | the number of people inside | `zone.home` is the house; attributes `latitude`, `longitude`, `radius` |
| `group.*` of people or trackers | `home` when any member is home | `attributes.entity_id` lists the members |

- Ask about people through `person.*`, not trackers: a person is home when their best tracker says so, and an old phone left in a drawer is one tracker among several.
- `attributes.source` on a person names the tracker that decided its state; `device_trackers` lists all of them. A person with no trackers stays `unknown`.
- A router or Bluetooth tracker only knows home or away; a zone name needs a GPS tracker (the Companion app).
- `not_home` means outside every zone, not "unknown": a phone that stopped reporting keeps its last state. `last_changed` on `state get` says how old it is.

## Answering

```bash
hass-cli state list --domain person
hass-cli state get person.alice                      # source, device_trackers, last_changed
hass-cli system logbook --entity person.alice --start 2026-11-27T00:00:00+08:00
```

Say the zone, not the coordinates, unless the user asked for a position. Times in `last_changed` and the logbook are UTC; give them in the house's time zone (`hass-cli config get`).

## Zones and trackers

Zones made in the UI and people are changed over WebSocket; there is no typed command, so these go through `raw ws`:

```bash
hass-cli raw ws zone/list
hass-cli raw ws zone/create --data '{"name":"Office","latitude":31.23,"longitude":121.47,"radius":150}'
hass-cli raw ws person/list
hass-cli raw ws person/update --data '{"person_id":"alice","device_trackers":["device_tracker.alice_phone","device_tracker.alice_watch"]}'
```

- `zone.home` is not a zone entry: its position and radius are the instance's location (`hass-cli config get`).
- `person/update` replaces the whole `device_trackers` list: start from `person/list` and add to it.
- A person or zone defined in YAML shows in `state list` but not in these lists, and cannot be changed from here.

## Verb index

| Verb | For | Read when triggered |
|---|---|---|
| `state list --domain person` (or `zone`, `device_tracker`) | who is where now | [the model](#the-model) |
| `state get person.<id>` | which tracker decided, since when | [the model](#the-model) |
| `system logbook --entity`, `system history --entities` | arrivals and departures | [`ha-states`](../ha-states/SKILL.md) for the time window |
| `raw ws zone/*`, `raw ws person/*` | add a zone, give a person a tracker | [zones and trackers](#zones-and-trackers) |
| `state set device_tracker.<id> --state home` | fake a location to test an automation | [safety](#safety-and-escalation) first |

## Safety and escalation

- Presence is personal. Answer about the people the user asks about; do not volunteer where others in the house are, and give a zone rather than a position.
- `state set` on a tracker or person only overwrites Home Assistant's state machine; it does not move anyone, and the next real update replaces it. Use it only when the user asks to test an automation, and say it was a fake value. It can fire real automations (arrival unlocks a door, departure arms the alarm): say which ones before faking an arrival or departure.
- Deleting a person or a zone, or removing a tracker from a person, breaks automations that use them: only on an explicit request.
