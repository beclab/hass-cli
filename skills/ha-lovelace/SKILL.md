---
name: ha-lovelace
version: "0.0.0"
description: "Dashboards (Lovelace): add a card for a device, a graph or gauge to a dashboard, make a new dashboard, rename or hide one from the sidebar, undo a dashboard change, add a custom card resource. 仪表盘、卡片、加个图表."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-lovelace

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

A dashboard is a sidebar entry; its config is one document of views and cards, read and saved whole. Changing a dashboard never changes a device. The Energy dashboard's sources are [`ha-energy`](../ha-energy/SKILL.md), not a dashboard config.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| A card for a device on a dashboard | this file | `config get` to a file, add the card, `config save --data @file` |
| A graph, gauge, thermostat or another card type | [cards](references/ha-lovelace-cards.md) | the same round trip |
| A new dashboard | this file | `dashboard create`, then `config save --dashboard <url_path>` |
| Rename a dashboard, hide it from the sidebar, make it admin-only | this file | `dashboard update <id>` |
| Undo a dashboard change | this file | `config save` with the file kept before the change |
| A custom card (HACS, `custom:...`) to load | this file | `resource create` |

## Dashboards and their ids

```bash
hass-cli lovelace dashboard list
```

- The default dashboard (Overview) is not in that list. Every `config` verb reaches it by leaving out `--dashboard`.
- Another dashboard has two names: its `url_path` (`ops-room`, the address, which must contain a hyphen) for `config --dashboard`, and its `id` (`ops_room`) for `dashboard update` and `delete`. `url_path` cannot be changed after creation.
- `mode: storage` dashboards are edited here. `mode: yaml` ones live in the server's YAML files; `config save` cannot change them, so give the user the YAML instead.

## Editing a config

```bash
hass-cli lovelace config get --dashboard ops-room -o yaml > ops.yaml
cp ops.yaml ops.before.yaml                       # the undo
# add the card in the right view, then:
hass-cli lovelace config save --dashboard ops-room --data @ops.yaml
```

- `save` replaces the whole config. Edit what `get` returned; never write a dashboard from memory, or every other card is lost.
- Find the view by `title` or `path`. A view with `type: sections` holds cards under `sections[].cards`; other views hold them under `cards`. Put the card next to related ones, not at the end of the first view by default.
- A card for an entity is at least:

  ```yaml
  - type: tile
    entity: light.kitchen
  ```

  Take the `entity_id` from `hass-cli state list --search ...`; a card naming a missing entity shows a warning on the dashboard.
- The default dashboard may answer `config_not_found`, or a config with only a `strategy:` key: it is generated from the house's entities. Saving views to it takes it over, and from then on new devices no longer appear on it by themselves. Say so before the first save, or create a separate dashboard instead.
- `config delete` throws the stored config away and the dashboard goes back to generated. It is not an undo; saving the kept file is.

## New dashboards

```bash
hass-cli lovelace dashboard create --data '{"url_path":"energy-room","title":"能源","icon":"mdi:flash","show_in_sidebar":true}'
hass-cli lovelace config save --dashboard energy-room --data @energy-room.yaml
```

`update <id>` changes `title`, `icon`, `show_in_sidebar` and `require_admin`. A config file is `views:` with at least one view:

```yaml
views:
  - title: 客厅
    path: living
    type: sections
    sections:
      - type: grid
        cards:
          - type: tile
            entity: light.living_room
```

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `lovelace dashboard list` | nothing | the `id` and `url_path` of every dashboard except the default |
| `lovelace dashboard create --data`, `update <id> --data` | dashboard fields | [new dashboards](#new-dashboards) |
| `lovelace dashboard delete <id>` | the `id` | deletes its config with it; [safety](#safety-and-escalation) |
| `lovelace config get [--dashboard url_path] [--force]` | `url_path` | `-o yaml` to a file; `--force` rereads storage when a UI edit is not showing |
| `lovelace config save [--dashboard url_path] --data @f` | YAML or JSON, `@file` or inline | [editing a config](#editing-a-config) |
| `lovelace config delete [--dashboard url_path]` | `url_path` | back to generated, not an undo |
| `lovelace resource list`, `create`, `update <id>`, `delete <id>` | `{"res_type":"module","url":"/local/x.js"}` | a `custom:` card needs its resource; `res_type` is `module` for current custom cards, otherwise `js`, `css` or `html` |

## Safety and escalation

- Keep the file from `config get` before every save and say the change can be undone. One request, one change: do not tidy, reorder or restyle other cards.
- Taking over the generated default dashboard, deleting a dashboard and `config delete` lose work the user may not expect to lose; do them only on an explicit request, saying what goes.
- A resource loads code into every browser that opens Home Assistant. Add only a URL the user gave or a file under `/local/` they put there; do not invent a CDN address.
- Dashboards are per house, not per person: a change shows up for everyone who opens that dashboard. `require_admin` hides it from non-admin users; it does not restrict the devices on it.
