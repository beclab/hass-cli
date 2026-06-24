# hass-cli

A command-line interface for [Home Assistant](https://www.home-assistant.io/),
talking to a local or remote instance over its REST and WebSocket APIs.

It mirrors how the Home Assistant frontend calls Core, exposing a small set of
generic verbs plus agent **skills** that teach an AI assistant how to drive
them.

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
```

## Usage

```bash
hass-cli ping                              # connectivity + auth check
hass-cli config get                        # instance configuration
hass-cli state list                        # all entity states
hass-cli state get sun.sun
hass-cli service list
hass-cli service call light.turn_on --arguments entity_id=light.kitchen
hass-cli event fire my_event --data '{"foo":"bar"}'
hass-cli event watch state_changed         # stream events (Ctrl-C to stop)
hass-cli template '{{ states("sun.sun") }}'
hass-cli registry area list                # area/device/entity/floor/label
hass-cli registry area create --data '{"name":"Garage"}'
hass-cli registry area update garage --data '{"name":"Garage Bay"}'  # id is positional
hass-cli registry area delete garage
hass-cli helper input_boolean list         # input_*/counter/timer/schedule
hass-cli helper input_boolean create --data '{"name":"Guest Mode"}'
hass-cli helper counter delete visits
hass-cli workflow automation list
hass-cli workflow automation save my_auto --file my_auto.yaml
hass-cli system health                     # integration system-health report
```

Output format: `-o json|yaml|table|ndjson` (table is default). Shape tables with
`--columns ENTITY=entity_id,STATE=state` and `--sort-by`.

Any `--data` flag also accepts `@file.json` to read JSON from a file. On Windows
PowerShell prefer `@file.json`, since inline JSON quoting is unreliable there.

### Raw passthrough (full coverage)

Anything not yet wrapped by a typed command:

```bash
hass-cli raw api GET states/sun.sun
hass-cli raw ws get_config
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons","method":"get"}'
```

## Skills

Agent skills are bundled into the binary:

```bash
hass-cli skill list
hass-cli skill show ha-shared
```

## Development

```bash
go build ./...
go test ./...    # runs an in-process mock HA (REST + WebSocket)
```

See [AGENTS.md](AGENTS.md) for repository working rules.
