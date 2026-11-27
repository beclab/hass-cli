---
name: ha-lighting
version: "0.0.0"
description: "Light colour, colour temperature, brightness, effects and transitions; fan speed, oscillation and presets; whole rooms of lights at once. Plain on/off is the front door's two commands."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-lighting

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

What lights, switches and fans take beyond on and off. A light that should keep changing (a colour loop, a fade every few minutes) is a script: [`ha-automation`](../ha-automation/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| A colour, a warmth, a brightness | this file | one `light.turn_on` with the field |
| Every light in a room or on a floor | this file | one call with `area_id` or `floor_id` |
| Fan speed, oscillation, a preset | this file | `fan.set_percentage`, `fan.oscillate`, `fan.set_preset_mode` |
| The light to cycle colours or keep changing | [`ha-automation`](../ha-automation/SKILL.md) | a script |

## Lights

Every setting is one `light.turn_on`, which also switches the light on:

```bash
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","color_name":"yellow"}'
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","rgb_color":[255,120,0]}'
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","color_temp_kelvin":2700}'
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","brightness_pct":30,"transition":3}'
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","effect":"colorloop"}'
```

| Field | Values |
|---|---|
| `brightness_pct` | 0 to 100; `brightness_step_pct: -20` dims relative to now |
| `color_name` | CSS names: `red`, `yellow`, `warmwhite`, ... |
| `rgb_color`, `hs_color` | `[r, g, b]` 0 to 255; `[hue 0-360, saturation 0-100]` |
| `color_temp_kelvin` | about 2700 warm to 6500 cool |
| `transition` | seconds to fade over |
| `effect` | one of the light's own `effect_list` |

What a light takes is in its attributes: `hass-cli state get light.kitchen -o json`, then `supported_color_modes`, `effect_list`, `min_color_temp_kelvin` / `max_color_temp_kelvin`.

- A light whose modes include `hs`, `xy`, `rgb`, `rgbw` or `rgbww` takes any colour field; Home Assistant converts it to the light's own mode.
- A light with only `color_temp` cannot take a colour: offer the nearest warmth instead.
- A light with only `onoff` takes no brightness; `brightness` mode takes brightness but no colour.
- `effect` names are the device's own; use one from `effect_list` as written.

## Switches

A `switch.*` is on or off: `turn_on`, `turn_off`, `toggle`. A plug that powers a lamp is a switch, so a "lamp" that does not show up under `--domain light` is often one; search without `--domain`.

## Fans

```bash
hass-cli service call fan.set_percentage --data '{"entity_id":"fan.bedroom","percentage":66}'
hass-cli service call fan.oscillate --data '{"entity_id":"fan.bedroom","oscillating":true}'
hass-cli service call fan.set_preset_mode --data '{"entity_id":"fan.bedroom","preset_mode":"sleep"}'
hass-cli service call fan.set_direction --data '{"entity_id":"fan.bedroom","direction":"reverse"}'
```

Presets come from the fan's `preset_modes` attribute. "Faster" or "slower" is `fan.increase_speed` / `fan.decrease_speed`.

## Many at once

A target names an area, floor or label instead of entities; the ids come from [`ha-registry`](../ha-registry/SKILL.md):

```bash
hass-cli service call light.turn_off --data '{"area_id":"living_room"}'
hass-cli service call light.turn_on --data '{"floor_id":"upstairs","brightness_pct":50}'
```

## Verb index

| Service | For | Read when triggered |
|---|---|---|
| `light.turn_on` | every light setting | the field table above |
| `light.turn_off`, `light.toggle` | on and off; `transition` fades | none |
| `switch.*` | `turn_on`, `turn_off`, `toggle` | none |
| `fan.*` | speed, oscillation, direction, presets | the fans section above |
| Anything else on these domains | | `hass-cli service describe <domain.service>` lists its fields; [`ha-services`](../ha-services/SKILL.md) |

## Safety and escalation

- "All the lights" means lights. Switches can be heaters, pumps or a server's power: include a switch only when the user named it or it is clearly a light.
- An area-wide `turn_off` at night can leave someone in the dark: when the request was one light, call one light.
- A light that turns back on or keeps changing after the call is driven by a script or automation; the call prints a `Note:` naming it, and [the front door](../ha-shared/SKILL.md#did-it-work) says what to do.
