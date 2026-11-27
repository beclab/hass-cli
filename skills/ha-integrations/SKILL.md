---
name: ha-integrations
version: "0.0.0"
description: "Integrations (config entries): add one, set up a discovered device, see which are broken or failing, reload, enable, disable, rename or remove one, change its options. Hue, MQTT, ESPHome, HomeKit, a new device found on the network."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-integrations

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

An integration is the code that talks to a brand or protocol (Hue, MQTT, ESPHome); each time it is set up it makes a **config entry**, which owns devices and their entities. One integration can have several entries (two Hue bridges). Every verb here takes the entry's `entry_id`, never an entity id. Pairing devices onto a Zigbee, Z-Wave, Matter or Thread network is [`ha-gateway`](../ha-gateway/SKILL.md); renaming or moving the devices an entry made is [`ha-registry`](../ha-registry/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Is anything broken", "why is Hue unavailable" | this file | `integration list`, then `get` on the failing entry |
| Reload one after a network blip or a changed device | this file | `integration reload <entry_id>` |
| Turn one off for now, or back on | this file | `integration disable` / `enable <entry_id>` |
| Add an integration, or set up a device Home Assistant discovered | [config flows](references/ha-integrations-flows.md) | `flow step` until `create_entry` |
| Change an integration's options (polling interval, which sensors) | [config flows](references/ha-integrations-flows.md) | the options flow |
| Remove an integration | this file | `integration delete <entry_id>`, on an explicit request |

## Entries and their state

```bash
hass-cli integration list                       # ID, DOMAIN, TITLE, STATE
hass-cli integration list --domain hue
hass-cli integration get <entry_id> -o json     # reason, source, disabled_by
```

| `state` | Means | Next step |
|---|---|---|
| `loaded` | working | none |
| `setup_retry` | the device or service did not answer; Home Assistant keeps retrying | check it is powered and reachable, then `reload` |
| `setup_error` | setup failed and will not retry by itself | read `reason`, then `hass-cli system errorlog` for that domain; [`ha-system`](../ha-system/SKILL.md) |
| `not_loaded` | disabled (`disabled_by` is set) or never started | `enable` if the user disabled it |
| `migration_error`, `failed_unload` | needs a Home Assistant restart or a fix in the integration | report it; a restart is [`ha-system`](../ha-system/SKILL.md) |

- An entry whose `source` is `ignore` is a discovery the user dismissed, not a working integration. Deleting it lets the discovery come back.
- Credentials that expired show up as a repair issue (`system repairs`) and a `reauth` flow in `integration flow progress`; finishing it is a config flow with the new password.
- Entities of an entry that is not `loaded` are `unavailable` (or gone from the list, when the entry is disabled). Fixing the entry fixes them; do not chase each entity.

## Changing an entry

- `reload` re-runs setup without a restart. It is the first thing to try for `setup_retry`, and after the device behind it changed address or firmware.
- `disable` stops the entry and its entities until `enable`; nothing is deleted. Prefer it to `delete` when the user wants something "off" or "gone for now".
- `delete` removes the entry with its devices and entities; the automations and dashboards that used them break. Setting it up again is a new config flow.
- `reload`, `disable` and `delete` answer `require_restart`. When it is `true`, the change is not complete until Home Assistant restarts; tell the user rather than restarting unasked.
- `update` changes the entry's own preferences only: `title`, `pref_disable_new_entities`, `pref_disable_polling`. What the integration itself is configured with (host, interval, enabled sensors) is its options flow.

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `integration list [--domain d]` | nothing | the state table above; `-o json` adds `reason` and `disabled_by` |
| `integration get` | entry id | why one entry failed |
| `integration reload`, `enable`, `disable` | entry id | the notes above on `require_restart` |
| `integration update --data` | entry id; title and `pref_*` only | renaming an entry; options are a flow |
| `integration delete` | entry id | only on an explicit request |
| `integration flow handlers`, `start`, `get`, `step`, `abort` | domain or flow id | [config flows](references/ha-integrations-flows.md) |
| `integration flow progress`, `ignore` | flow id | discovered devices: [config flows](references/ha-integrations-flows.md) |

## Safety and escalation

- `delete` only when the user asked to remove that integration by name. Name the entry and what goes with it ("removes the Hue bridge and its 12 lights") and, if the user only wants it quiet, offer `disable` instead.
- Several entries of one domain: ask which (by `title`) before changing one; do not act on all of them.
- Passwords, API keys and pairing codes come from the user. Never invent them, never echo them back, and do not retry a rejected credential with variations.
- A flow that needs a browser (OAuth sign-in) cannot be finished here: give the user the URL from the flow or point them to Settings > Devices & services, and abort the flow if they decline.
- `ignore` hides a discovered device for good; use it only when the user says they do not want that device.
