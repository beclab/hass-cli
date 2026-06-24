---
name: ha-registry
version: 0.1.0
description: "Manage Home Assistant areas, devices, entities, floors, and labels with hass-cli. Use to list/create/rename/delete areas, assign devices or entities to areas, organize with floors and labels, rename entity_ids, or audit the registry. These are WebSocket-only operations (config/*_registry/*)."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli registry --help"
---

# ha-registry

Organize the home: areas, devices, entities, floors, labels. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> All registry operations are WebSocket-only. hass-cli routes them
> automatically; you do not pick the transport.

## List

```bash
hass-cli registry area list
hass-cli registry device list
hass-cli registry entity list
hass-cli registry floor list
hass-cli registry label list
```

## Mutate

`create` takes only `--data` (the new entry's fields). `update` and `delete`
take the entry **id as a positional argument** — the CLI injects it as the
correct `<kind>_id` key for you. Extra `--data` fields are merged in.

```bash
# Create an area (id is derived from the name by HA)
hass-cli registry area create --data '{"name":"Garage"}'

# Rename an area (positional id -> area_id)
hass-cli registry area update garage --data '{"name":"Garage Bay"}'

# Assign a device to an area
hass-cli registry device update <device_id> --data '{"area_id":"garage"}'

# Move an entity to an area / rename it
hass-cli registry entity update light.x --data '{"area_id":"garage","name":"Bay Light"}'

# Delete
hass-cli registry area delete garage
```

> `--data` also accepts `@file.json`; on PowerShell prefer that over inline JSON
> (see ha-shared for the quoting note).

## Tips

- Get ids first via the matching `list`. Names are not ids.
- Entity/device area assignment: an entity inherits its device's area unless
  given its own `area_id`.
- For fields you are unsure about, inspect the frontend behavior or use
  `hass-cli raw ws config/entity_registry/get --data '{"entity_id":"..."}'`.
