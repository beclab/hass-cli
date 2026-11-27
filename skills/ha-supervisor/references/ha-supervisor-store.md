# Add-on store: installing, removing, configuring

Read when installing, uninstalling or changing the options of an add-on. Back to [`ha-supervisor`](../SKILL.md).

These have no typed command. They go through the Supervisor's own API, which Home Assistant proxies over WebSocket: `endpoint` is the Supervisor path, `method` is `get` or `post`, and `data` is the body.

## Finding an add-on in the store

```bash
hass-cli raw ws supervisor/api --data '{"endpoint":"/store","method":"get"}' -o json
```

The answer lists `repositories` and `addons`, each add-on with its `slug`, `name`, `installed` and `available` (false when it does not run on this machine's architecture). Match the user's words against `name`; take the `slug` from there.

A community add-on not in the list lives in a repository the store does not know yet. Adding a repository is a trust decision about third-party code: ask the user for its URL rather than searching for one.

## Installing and removing

```bash
hass-cli raw ws supervisor/api --data '{"endpoint":"/store/addons/core_mosquitto/install","method":"post"}' --timeout 600
hass-cli addon start core_mosquitto
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons/core_mosquitto/uninstall","method":"post"}'
```

- An install downloads an image and can take minutes; raise `--timeout`. If it times out, `addon info <slug>` shows whether it finished before trying again.
- A new add-on is installed stopped. Check its options (below) before the first start; many need at least a password or a device path.
- Uninstalling deletes the add-on's data. Integrations that used it (MQTT, Z-Wave JS) go to `setup_retry` until something replaces it.

## Options

```bash
hass-cli addon info core_mosquitto -o json      # options: current values; schema: what each takes
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons/core_mosquitto/options","method":"post","data":{"options":{"logins":[],"require_certificate":false}}}'
hass-cli addon restart core_mosquitto
```

- `options` replaces the whole set: start from the current `options` of `addon info`, change the one key the user asked about, send all of it back. A key left out falls back to its default.
- Options take effect on the next start: restart the add-on afterwards, and say so.
- Passwords in options come from the user. Do not print the current options back if they contain one.
