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

Get a token: HA profile page -> "Long-Lived Access Tokens" -> Create Token.

**Preferred: save a profile.** A profile bundles a server URL with its token;
the token is stored in the OS keychain (not in a plaintext file), and the index
(`profiles.json`, no secrets) lives under the config dir.

```bash
# interactive on a terminal: prompts for URL + token (hidden), validates, saves
hass-cli profile login home
# non-interactive (token via stdin):
printf '%s' "$TOKEN" | hass-cli profile login home --server http://homeassistant.local:8123 --token-stdin
```

Manage profiles with `hass-cli profile list/use/show/remove`. Select one per
call with `--profile <name>`; the default is the current profile.

**Alternative: environment variables** (no keychain, good for CI):

```bash
export HASS_SERVER="http://homeassistant.local:8123"
export HASS_TOKEN="<long-lived access token>"
```

Precedence (low → high): profile (`profiles.json` + keychain) → legacy
`config.yaml` → environment → explicit `--server` / `--token` flags.

## Output

`-o, --output` selects `table` (default), `json`, `yaml`, or `ndjson`. For agent
consumption prefer `--output json`. Table shaping: `--columns`, `--sort-by`,
`--no-headers`.

## Verb map (route to the right skill)

| Intent | Command | Skill |
|---|---|---|
| First-run setup / manage profiles | `hass-cli profile login/list/use/show/remove` | ha-shared |
| Connectivity check | `hass-cli ping` | ha-shared |
| Read entity states | `hass-cli state list/get` | ha-states |
| Call a service | `hass-cli service call <domain.service>` | ha-services |
| Lights / switches / fans | `hass-cli service call light.* ...` | ha-lighting |
| Climate / thermostat / humidity | `hass-cli service call climate.* ...` | ha-climate |
| Covers / locks / valves | `hass-cli service call cover.*/lock.* ...` | ha-openings |
| Media players | `hass-cli service call media_player.* ...` | ha-media |
| Notifications / TTS | `hass-cli service call notify.*/tts.* ...` | ha-notify |
| People / zones / presence | `hass-cli state list/get person.* ...` | ha-presence |
| Helpers (input_*, counter, timer) | `hass-cli helper <type> ...` | ha-helpers |
| Sensors/number/select/vacuum/alarm/todo/calendar | `hass-cli state get` / `service call` | ha-entities |
| Energy dashboard / consumption | `hass-cli energy ...` | ha-energy |
| Long-term statistics / trends | `hass-cli statistics ...` | ha-statistics |
| Voice / conversation / Assist pipelines | `hass-cli assist ...` | ha-assist |
| Dashboards / cards / resources | `hass-cli lovelace ...` | ha-lovelace |
| Areas/devices/entities/floors/labels/categories | `hass-cli registry <kind> ...` | ha-registry |
| Automations/scripts/scenes | `hass-cli workflow <domain> ...` | ha-automation |
| Build a new automation from a request (playbook) | `hass-cli workflow ...` | ha-workflow-automation-builder |
| Audit for dead entities / broken/unused automations (playbook) | `hass-cli system ...` + `state list` | ha-workflow-audit |
| Integrations / config entries / discovery | `hass-cli integration ...` | ha-integrations |
| Zigbee/Z-Wave/Matter/Thread gateways & networks | `hass-cli service call zha.* ...` / `hass-cli raw ws zha\|zwave_js\|matter\|thread/...` | ha-gateway |
| Backups (create/restore/delete) | `hass-cli backup ...` | ha-backup |
| System health / repairs / logs / hardware / analytics / labs | `hass-cli system ...` | ha-system |
| Add-ons / Supervisor (HA OS only) | `hass-cli addon ...` / `hass-cli supervisor ...` | ha-supervisor |

## Raw passthrough (full coverage)

Anything not yet wrapped by a typed command is still reachable:

```bash
hass-cli raw api GET states/sun.sun
hass-cli raw ws get_config
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons","method":"get"}'
```

REST paths are relative to `/api`, but a leading `/api/` (or `api/`) is tolerated, so values copied from docs still resolve.

## Passing JSON with `--data` (avoid shell quoting traps)

Every `--data` flag accepts either inline JSON or `@path` to read JSON from a
file. On Windows PowerShell, inline JSON quoting is fragile (PowerShell strips
inner `"`), so **prefer `@file.json`** there:

```bash
# portable: read JSON from a file
hass-cli registry area create --data @area.json

# inline (bash/zsh): single-quote the whole object
hass-cli registry area create --data '{"name":"Garage"}'

# inline (PowerShell): escape inner quotes with backslash
hass-cli registry area create --data '{\"name\":\"Garage\"}'
```

## Auth-error recovery

| Symptom | Cause | Fix |
|---|---|---|
| `no server configured` / `no token configured` | no profile, env, or flags | run `hass-cli profile login`, or set `HASS_SERVER` + `HASS_TOKEN` |
| `HTTP 401` | bad/expired token | regenerate the long-lived token, then `hass-cli profile login <name> --force` |
| `profile "x" already has a valid token` | re-login to an active profile | pass `--force`, or `hass-cli profile remove x` first |
| `authentication failed: auth_invalid` (WS) | token rejected on WS | same token as REST; regenerate |
| `HTTP 404` on `config/automation/config/...` | `config` integration disabled | enable default_config / config integration |
| TLS errors with self-signed cert | cert not trusted | add `--insecure` (dev only) |
