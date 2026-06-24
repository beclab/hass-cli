---
name: ha-supervisor
version: 0.1.0
description: "Manage Home Assistant add-ons and the Supervisor/OS layer with hass-cli. Use to list/install/uninstall/update/start/stop/restart add-ons, browse the add-on store, read add-on logs, or check Core/Supervisor/OS info. Works through Core's supervisor/api WebSocket proxy with a regular admin token (no separate supervisor token needed)."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli raw ws --help"
---

# ha-supervisor

Add-ons and the Supervisor/OS layer. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> Reachability: a regular admin `HASS_TOKEN` is enough — Core proxies Supervisor
> calls via the `supervisor/api` WebSocket command. A separate
> `HASS_SUPERVISOR_TOKEN` is only needed for direct access when Core is down.

## Add-on lifecycle (via Core services — simplest)

```bash
hass-cli service call hassio.addon_start   --data '{"addon":"core_mosquitto"}'
hass-cli service call hassio.addon_restart --data '{"addon":"core_mosquitto"}'
hass-cli service call hassio.addon_stop    --data '{"addon":"core_mosquitto"}'
hass-cli service call hassio.backup_full
```

## Full management (via Supervisor proxy)

```bash
# List installed add-ons
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons","method":"get"}'

# Add-on info / stats / logs
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons/core_mosquitto/info","method":"get"}'
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons/core_mosquitto/stats","method":"get"}'

# Install / uninstall / update / set options
hass-cli raw ws supervisor/api --data '{"endpoint":"/store/addons/core_mosquitto/install","method":"post"}'
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons/core_mosquitto/options","method":"post","data":{"options":{}}}'

# Browse store / Core / Supervisor / OS info
hass-cli raw ws supervisor/api --data '{"endpoint":"/store","method":"get"}'
hass-cli raw ws supervisor/api --data '{"endpoint":"/core/info","method":"get"}'
hass-cli raw ws supervisor/api --data '{"endpoint":"/supervisor/info","method":"get"}'
```

## Notes

- Add-ons also surface as entities (switch to start/stop, sensors for cpu/mem,
  `update.*` for available updates) — see `ha-states` / `update` entities.
- Typed `hass-cli ha addons ...` commands are planned (P2); until then use the
  proxy calls above.
