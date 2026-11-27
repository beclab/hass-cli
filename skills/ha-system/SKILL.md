---
name: ha-system
version: "0.0.0"
description: "Home Assistant itself: is anything wrong, repairs and warnings, the error log, system health, check the configuration, restart Home Assistant, hardware and USB dongles, analytics and labs preview features."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-system

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

The instance as a whole rather than one device: what Home Assistant already knows is wrong, what it logged, and restarting it. What one entity was doing earlier (`system history`, `system logbook`) is [`ha-states`](../ha-states/SKILL.md); one failing integration is [`ha-integrations`](../ha-integrations/SKILL.md); add-ons and the host OS are [`ha-supervisor`](../ha-supervisor/SKILL.md). A sweep for dead entities and unused automations is [`ha-workflow-audit`](../ha-workflow-audit/SKILL.md), which starts from the repairs here.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Is anything wrong", "any warnings" | this file | `system repairs`, then `integration list` |
| Why something errored, what the log says | this file | `system errorlog`, narrowed with `rg` |
| Restart Home Assistant | this file | check the config, then `service call homeassistant.restart` |
| Pick up a YAML change without restarting | this file | the matching reload service |
| Which board, Zigbee or Bluetooth dongle is plugged in | this file | `system hardware` |

## Something wrong

```bash
hass-cli system repairs -o json                      # issues Home Assistant raised itself
hass-cli integration list                            # entries not in state loaded
hass-cli system errorlog | rg -i 'error|warning' | tail -n 40
hass-cli system health -o json                       # per integration: reachable, version, quotas
```

- Repairs are the first answer: each issue names its `domain`, a `severity` (`warning`, `error`, `critical`), and `breaks_in_ha_version` when something will stop working after an upgrade. Report them in the user's words, not as raw ids.
- An issue with `is_fixable: true` has a guided fix in Settings > Repairs; there is no typed command for it. Tell the user it is there rather than editing config to make it disappear.
- `errorlog` is the plain log text since the last start, oldest first. Filter by the integration's domain (`rg -i zha`) or take the end; never paste the whole log back to the user.
- `errorlog` is Home Assistant Core's log only. An add-on (Mosquitto, Z-Wave JS, Zigbee2MQTT) logs on its own: `addon logs` in [`ha-supervisor`](../ha-supervisor/SKILL.md).

## Restart and reload

A restart takes Home Assistant down for a minute or more: every automation pauses and the voice assistant answering this request may drop. Prefer a reload when one covers the change:

| Changed | Instead of restarting |
|---|---|
| automations, scripts, scenes | `hass-cli workflow automation reload` (or `script`, `scene`) |
| one integration's entry | `hass-cli integration reload <entry_id>` |
| location, units, `customize` | `hass-cli service call homeassistant.reload_core_config` |
| several YAML files at once | `hass-cli service call homeassistant.reload_all` |

When a restart is what the user asked for, check the configuration first. The restart service refuses an invalid config, and the check says why:

```bash
hass-cli raw api POST config/core/check_config -o json    # result: valid | invalid, with errors
hass-cli service call homeassistant.restart
```

The connection drops while it restarts, so an error from the restart call itself is expected. Wait, then `hass-cli config get` answers again when it is back. `homeassistant.stop` shuts it down without coming back: never as a step towards anything else.

## Verb index

| Verb | Answers | Read when triggered |
|---|---|---|
| `system repairs` | open issues, severity, fixable | the section on something wrong |
| `system errorlog` | the log as text | filter it; do not page it all |
| `system health` | per-integration health (cloud reachable, versions) | a cloud or remote integration misbehaves |
| `system history`, `system logbook` | what entities did and when | [`ha-states`](../ha-states/SKILL.md) |
| `system hardware` | the board and USB radios | empty on a plain Container install; radios are [`ha-gateway`](../ha-gateway/SKILL.md) |
| `system analytics`, `analytics set` | what usage data is shared | only on the user's request |
| `system labs`, `labs update` | preview features, on or off | only on the user's request; name the feature back |
| `config get` | version, location, units, time zone | the version, before talking about upgrades |

`hardware`, `analytics` and `labs` come from integrations that load shortly after start; `unknown_command` right after a restart means wait and ask again.

## Safety and escalation

- Restart, stop and `reload_all` only on the user's explicit request. Say first that Home Assistant will be unavailable for a while; afterwards, say whether it came back.
- `check_config` reporting `invalid`: do not restart. Report the errors and stop; fixing YAML on the server is the user's call.
- Analytics and labs change what the instance shares or how it behaves: change only the preference the user named, and read back the new setting.
- The error log can contain addresses, user names and tokens in URLs. Quote the lines that matter, not the log.
