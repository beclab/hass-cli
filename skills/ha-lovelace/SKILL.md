---
name: ha-lovelace
version: 0.1.0
description: "Read and write Home Assistant Lovelace dashboards with hass-cli: list dashboards, create/update/delete storage-mode dashboards, get and save a dashboard's full view/card config (YAML or JSON), and manage custom JS/CSS resources. Use for 'show my dashboards', 'edit the dashboard config', 'add a card', 'create a new dashboard', 'export/import a dashboard' requests."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli lovelace --help"
---

# ha-lovelace

Dashboards, their stored view/card config, and custom resources. All WebSocket.
**Prerequisite:** [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Mental model

- A **dashboard** is a container (sidebar entry). Storage-mode dashboards are
  editable via the API; YAML-mode ones are read-only here.
- Each dashboard has a **config** (the `views` array of cards). The default
  dashboard is addressed by omitting `--dashboard` (null url_path); named ones
  by their `url_path`.
- **Resources** are extra JS/CSS modules loaded into the frontend.

## Dashboards

```bash
hass-cli lovelace dashboard list
hass-cli lovelace dashboard create --data '{"url_path":"ops-room","title":"Ops Room"}'
hass-cli lovelace dashboard update <dashboard_id> --data '{"show_in_sidebar":false}'
hass-cli lovelace dashboard delete <dashboard_id>
```

> `url_path` MUST contain a hyphen (e.g. `ops-room`). The returned `id` (e.g.
> `ops_room`) is what `update`/`delete` take, while `--dashboard` for config
> takes the `url_path`.

## Read / write a dashboard config

The config is the whole `views` tree. Round-trip it as YAML for editing:

```bash
# Export current config (default dashboard) to a file
hass-cli lovelace config get -o yaml > dashboard.yaml

# Export a named dashboard
hass-cli lovelace config get --dashboard ops-room -o yaml > ops.yaml

# Edit, then save it back
hass-cli lovelace config save --dashboard ops-room --file ops.yaml

# Revert to auto-generated
hass-cli lovelace config delete --dashboard ops-room
```

Minimal config file:

```yaml
views:
  - title: Home
    cards:
      - type: markdown
        content: Hello from hass-cli
```

> `config get` on a default storage dashboard that was never edited may error
> with "config not found" — it's still auto-generated. Save a config first, or
> create a dedicated dashboard.

## Add a card (typical agent flow)

1. `lovelace config get --dashboard <path> -o yaml > d.yaml`
2. Append a card under the right view's `cards:` list.
3. `lovelace config save --dashboard <path> --file d.yaml`

Discover valid `entity_id`s with `hass-cli state list` first.

## Resources

```bash
hass-cli lovelace resource list
hass-cli lovelace resource create --data '{"res_type":"module","url":"/local/my-card.js"}'
hass-cli lovelace resource update <resource_id> --data '{"url":"/local/my-card-v2.js"}'
hass-cli lovelace resource delete <resource_id>
```

`res_type` is one of `module|js|css|html`.
