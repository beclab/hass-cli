---
name: ha-automation
version: 0.2.0
description: "List, create, edit, run and delete automations, scripts and scenes."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli workflow --help"
---

# ha-automation

Manage automations, scripts, and scenes. Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md). To author a NEW automation from
a request, follow [`../ha-workflow-automation-builder/SKILL.md`](../ha-workflow-automation-builder/SKILL.md).

## Commands

```bash
hass-cli workflow automation list                 # entities + state (on/off, last_triggered)
hass-cli workflow automation get <id>             # full config (REST config editor)
hass-cli workflow automation save <id> --file a.yaml
hass-cli workflow automation delete <id>
hass-cli workflow automation reload
hass-cli workflow automation trigger automation.<name>
```

`script` and `scene` share the same verbs (`scene` trigger maps to
`scene.turn_on`).

## Dual storage (important)

Home Assistant has two automation storages:

- **UI storage** (`.storage`): 32-char hex id, plural keys
  `triggers/conditions/actions`.
- **File storage** (`automations.yaml`): any id, singular keys
  `trigger/condition/action`.

When you `save`, match the keys to the target. `save <id>` to a UI-managed id
should use the plural form. If `save` returns `HTTP 404`, the `config`
integration (default_config) is not enabled on that instance.

## Inspect failures

```bash
hass-cli workflow automation get <id>
hass-cli system logbook --entity automation.<name>
hass-cli raw ws trace/list --data '{"domain":"automation","item_id":"<id>"}'
```
