---
name: ha-integrations
version: 0.1.0
description: "Manage Home Assistant integrations (config entries) and device discovery with hass-cli: list integrations and their state, reload, enable/disable, delete, update entry options, and inspect discovered/in-progress config flows. Use for 'list my integrations', 'reload the X integration', 'why is this integration failing', 'disable this entry', 'what was discovered on my network' requests."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli integration --help"
---

# ha-integrations

Config entries are the runtime instances of integrations. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> List/get/update/enable/disable go over WebSocket; reload and delete are
> REST-only config endpoints. The CLI routes both for you.

## List & inspect

```bash
hass-cli integration list                       # all config entries
hass-cli integration list --domain hue          # only one integration's entries
hass-cli integration get <entry_id>
```

Key fields per entry: `entry_id`, `domain`, `title`, `state`
(`loaded`/`setup_retry`/`setup_error`/`not_loaded`/...), `disabled_by`,
`reason` (why it failed), `source` (how it was set up).

Find broken integrations:

```bash
hass-cli integration list -o json | rg -B2 -A2 '"state": "setup_(error|retry)"'
```

## Lifecycle

```bash
hass-cli integration reload  <entry_id>         # re-run setup (no restart)
hass-cli integration disable <entry_id>         # stop loading it
hass-cli integration enable  <entry_id>
hass-cli integration delete  <entry_id>         # remove the entry entirely
```

`reload`/`disable`/`delete` return `{"require_restart": ...}`; if true, the
change needs a Home Assistant restart to fully apply.

## Update entry preferences

```bash
hass-cli integration update <entry_id> --data '{"title":"Living Room Hue","pref_disable_polling":true}'
```

Updatable fields: `title`, `pref_disable_new_entities`, `pref_disable_polling`.
Per-integration *options* (the options flow) are interactive multi-step and are
not exposed as a single command; drive them with `raw ws` against
`config_entries/options/flow` if needed.

## Discovery & in-progress flows

Discovered devices surface as in-progress config flows:

```bash
hass-cli raw ws config_entries/flow/progress -o json     # awaiting-setup discoveries
hass-cli raw api GET 'config/config_entries/flow_handlers' -o json   # what can be set up
hass-cli raw ws usb/scan -o json                         # trigger a USB rescan
```

Starting/finishing a config flow is an interactive, schema-driven sequence
(`config/config_entries/flow` POST then step submissions); use `raw api`/`raw ws`
for that. For everyday work, prefer reload/enable/disable above.
