---
name: ha-registry
version: "0.0.0"
description: "Rooms, floors and labels; which room a device is in, move it to another room, rename or hide a device or entity, give it a voice alias, add or delete an area, floor, label or category; the area_id, floor_id and label_id other calls target."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-registry

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

How the house is organised: floors hold areas (rooms), areas hold devices, devices hold entities, and labels tag any of them across rooms. Calling a service on a whole room uses the ids from here; the call itself is [`ha-services`](../ha-services/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| The id of a room, floor or label for a service call | this file | `registry area list` (or `floor`, `label`) |
| "Which room is it in", "what is in the kitchen" | this file | one `template` with `area_name` or `area_entities` |
| Move a device to another room | this file | `registry device update <id> --data '{"area_id":...}'` |
| Rename or hide an entity, give it a voice alias | this file | `registry entity update <entity_id> --data '{...}'` |
| Add a room or floor, tag things with a label | this file | `registry area create`, then `update` on each device |

## Ids

| Registry | Id key | Name key | Notes |
|---|---|---|---|
| floor | `floor_id` | `name` | `level` orders floors: 0 ground, 1 first, -1 basement |
| area | `area_id` | `name` | `floor_id` places it on a floor |
| label | `label_id` | `name` | `color`, `icon`, `description` |
| device | `id` | `name_by_user`, else `name` | `area_id`, `labels` |
| entity | `entity_id` | `name`, else `original_name` | `area_id`, `labels`, `aliases`, `hidden_by`, `disabled_by` |

- An id is not a name. Home Assistant makes it from the name at creation and keeps it through renames, so `客厅` may be `ke_ting` or `living_room`; read it from `list`.
- `registry entity list` prints every entity in the house. For one entity, `raw ws config/entity_registry/get --data '{"entity_id":"light.kitchen"}'` shows its `device_id`, area and labels.
- Areas, floors and labels are created here: `create` takes `--data` with at least `name`. Devices and entities come from integrations and are never created here.
- `update` and `delete` take the id as the argument and only the fields to change in `--data`.

## Rooms

An entity is in its device's area unless it has its own `area_id`; a set `area_id` on the entity overrides the device. To move a whole device, update the device, not each entity. Templates resolve this inheritance for you:

```bash
hass-cli template "{{ area_name('light.desk_lamp') }}"
hass-cli template "{{ area_entities('kitchen') }}"
hass-cli registry device update 8f3c2a... --data '{"area_id":"study"}'
hass-cli registry entity update light.desk_lamp --data '{"area_id":null}'   # follow the device again
```

## Names, hiding and voice

```bash
hass-cli registry entity update light.0x1234_light --data '{"name":"书桌灯","aliases":["台灯","desk lamp"]}'
hass-cli registry device update 8f3c2a... --data '{"name_by_user":"Study lamp"}'
hass-cli registry entity update sensor.plug_voltage --data '{"hidden_by":"user"}'
```

- `name` changes what the user sees and says; the entity id stays, so automations keep working. Changing the id itself is `new_entity_id`, which breaks every automation, script and card that names the old id.
- `aliases` are extra names Assist matches by voice; exposing the entity to Assist is [`ha-assist`](../ha-assist/SKILL.md).
- `hidden_by: "user"` hides it from dashboards and voice but keeps it working; `null` shows it again. `disabled_by: "user"` stops the entity altogether: it leaves the state list and its integration stops updating it.
- `labels` and `aliases` replace the whole list: read the current one first and send it with the change.

## Categories

Categories sort automations, scripts and scenes in the UI, per scope; every call needs `--scope`:

```bash
hass-cli registry category create --scope automation --data '{"name":"Lighting"}'
hass-cli registry entity update automation.sunset_lights --data '{"categories":{"automation":"<category_id>"}}'
```

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `registry <area\|floor\|label> list` | nothing | ids for targets and `area_id` fields |
| `registry device list`, `registry entity list` | nothing | [ids](#ids): device ids are hex strings; the entity list is long |
| `registry <area\|floor\|label> create --data '{"name":...}'` | name plus extra fields | [ids](#ids) for the extra fields |
| `registry <kind> update <id> --data '{...}'` | any kind | [rooms](#rooms), [names](#names-hiding-and-voice) |
| `registry <area\|floor\|label> delete <id>` | id | only on an explicit request |
| `registry entity delete <entity_id>` | entity id | removes the registry entry only; for a leftover entity its integration no longer provides. A live one comes back at the next reload: hide or disable it instead |
| `registry category <verb> --scope <scope>` | scope plus id | [categories](#categories) |
| Removing a device | | not a registry verb: remove its integration or config entry ([`ha-integrations`](../ha-integrations/SKILL.md)) |

## Safety and escalation

- Deleting an area leaves its devices without a room, and calls on that area stop reaching them; deleting a floor or label drops it from everything that had it. Delete only what the user named, and say what was left unassigned.
- `registry entity delete`, `new_entity_id` and `disabled_by` break automations and dashboards that use the entity. Do them only when asked, and say so; for "I don't want to see it", hide instead.
- Moving or renaming many devices at once ("tidy up the rooms") is a plan the user approves first: list the changes, then make them.
