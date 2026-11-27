---
name: ha-helpers
version: "0.0.0"
description: "Helpers: set a timer, count something, a guest-mode or vacation switch, a number, a dropdown, a text or date value automations read; create, rename or delete input_boolean, input_number, input_select, input_text, input_datetime, input_button, counter, timer and schedule."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-helpers

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

A helper is a value the user owns rather than a device: a switch for guest mode, a number for a target temperature, a timer, a counter. Automations read and set them. Two different jobs, two different commands:

- **Use a helper** (turn it on, set its value, start the timer) is a service call on its entity, like any device.
- **Manage the helper itself** (create, rename, change its range or options, delete) is `hass-cli helper <type> ...`.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Turn guest mode on, set a value, press a helper button | this file | one `service call` on `input_*.<id>` |
| "Set a timer for 5 minutes" | this file | `timer.start` with `duration`, after creating a timer if none exists |
| Add one, subtract one, reset a count | this file | `counter.increment` / `decrement` / `reset` |
| A new helper an automation will use | this file | `helper <type> create --data '{"name":...}'` |
| Rename a helper, change its range or options | this file | `helper <type> list`, then `update <id>` |

## Using a helper

| Type | Services |
|---|---|
| `input_boolean` | `turn_on`, `turn_off`, `toggle` |
| `input_number` | `set_value` (`value`), `increment`, `decrement` |
| `input_select` | `select_option` (`option`), `select_next`, `select_previous`, `select_first`, `select_last` |
| `input_text` | `set_value` (`value`) |
| `input_datetime` | `set_datetime` with `date`, `time` or `datetime` |
| `input_button` | `press` |
| `counter` | `increment`, `decrement`, `reset`, `set_value` (`value`) |
| `timer` | `start` (optional `duration` `"00:05:00"`), `pause`, `cancel`, `finish`, `change` |

```bash
hass-cli service call input_boolean.turn_on --entity input_boolean.guest_mode
hass-cli service call input_select.select_option --data '{"entity_id":"input_select.house_mode","option":"Away"}'
hass-cli service call timer.start --data '{"entity_id":"timer.kitchen","duration":"00:05:00"}'
```

- `input_number.set_value` outside the helper's `min`/`max` is refused; read them from `state get`.
- `input_select.select_option` takes an option exactly as written in the helper's `options` attribute.
- A timer counts down and then fires a `timer.finished` event; nothing rings or speaks unless an automation listens for it. Say so when the user asked for a reminder, and offer one ([`ha-automation`](../ha-automation/SKILL.md), [`ha-notify`](../ha-notify/SKILL.md)).
- A `schedule` has no service to switch it: it is `on` inside its time slots. Change the slots with `helper schedule update`.

## Managing helpers

```bash
hass-cli helper input_boolean list
hass-cli helper input_boolean create --data '{"name":"Guest Mode","icon":"mdi:account"}'
hass-cli helper input_number update target_temp --data '{"min":16,"max":28}'
hass-cli helper timer delete kitchen
```

- `create` needs `name`; Home Assistant makes the id from it and prints it. The entity is `<type>.<id>`. A Chinese name gets a transliterated id, so read it from the output or from `state list --domain <type> --search <name>`, not from the name.
- `update` and `delete` take the helper id from `list`, which is usually but not always the entity's object id.
- `list` shows helpers made in the UI or by this CLI. A helper defined in YAML (`configuration.yaml`) appears in `state list` but not here, and cannot be updated or deleted from the CLI: tell the user where it lives.
- Changing a select's options for good is `update` with the full `options` list; `input_select.set_options` changes them only until the next restart.

| Type | Fields beyond `name` and `icon` |
|---|---|
| `input_boolean` | `initial` |
| `input_number` | `min`, `max`, `step`, `mode` (`slider` or `box`), `unit_of_measurement`, `initial` |
| `input_select` | `options` (list of text), `initial` |
| `input_text` | `min`, `max` (length), `pattern`, `mode` (`text` or `password`), `initial` |
| `input_datetime` | `has_date`, `has_time` (at least one `true`), `initial` |
| `counter` | `initial`, `minimum`, `maximum`, `step`, `restore` |
| `timer` | `duration` (`"00:05:00"`), `restore` |
| `schedule` | `monday` ... `sunday`, each a list of `{"from":"07:00:00","to":"09:00:00"}` |

For a field not in this table, create the helper with the fields that are, then read it back with `helper <type> list -o json`.

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `helper <type> list` | nothing | ids for `update` and `delete`; YAML helpers are absent |
| `helper <type> create --data '{...}'` | `name` plus the type's fields | the field table above |
| `helper <type> update <id> --data '{...}'` | only the fields to change | [managing helpers](#managing-helpers) |
| `helper <type> delete <id>` | helper id | only on an explicit request |
| `service call <type>.<service>` | the helper's entity id | [using a helper](#using-a-helper) |

## Safety and escalation

- Deleting a helper breaks every automation, script and dashboard card that reads it, and they fail silently. Delete only the helper the user named, and say that automations using it will stop working.
- Before changing a helper's range or options, check which automations read it (`workflow automation get`, [`ha-automation`](../ha-automation/SKILL.md)); a removed option an automation selects makes that automation fail.
- Creating a helper is a configuration change. For a one-off timer it is fine; say that a timer helper was created and that it stays for next time.
- A helper named like a security mode ("away", "alarm armed") may drive automations that lock or arm things: setting it is the same as asking for those, so set it only when the user asked.
