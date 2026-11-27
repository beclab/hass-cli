# Raw REST and WebSocket access

Read when no typed command covers the request. Back to [the front door](../SKILL.md).

hass-cli talks to Home Assistant over two APIs. Most reads and writes are REST (`/api/*`); registries, subscriptions and the Supervisor proxy are WebSocket (`/api/websocket`). `raw` reaches either directly, with the same connection and token as every other command.

```bash
hass-cli raw api GET states/sun.sun
hass-cli raw api POST services/light/turn_on --data '{"entity_id":"light.kitchen"}'
hass-cli raw ws get_config
hass-cli raw ws supervisor/api --data '{"endpoint":"/addons","method":"get"}'
```

- A REST path is relative to `/api`; a leading `/api/` or `api/` is tolerated, so a path copied from Home Assistant's documentation resolves.
- `raw ws <type>` sends `{"type": "<type>", ...}` with the fields of `--data` merged in; the id is added for you.
- The answer is printed as Home Assistant returned it. Read it with `-o json`.

Prefer a typed command when one exists: it checks targets and fields before sending, and its errors name the next step. `raw` sends what it is given.

Writes through `raw` follow [the front door's safety rules](../SKILL.md#safety-and-escalation) like any other change.
