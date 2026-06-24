# hass-cli

A command-line interface for [Home Assistant](https://www.home-assistant.io/),
talking to a local or remote instance over its REST and WebSocket APIs.

It mirrors how the Home Assistant frontend calls Core, exposing typed commands
for the common surfaces plus a `raw` passthrough for everything else, and agent
**skills** that teach an AI assistant how to drive them.

Version: `0.1.0`.

## Install

```bash
go install github.com/bytetrade/hass-cli@latest
# or from source
go build -o hass-cli .
```

## Configure

Create a long-lived access token on your HA profile page
("Long-Lived Access Tokens" -> Create Token), then:

```bash
export HASS_SERVER="http://homeassistant.local:8123"
export HASS_TOKEN="<token>"
```

Or pass `--server` / `--token` per call, or use `--profile <name>` with
`~/.config/hass-cli/config.yaml`:

```yaml
default: home
profiles:
  home:
    server: http://homeassistant.local:8123
    token: <token>
    insecure: false
    timeout: 10
```

Precedence is profile < environment < flags. `HASS_SERVER` must include a
scheme (`http://` or `https://`).

## Global flags

| Flag | Description |
|------|-------------|
| `-s, --server` | HA URL (env `HASS_SERVER`) |
| `--token` | Long-lived token (env `HASS_TOKEN`) |
| `--profile` | Named profile from `config.yaml` |
| `--insecure` | Skip TLS certificate verification |
| `--timeout` | Request timeout in seconds (default 10); also bounds WebSocket calls |
| `-o, --output` | `json` \| `yaml` \| `table` \| `ndjson` (table is default) |
| `--columns` | Table columns, e.g. `ENTITY=entity_id,STATE=state` |
| `--sort-by` | Sort table rows by a gjson path |
| `--no-headers` | Do not print table headers |

List commands set sensible default table columns; pass `--columns` to override.

Any `--data` flag also accepts `@file.json` to read JSON from a file. On Windows
PowerShell prefer `@file.json`, since inline JSON quoting is unreliable there.
Commands that take YAML/JSON documents (`workflow save`, `lovelace config save`,
`energy prefs save`, `template`) accept `--file`.

## Commands

### Core

```bash
hass-cli ping                              # connectivity + auth check
hass-cli config get                        # instance configuration
hass-cli state list                        # all entity states
hass-cli state get sun.sun
hass-cli state set sensor.foo --state 42 --attributes '{"unit_of_measurement":"C"}'
hass-cli service list
hass-cli service describe light.turn_on
hass-cli service call light.turn_on --arguments entity_id=light.kitchen
hass-cli event fire my_event --data '{"foo":"bar"}'
hass-cli event watch state_changed         # stream events (Ctrl-C to stop)
hass-cli template '{{ states("sun.sun") }}'
```

### Registry

Areas, devices, entities, floors, labels, and categories. The `<id>` is
positional on update/delete.

```bash
hass-cli registry area list                # area/device/entity/floor/label
hass-cli registry area create --data '{"name":"Garage"}'
hass-cli registry area update garage --data '{"name":"Garage Bay"}'
hass-cli registry area delete garage
hass-cli registry category list --scope automation
```

### Helpers

Nine helper types (`input_boolean`, `input_number`, `input_text`,
`input_select`, `input_button`, `input_datetime`, `counter`, `timer`,
`schedule`), each with `list`/`create`/`update`/`delete`.

```bash
hass-cli helper input_boolean list
hass-cli helper input_boolean create --data '{"name":"Guest Mode"}'
hass-cli helper counter update visits --data '{"step":2}'
hass-cli helper counter delete visits
```

### Workflows

Automations, scripts, and scenes.

```bash
hass-cli workflow automation list
hass-cli workflow automation save my_auto --file my_auto.yaml
hass-cli workflow automation trigger my_auto
hass-cli workflow script reload
hass-cli workflow scene get scene.movie_time
```

### Integrations

Config entries and the data-entry config flow used to add an integration.

```bash
hass-cli integration list
hass-cli integration get <entry_id>
hass-cli integration reload <entry_id>
hass-cli integration update <entry_id> --data '{"title":"New name"}'
hass-cli integration flow handlers         # which integrations can be added
hass-cli integration flow start hue
hass-cli integration flow get <flow_id>
hass-cli integration flow step <flow_id> --data '{...}'
```

### Backups

```bash
hass-cli backup list
hass-cli backup agents
hass-cli backup create --auto              # or --data '{"agent_ids":[...]}'
hass-cli backup get <slug>
hass-cli backup restore <slug> --data '{"agent_id":"backup.local"}'
hass-cli backup delete <slug>
```

### Lovelace

Dashboards, their stored configs, and custom resources (WebSocket-only).

```bash
hass-cli lovelace dashboard list
hass-cli lovelace dashboard create --data '{"url_path":"ops-room","title":"Ops"}'
hass-cli lovelace config get --dashboard ops-room
hass-cli lovelace config save --dashboard ops-room --file dash.yaml
hass-cli lovelace resource list
```

### Assist

Voice pipelines (conversation/STT/TTS chains). To run text through a pipeline,
prefer `service call conversation.process`.

```bash
hass-cli assist pipeline list
hass-cli assist pipeline get               # preferred pipeline unless --id
hass-cli assist languages
hass-cli assist devices
```

### Energy & statistics

```bash
hass-cli energy prefs get
hass-cli energy prefs save --file energy.yaml      # or --data '{...}'
hass-cli energy validate
hass-cli statistics list
hass-cli statistics metadata --ids sensor.grid_consumption
hass-cli statistics period --ids sensor.grid_consumption --start 2026-01-01T00:00:00+00:00
```

### System

```bash
hass-cli system health                     # integration system-health report
hass-cli system repairs
hass-cli system errorlog
hass-cli system logbook
hass-cli system history --start 2026-01-01T00:00:00+00:00 --entities sun.sun
hass-cli system hardware
hass-cli system analytics                  # get / set opt-in preferences
hass-cli system labs                       # experimental preview features
```

### Add-ons & Supervisor

Only available on HA OS / Supervised installs. On a Core (Container) install
these fail fast with a clear message. A regular admin `HASS_TOKEN` is enough
(calls are proxied through Core's `supervisor/api`).

```bash
hass-cli addon list
hass-cli addon info core_mosquitto
hass-cli addon start|stop|restart core_mosquitto
hass-cli addon logs core_mosquitto
hass-cli supervisor info|stats|host|os|core
```

### Raw passthrough (full coverage)

Anything not yet wrapped by a typed command:

```bash
hass-cli raw api GET states/sun.sun
hass-cli raw ws get_config
# install/uninstall/store browse have no typed command yet:
hass-cli raw ws supervisor/api --data '{"endpoint":"/store/addons/core_mosquitto/install","method":"post"}'
```

## Skills

Agent skills are bundled into the binary. `skill list` prints each skill's name
and description; `skill show` prints the full guide.

```bash
hass-cli skill list
hass-cli skill show ha-shared
```

## Development

```bash
go build ./...
go test ./...    # runs an in-process mock HA (REST + WebSocket)
# full integration smoke test against a Dockerized HA:
pwsh scripts/local-ha.ps1
```

See [AGENTS.md](AGENTS.md) for repository working rules.
