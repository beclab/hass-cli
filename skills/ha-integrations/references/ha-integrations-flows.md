# Config flows: adding, discovering and reconfiguring

Read when adding an integration, setting up a discovered device, finishing a re-authentication, or changing an integration's options. Back to [`ha-integrations`](../SKILL.md).

Home Assistant adds an integration through a flow: a short series of forms whose fields it defines at each step. The CLI drives the same flow the UI does, one answer at a time; nothing is created until the flow ends in `create_entry`.

## Adding an integration

```bash
hass-cli integration flow handlers --type integration -o json   # domains that can be added; --type helper for helpers
hass-cli integration flow start shelly -o json                  # returns flow_id and the first step
hass-cli integration flow step <flow_id> --data '{"host":"192.168.1.40"}'
hass-cli integration flow get <flow_id> -o json                 # re-read the current step
hass-cli integration flow abort <flow_id>                       # give up; nothing is kept
```

Every answer has a `type`, and it decides the next command:

| `type` | Do |
|---|---|
| `form` | Send the fields of `data_schema` (each has `name`, `required`, and a type or `options`) with `flow step`. `errors` set means the last answer was refused: fix that field and send the step again. |
| `menu` | Send `{"next_step_id":"<one of menu_options>"}`. |
| `create_entry` | Done. `result.entry_id` is the new entry; its devices and entities appear within seconds. |
| `abort` | Ended without an entry; `reason` says why (`already_configured`, `cannot_connect`, `no_devices_found`). Report it. |
| `external` | Needs a browser sign-in (OAuth) at `url`. The CLI cannot finish it: [what to tell the user](../SKILL.md#safety-and-escalation). |
| `show_progress` | Home Assistant is working (pairing, downloading). Wait a few seconds and `flow get` again. |

- Ask the user for every required field you cannot read from the house (host, password, API key). Optional fields keep their defaults unless the user named a value.
- A field list that only has `next_step_id` choices is a menu, not a form; sending form data there is refused.

## Discovered devices

Home Assistant finds devices on the network, Bluetooth and USB by itself and parks each as a flow waiting for the user:

```bash
hass-cli integration flow progress -o json      # flow_id, handler, context.source (zeroconf, dhcp, bluetooth, usb...)
hass-cli integration flow step <flow_id> --data '{}'
hass-cli integration flow ignore <flow_id>      # only when the user does not want it
```

A discovered flow is already started: continue it with `flow step` (often an empty confirm), do not `flow start` the domain again. A flow whose `context.source` is `reauth` is an existing integration asking for new credentials; it ends in `abort` with `reauth_successful`, not in a new entry.

## Options of an existing entry

What an integration is configured with after setup (scan interval, which sensors to create, a changed host) is its options flow. It has no typed command; it is a flow over REST, keyed by the `entry_id`:

```bash
hass-cli raw api POST config/config_entries/options/flow --data '{"handler":"<entry_id>"}'
hass-cli raw api POST config/config_entries/options/flow/<flow_id> --data '{"scan_interval":60}'
```

The answers have the same `type`s as above; `create_entry` here means the options were saved. Not every integration has options: an `abort` or `HTTP 400` on the first call means this one does not.
