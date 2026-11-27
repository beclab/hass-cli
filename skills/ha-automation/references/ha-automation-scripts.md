# Writing a script

Read when making a device loop or repeat, or when changing an existing script or automation. Back to [`ha-automation`](../SKILL.md).

## A loop that stops itself

"Make the kitchen light cycle colours" is one script, saved once and run. Write it to a file and save it with a bare English id:

```yaml
# color_cycle.yaml
alias: 厨房灯循环变色
mode: restart
sequence:
  - action: light.turn_on
    target:
      entity_id: light.kitchen
  - repeat:
      while:
        - condition: state
          entity_id: light.kitchen
          state: "on"
      sequence:
        - action: light.turn_on
          target:
            entity_id: light.kitchen
          data:
            hs_color: ["{{ range(360) | random }}", 100]
            transition: 1
        - delay:
            seconds: 3
```

```bash
hass-cli workflow script save color_cycle --data @color_cycle.yaml
hass-cli workflow script trigger script.color_cycle
```

- The `while` condition is checked before each step, so turning the light off during the delay ends the loop at its next check: an ordinary "turn the light off" works. A loop with `count` or `until` keeps running until it ends or is stopped with `workflow script stop`.
- `mode: restart` makes running it again start over instead of queueing a second loop.
- A delay of 1 second or more. Shorter steps flood the device and Home Assistant's log.
- Keep the `hs_color` line as written for any colour light, including one whose `supported_color_modes` lists only `xy`: Home Assistant converts it. Do not write `xy_color` by hand; a random `xy` pair is mostly outside the light's gamut, and `range(0, 1)` is always 0. A light with only `color_temp` or `brightness` takes no colour; [`ha-lighting`](../../ha-lighting/SKILL.md) covers the modes.
- `action:` is the service key. Older configs say `service:`; both work, do not mix them in one file.

To loop a fixed number of times, replace `while` with `count: 10`. To loop through a list, `for_each: [...]` with `{{ repeat.item }}` in the step.

## Script config fields

| Field | Meaning |
|---|---|
| `alias` | the display name; any language |
| `sequence` | the steps, in order |
| `mode` | `single` (the default; a second run is refused while one runs), `restart`, `queued`, `parallel` |
| `fields` | parameters a caller passes in `service call script.<id> --data '{...}'` |
| `icon`, `description` | shown in the UI |

The id is not in the config: it is the argument to `save`.

## Changing an existing script or automation

```bash
hass-cli workflow script get color_cycle -o yaml > color_cycle.yaml
# edit the file
hass-cli workflow script save color_cycle --data @color_cycle.yaml
```

For an automation, the config id is `attributes.id` from `state get automation.<name> -o json`.

An automation made in the UI is stored with plural keys (`triggers`, `conditions`, `actions`) and a 32-character hex id; one from `automations.yaml` may use the singular keys and any id. Keep whichever form `get` returned. An automation that `get` cannot find (`HTTP 404` while its entity exists) lives in a YAML file the config API does not manage: it has to be edited there, then `workflow automation reload`.
