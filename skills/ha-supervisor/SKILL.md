---
name: ha-supervisor
version: 0.1.0
description: "Manage Home Assistant add-ons and the Supervisor/OS layer with hass-cli. Use to list/install/uninstall/update/start/stop/restart add-ons, browse the add-on store, read add-on logs, or check Core/Supervisor/OS info. Works through Core's supervisor/api WebSocket proxy with a regular admin token (no separate supervisor token needed)."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli addon --help"
---

# ha-supervisor

Add-ons and the Supervisor/OS layer. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> Availability: these only work on **Supervised / HA OS** installs. On a Core
> (Container) install there is no Supervisor and the commands fail fast with
> "this Home Assistant instance has no Supervisor". A regular **admin**
> `HASS_TOKEN` is enough — Core proxies Supervisor calls via the
> `supervisor/api` WebSocket command. If the token is not an admin, the CLI
> reports that you need an admin long-lived token.

## Typed commands (preferred)

```bash
hass-cli addon list                     # installed add-ons
hass-cli addon info core_mosquitto
hass-cli addon start  core_mosquitto
hass-cli addon stop   core_mosquitto
hass-cli addon restart core_mosquitto
hass-cli addon logs   core_mosquitto

hass-cli supervisor info                # supervisor/host/os/core info
hass-cli supervisor host
hass-cli supervisor os
hass-cli supervisor core
```

## Add-on lifecycle via Core services (also works)

```bash
hass-cli service call hassio.addon_start   --data '{"addon":"core_mosquitto"}'
hass-cli service call hassio.addon_restart --data '{"addon":"core_mosquitto"}'
hass-cli service call hassio.addon_stop    --data '{"addon":"core_mosquitto"}'
hass-cli service call hassio.backup_full
```

## Anything else (raw Supervisor proxy)

For endpoints without a typed command (install/uninstall/update, options, store):

```bash
hass-cli raw ws supervisor/api --data '{"endpoint":"/store/addons/core_mosquitto/install","method":"post"}'
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons/core_mosquitto/options","method":"post","data":{"options":{}}}'
hass-cli raw ws supervisor/api --data '{"endpoint":"/store","method":"get"}'
```

## Notes

- Add-ons also surface as entities (switch to start/stop, sensors for cpu/mem,
  `update.*` for available updates) — see `ha-states` / `update` entities.
- `HASS_SUPERVISOR_TOKEN` is reserved for direct Supervisor access when Core is
  down; the typed/proxy paths above use the normal `HASS_TOKEN`.
