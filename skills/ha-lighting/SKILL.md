---
name: ha-lighting
version: 0.3.0
description: "Turn lights, switches and fans on or off, dim them, set colour or colour temperature, set fan speed. The skill for 'turn on the light' or 'turn off the fan'."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-lighting

Lights, switches, fans. Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Quick control (most requests)

Two commands turn an existing device on or off or set it. Do not load other
skills, run `service describe`, or check the state afterwards.

```bash
# 1. Find it by name: a few rows, not the whole house
hass-cli state list --domain light --search kitchen

# 2. Call the service on the entity_id from step 1
hass-cli service call light.turn_on --arguments entity_id=light.kitchen
hass-cli service call light.turn_off --arguments entity_id=light.kitchen
hass-cli service call switch.turn_off --arguments entity_id=switch.porch
hass-cli service call fan.turn_on --arguments entity_id=fan.bedroom
```

The call prints the entities it changed: that is the confirmation. `No entity
changed state` means it was already that way. No row: search one word, another
language, or the domain alone. Several equally likely rows: ask which one.

> No `hass-cli light` command — drive everything with `service call`.

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
