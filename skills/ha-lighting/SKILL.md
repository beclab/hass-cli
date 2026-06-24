---
name: ha-lighting
version: 0.1.0
description: "Control Home Assistant lights, switches, and fans with hass-cli: turn on/off, dim, set color or color temperature, set fan speed/oscillation. Use for any 'turn on the lights', 'dim to 30%', 'make it warm white', 'set fan to medium' style request. All control goes through service call."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-lighting

Lights, switches, fans. **Prerequisites:** [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md),
[`../ha-services/SKILL.md`](../ha-services/SKILL.md).

> No `hass-cli light` command — drive everything with `service call`. Find
> targets with `hass-cli state list -o json` (filter `entity_id` by domain).

## Lights (`light.*`)

```bash
hass-cli service call light.turn_on  --arguments entity_id=light.kitchen
hass-cli service call light.turn_off --arguments entity_id=light.kitchen
hass-cli service call light.toggle   --arguments entity_id=light.kitchen

# Brightness: brightness_pct 0-100, or brightness 0-255
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","brightness_pct":30}'

# Color temperature (warm/cool). Kelvin: ~2700 warm, ~6500 cool
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","color_temp_kelvin":2700}'

# RGB color
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","rgb_color":[255,120,0]}'

# Transition (seconds) and effect
hass-cli service call light.turn_on --data '{"entity_id":"light.kitchen","brightness_pct":80,"transition":3}'
```

Check what a light supports: `hass-cli state get light.kitchen -o json` →
`attributes.supported_color_modes`, `effect_list`, `min/max_color_temp_kelvin`.

## Switches (`switch.*`)

```bash
hass-cli service call switch.turn_on  --arguments entity_id=switch.porch
hass-cli service call switch.toggle   --arguments entity_id=switch.porch
```

## Fans (`fan.*`)

```bash
hass-cli service call fan.turn_on        --arguments entity_id=fan.bedroom
hass-cli service call fan.set_percentage --data '{"entity_id":"fan.bedroom","percentage":66}'
hass-cli service call fan.oscillate      --data '{"entity_id":"fan.bedroom","oscillating":true}'
hass-cli service call fan.set_preset_mode --data '{"entity_id":"fan.bedroom","preset_mode":"sleep"}'
```

## Targeting many at once

Use a target instead of a single entity:

```bash
hass-cli service call light.turn_off --data '{"target":{"area_id":"living_room"}}'
hass-cli service call light.turn_on  --data '{"target":{"label_id":"downstairs"},"brightness_pct":50}'
```
