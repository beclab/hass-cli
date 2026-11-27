# Debugging an automation

Read when an automation did not fire, fired at the wrong time, or did the wrong thing. Back to [`ha-automation`](../SKILL.md).

Start from what the automation last did, not from its config:

```bash
hass-cli state get automation.sunset_lights -o json          # state (on = enabled), attributes.last_triggered, attributes.id
hass-cli system logbook --entity automation.sunset_lights    # when it fired and what it changed
hass-cli raw ws trace/list --data '{"domain":"automation","item_id":"<attributes.id>"}'
hass-cli raw ws trace/get --data '{"domain":"automation","item_id":"<attributes.id>","run_id":"<run_id from trace/list>"}'
```

| What the evidence shows | Likely cause |
|---|---|
| state `off` | the automation is disabled: `service call automation.turn_on --entity automation.x` |
| `last_triggered` empty or old, no traces | the trigger never matched: compare its entity, `to`/`from` state and time with the history in [`ha-states`](../../ha-states/SKILL.md) |
| a trace stops at a condition | the condition was false at the time; the trace records the values it saw |
| a trace stops at an action with an error | the action's entity or field is wrong; check it with `service describe` |
| traces pile up, `mode: single` warnings in the log | it re-triggers while still running; `mode: restart` or `queued` |
| ran, but the device changed back | another script or automation drives the device; the front door's [check after a call](../../ha-shared/SKILL.md#did-it-work) |

Traces are kept for the last few runs only. To test without waiting for the trigger, `workflow automation trigger automation.x` runs the actions and skips the conditions; it does not prove the trigger matches.
