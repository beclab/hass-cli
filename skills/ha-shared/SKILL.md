---
name: ha-shared
version: 0.1.0
description: "Shared foundation for all hass-cli skills. Read this FIRST. Covers connecting to a Home Assistant instance (HASS_SERVER + HASS_TOKEN long-lived token), the unified REST+WebSocket transport facade, output formats (table/json/yaml/ndjson), the raw passthrough escape hatches (raw api / raw ws), and the auth-error recovery table. Use whenever the user mentions Home Assistant, hass-cli, controlling smart-home entities, calling services, automations, or any HA REST/WebSocket API task."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli --help"
---

# ha-shared (foundation)

**Read this before any other `ha-*` skill.** It defines how to connect and the conventions every command follows.

> Source of truth for flags is always `hass-cli <command> --help`. This file carries the connection model, transport routing, and error recovery that `--help` cannot.

## Connection

hass-cli reaches a Home Assistant instance two ways under one facade:

- **REST API** (`/api/*`) — most reads/writes.
- **WebSocket API** (`/api/websocket`) — registries, subscriptions, and the Supervisor proxy.

Configure once via environment (preferred):

```bash
export HASS_SERVER="http://homeassistant.local:8123"
export HASS_TOKEN="<long-lived access token>"
```

Get a token: HA profile page -> "Long-Lived Access Tokens" -> Create Token.

Override per call with `--server` / `--token`, or use `--profile <name>` to select a profile from `~/.config/hass-cli/config.yaml`.

## Output

`-o, --output` selects `table` (default), `json`, `yaml`, or `ndjson`. For agent
consumption prefer `--output json`. Table shaping: `--columns`, `--sort-by`,
`--no-headers`.

## Verb map (route to the right skill)

| Intent | Command | Skill |
|---|---|---|
| Connectivity check | `hass-cli ping` | ha-shared |
| Read entity states | `hass-cli state list/get` | ha-states |
| Call a service | `hass-cli service call <domain.service>` | ha-services |
| Areas/devices/entities | `hass-cli registry <kind> ...` | ha-registry |
| Automations/scripts/scenes | `hass-cli workflow <domain> ...` | ha-automation |
| System health / repairs / logs | `hass-cli raw ...` / `system` | ha-system |
| Add-ons (via Core proxy) | `hass-cli raw ws supervisor/api ...` | ha-supervisor |

## Raw passthrough (full coverage)

Anything not yet wrapped by a typed command is still reachable:

```bash
hass-cli raw api GET states/sun.sun
hass-cli raw ws get_config
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons","method":"get"}'
```

## Auth-error recovery

| Symptom | Cause | Fix |
|---|---|---|
| `no server configured` / `no token configured` | env/flags unset | set `HASS_SERVER` + `HASS_TOKEN` |
| `HTTP 401` | bad/expired token | regenerate long-lived token |
| `authentication failed: auth_invalid` (WS) | token rejected on WS | same token as REST; regenerate |
| `HTTP 404` on `config/automation/config/...` | `config` integration disabled | enable default_config / config integration |
| TLS errors with self-signed cert | cert not trusted | add `--insecure` (dev only) |
