---
name: ha-supervisor
version: "0.0.0"
description: "Add-ons and the Supervisor on Home Assistant OS or Supervised installs: start, stop, restart or read the logs of an add-on (Mosquitto, Z-Wave JS, Node-RED), install or remove one, update Home Assistant, an add-on or the OS, reboot the host."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-supervisor

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

On Home Assistant OS and Supervised installs, the Supervisor runs Home Assistant Core, the add-ons (separate containers such as the Mosquitto broker or the Z-Wave JS server) and the host. A Container or Core install has none of this: every command here answers that the instance has no Supervisor, and the user manages those services outside Home Assistant. Add-ons are not integrations: the Mosquitto add-on is the broker, the MQTT integration in [`ha-integrations`](../ha-integrations/SKILL.md) is Home Assistant's connection to it.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Restart, stop or start an add-on | this file | `addon restart <slug>` |
| Why an add-on is not working | this file | `addon info`, then `addon logs <slug>` |
| What can be updated, update Home Assistant or an add-on | this file | `service call update.install` |
| Install, remove or configure an add-on | [add-on store](references/ha-supervisor-store.md) | `raw ws supervisor/api` |
| Reboot or shut down the machine | this file | `service call hassio.host_reboot`, on an explicit request |

## Add-ons

```bash
hass-cli addon list -o json             # slug, name, state, version, update_available
hass-cli addon info core_mosquitto -o json
hass-cli addon logs core_mosquitto | tail -n 50
hass-cli addon restart core_mosquitto
```

- A slug is the add-on's id, not its name: official ones are `core_<name>` (`core_mosquitto`), community ones carry a repository prefix (`a0d7b954_nodered`). Take it from `addon list`; never build it from the name.
- `state` is `started` or `stopped` (`error` when it crashed). A stopped add-on with `boot: auto` crashed or was stopped by hand; its logs say which.
- An integration in `setup_retry` often means its add-on is down (MQTT and `core_mosquitto`, Z-Wave and `core_zwave_js`). Restart the add-on, then `integration reload` the entry.
- The admin token is enough: the CLI reaches the Supervisor through Home Assistant. A refusal naming admin rights means the token belongs to a non-admin user; the user has to create a long-lived token as an administrator and switch to it ([connection](../ha-shared/references/ha-shared-connection.md)).

## Updates

Every pending update is an `update.*` entity: Home Assistant Core, the OS, the Supervisor and each add-on.

```bash
hass-cli state list --domain update           # state on = an update is waiting
hass-cli supervisor core -o json              # version and version_latest of Core; also supervisor os, info
hass-cli service call update.install --data '{"entity_id":"update.mosquitto_broker_update","backup":true}'
```

- `backup: true` takes a backup first, so the update can be undone. Use it for Core and for add-ons that hold data.
- A Core or OS update restarts Home Assistant or the machine; the connection drops and comes back several minutes later. That is expected, not a failure.
- Read the version change back from the entity's `installed_version` and `latest_version` before installing, so the user knows what they agreed to.

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `addon list`, `addon info` | slug for `info` | slugs, state, versions, options |
| `addon start`, `stop`, `restart` | slug | the add-on section above |
| `addon logs` | slug | the add-on's own log; Core's log is `system errorlog` in [`ha-system`](../ha-system/SKILL.md) |
| `supervisor info`, `stats`, `host`, `os`, `core` | nothing | versions, update channel, disk and memory of the host |
| `service call update.install` | an `update.*` entity | the updates section above |
| `service call hassio.host_reboot`, `hassio.host_shutdown` | nothing | only on an explicit request |
| `raw ws supervisor/api` | an endpoint and method | [add-on store](references/ha-supervisor-store.md) |

Backups on these installs are the same native backups as everywhere: [`ha-backup`](../ha-backup/SKILL.md).

## Safety and escalation

- Stopping an add-on stops what depends on it: stopping Mosquitto takes every MQTT device offline, stopping Z-Wave JS every Z-Wave device. Say that before stopping one the user did not name as the problem.
- Uninstalling an add-on deletes its data and settings. Only on an explicit request naming it, and offer a backup first.
- Updates, reboots and shutdowns only on the user's request. A shutdown needs someone at the machine to bring it back: say so and ask once more before sending it.
- A long operation (an install, an update) that times out on the CLI is usually still running. Check `addon info` or the `update.*` entity before sending it again.
