# Connection, auth and input

Read when a command cannot connect or authenticate, when setting hass-cli up, or when `--data` quoting fails. Back to [the front door](../SKILL.md).

## Where the server and token come from

From lowest to highest precedence: the profile (`profiles.json` plus the OS keychain, or a legacy `config.yaml` when the index has no such profile), the environment (`HASS_SERVER`, `HASS_TOKEN`, `HASS_INSECURE=1`), then the `--server` / `--token` / `--insecure` flags. A higher source wins field by field.

A token is a long-lived access token: Home Assistant profile page, Security, Long-Lived Access Tokens, Create Token.

**A profile** keeps the token in the OS keychain, not in a file:

```bash
hass-cli profile login home --server http://homeassistant.local:8123   # prompts for the token, hidden
printf '%s' "$TOKEN" | hass-cli profile login home --server http://homeassistant.local:8123 --token-stdin
hass-cli profile list
hass-cli profile use home        # `-` switches back
```

`--profile <name>` selects one for a single call. `profile login` validates the token before it saves anything.

**Environment variables** suit CI and containers, where there is no keychain:

```bash
export HASS_SERVER="http://homeassistant.local:8123"
export HASS_TOKEN="<long-lived access token>"
```

`hass-cli ping` checks both the connection and the token.

## Auth and connection errors

| Symptom | Cause | Fix |
|---|---|---|
| `no server configured` / `no token configured` | no profile, env or flag | `hass-cli profile login`, or set `HASS_SERVER` and `HASS_TOKEN` |
| `HTTP 401` | the token is wrong, expired or revoked | create a new token, then `hass-cli profile login <name> --force` |
| `profile "x" already has a valid token` | re-login over a working profile | `--force`, or `hass-cli profile remove x` first |
| `authentication failed: auth_invalid` (WebSocket) | the token was rejected on the WebSocket | the same token serves both; replace it as for `HTTP 401` |
| `HTTP 404` on `config/automation/config/...` or `config/script/config/...` | the `config` integration is not loaded | enable `default_config` or `config:` in `configuration.yaml` |
| certificate errors on a self-signed server | the certificate is not trusted | `--insecure`, or `profile login --insecure` for that profile; only on a network you control |
| timeouts on a slow call | the default is 10 s | `--timeout 60` (backups and restores take longer) |

## Passing `--data`

Every `--data` takes inline JSON or `@path`. `workflow ... save --data` also reads YAML from a file.

```bash
hass-cli registry area create --data '{"name":"Garage"}'          # bash, zsh: single-quote the object
hass-cli registry area create --data @area.json                     # any shell
hass-cli registry area create --data '{\"name\":\"Garage\"}'        # PowerShell strips bare inner quotes
```

On PowerShell prefer `@file`.

## Output

`-o` selects `table` (the default), `json`, `yaml` or `ndjson`. Read `-o json` when a value feeds the next command; tables are for showing the user. `--columns NAME=path,...`, `--sort-by <path>` and `--no-headers` shape a table, with paths in gjson syntax (`attributes.friendly_name`).
