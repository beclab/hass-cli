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

## Add an integration (config flow)

Adding an integration is a schema-driven, multi-step flow. Drive it with
`integration flow`:

```bash
hass-cli integration flow handlers --type helper    # what can be set up
hass-cli integration flow start <domain>            # begin; returns the first step
hass-cli integration flow step <flow_id> --data '{...}'   # submit each step's input
hass-cli integration flow abort <flow_id>           # cancel
```

Each step response has a `type`:

- `form` — fill `data_schema` fields, submit with `step`. `last_step` hints it's
  the final one. `errors` is populated on validation failure (re-submit).
- `menu` — submit `{"next_step_id":"<one of menu_options>"}`.
- `create_entry` — done; `result.entry_id` is the new config entry.
- `abort` — flow ended early; `reason` says why.
- `external_step` — needs a browser (e.g. OAuth); not CLI-friendly.

Worked example (the `random` helper integration):

```bash
ID=$(hass-cli integration flow start random -o json | jq -r .flow_id)
hass-cli integration flow step $ID --data '{"next_step_id":"sensor"}'   # menu -> form
hass-cli integration flow step $ID --data '{"name":"My Random"}'         # -> create_entry
```

> OAuth / hub integrations that need a browser redirect (`external_step`) can't
> be completed from the CLI. Helper and local integrations work fully.

## Discovery & in-progress flows

Discovered devices (bluetooth/zeroconf/dhcp/ssdp/usb/...) surface as in-progress
config flows:

```bash
hass-cli integration flow progress      # discovered + awaiting-input flows
hass-cli integration flow ignore <flow_id>   # stop suggesting a discovery
hass-cli raw ws usb/scan                 # trigger a USB rescan
```

To set up a discovered device, take its `flow_id` from `flow progress` and drive
it with `integration flow step` (it already has a started flow).
