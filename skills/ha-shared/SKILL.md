---
name: ha-shared
version: "0.0.0"
description: "Read first for any Home Assistant request: find a device and call a service on it in two commands, when to check afterwards, stopping a script or automation that keeps changing a device, and which ha-* skill owns the rest."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-shared

The front door of the `ha-*` suite. Every other skill assumes this one has been read: it holds the control that most requests need and the rules no domain skill repeats. Flags are `hass-cli <command> --help`'s job.

## Control a device

Turning something that exists on or off, dimming it, setting a temperature or a position takes two commands. Do not read another skill or run `service describe` for these.

```bash
# 1. Find it by name and/or domain: a few rows, not the whole house
hass-cli state list --domain light --search kitchen
hass-cli state list --search 客厅

# 2. Call the service on the entity_id from step 1
hass-cli service call light.turn_off --entity light.kitchen
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","brightness_pct":30}'
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","temperature":21}'
hass-cli service call cover.close_cover --entity cover.bedroom
```

- `--search` matches the entity_id and the friendly name, every word. No row: try one word, the other language, or `--domain` alone. Several rows: take the one whose name fits, or ask when two are equally likely.
- The usual services: `turn_on` / `turn_off` / `toggle` for light, switch, fan and input_boolean; `climate.set_temperature` / `set_hvac_mode`; `cover.open_cover` / `close_cover` / `set_cover_position`; `lock.lock` / `unlock`; `media_player.media_play_pause` / `volume_set`. The domain skills hold the rest.
- Colours, lists and nested values go in `--data '{...}'`; `--entity` and `--arguments key=value` cover the flat cases.

## Did it work

The call prints the entities it changed; that is the confirmation, and no `state get` is needed. `No entity changed state` means the device already was that way, which is not a failure.

Check further only when the output says something else:

- **A `Note:` names a running script or automation.** It controls the device and will change it again at its next step, so the call alone does not hold. When the user wants the device to stay as asked ("turn the light off" while a colour loop runs), stop the run with the command the note prints, then call the service again:

  ```bash
  hass-cli workflow script stop script.color_cycle
  hass-cli service call light.turn_off --entity light.kitchen
  ```

  `workflow automation stop` ends an automation's run and leaves it enabled. Stopping is a change the user did not name: say that the loop was stopped.
- **The user asks to verify**, or a device is known not to report back: `hass-cli state get <entity_id>`.

## Errors on a call

The CLI names the cause and the next command for an unknown entity, a missing target and an unknown field; follow it. For an `HTTP 400` it cannot explain, `hass-cli service describe <domain.service>` lists the fields.

Two failed calls in a row on one request: stop and say what failed rather than trying more variations.

## Which skill

Each skill's description says what it is for; [the suite map](references/ha-shared-suite-map.md) lists them by request when two seem to fit. Device-family skills (lights, climate, openings, media) only add what is particular to the family, on top of the control above.

Scripts, automations and scenes, including "make it loop", "every few seconds" and "stop that", are [`ha-automation`](../ha-automation/SKILL.md).

## Connection and raw access

| Trigger | Read |
|---|---|
| `no server configured`, `HTTP 401`, `auth_invalid`, TLS errors, profiles, env vars | [connection](references/ha-shared-connection.md) |
| Quoting `--data` on PowerShell, `@file` input, output formats | [connection](references/ha-shared-connection.md) |
| No typed command covers the request: `raw api`, `raw ws` | [raw access](references/ha-shared-raw.md) |

## Safety and escalation

- Act on what the user named. A name that matches several devices in different rooms is a question, not a guess; a request for "all lights" is not a reason to also switch fans.
- Locks, alarms, garage doors and anything that opens the house: do what was asked, and say what changed. Do not unlock or disarm as a side step of another request.
- Stopping a script or automation, deleting anything, restoring a backup and restarting Home Assistant are changes the user should hear about; the last three need their explicit request.
- Never print a token. `hass-cli profile show` masks it; do not echo `HASS_TOKEN`.
