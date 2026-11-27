---
name: ha-climate
version: "0.0.0"
description: "Heating, cooling and air conditioning: warmer, cooler, set the temperature, heat or cool mode, AC fan speed, eco or away presets, humidifiers and dehumidifiers, water heaters and boilers. 空调、暖气、温度、加湿器、热水器."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-climate

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Thermostats and air conditioners (`climate.*`), humidifiers and dehumidifiers (`humidifier.*`), water heaters (`water_heater.*`). A standalone fan is [`ha-lighting`](../ha-lighting/SKILL.md); a temperature sensor that only reads is [`ha-entities`](../ha-entities/SKILL.md). "Heat to 21 every morning" is an automation: [the builder](../ha-workflow-automation-builder/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| A temperature: "set it to 24", "26度" | this file | `climate.set_temperature` with `temperature` |
| Warmer or cooler by some degrees | this file | read `attributes.temperature`, then `set_temperature` with the new value |
| Heat, cool, dry, fan only, or off | this file | `climate.set_hvac_mode` |
| The AC's fan speed, swing, eco or sleep preset | this file | `set_fan_mode`, `set_swing_mode`, `set_preset_mode` |
| Humidity on a humidifier | this file | `humidifier.set_humidity` |
| Hot water temperature or mode | this file | `water_heater.set_temperature`, `set_operation_mode` |
| How warm is it in the room | [`ha-states`](../ha-states/SKILL.md) | `state get` on the sensor or `attributes.current_temperature` |

## The climate model

A `climate.*` entity's **state is its HVAC mode** (`off`, `heat`, `cool`, `heat_cool`, `auto`, `dry`, `fan_only`), not what it is doing; `attributes.hvac_action` (`heating`, `cooling`, `idle`, `off`) says that. Everything it accepts is in its attributes:

```bash
hass-cli state get climate.living -o json
```

| Attribute | Means |
|---|---|
| `hvac_modes`, `fan_modes`, `swing_modes`, `preset_modes` | the only values its services take, spelled as listed |
| `temperature` | the current single target; absent in a range mode |
| `target_temp_low`, `target_temp_high` | the targets in `heat_cool` (a range) |
| `min_temp`, `max_temp`, `target_temp_step` | the allowed range and step |
| `current_temperature`, `current_humidity` | what the room measures |

- Temperatures are in the unit Home Assistant is configured for. A user who says "75" in a Celsius house means Fahrenheit: say the conversion, or ask. `hass-cli config get` shows the unit system.
- Round to `target_temp_step` and stay inside `min_temp` / `max_temp`; outside them the call fails.
- "Warmer" with no number: one or two degrees (one step on a 0.5 step device is too small to notice). Base it on `temperature`, not `current_temperature`.
- Words map to modes: "AC on" or "制冷" is `cool`, "暖气" or "制热" is `heat`, "除湿" is `dry`, "送风" is `fan_only`. "Turn it on" with no mode is `climate.turn_on`, which resumes the last mode.

## Setting it

```bash
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","temperature":24}'
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","temperature":24,"hvac_mode":"cool"}'
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","target_temp_low":19,"target_temp_high":24}'
hass-cli service call climate.set_hvac_mode --data '{"entity_id":"climate.living","hvac_mode":"heat"}'
hass-cli service call climate.set_preset_mode --data '{"entity_id":"climate.living","preset_mode":"eco"}'
```

- `set_temperature` can carry `hvac_mode` in the same call: use it when a device is `off` and the user said "cool to 24", so it does not stay off.
- In `heat_cool` send the low and high pair, not `temperature`; in a single-target mode the reverse.
- A preset can override the target (`away`, `eco`). When a temperature does not take, look at `preset_mode` and offer to clear it with the `none` preset if that is in `preset_modes`.

## Humidifiers

State is `on` or `off`. `humidifier.set_humidity` takes `humidity` within `min_humidity` / `max_humidity`; `humidifier.set_mode` takes one of `available_modes`. A dehumidifier is the same domain; `attributes.device_class` says which.

## Water heaters

`water_heater.set_temperature` (`temperature`) and `water_heater.set_operation_mode` (`operation_mode` from `operation_list`). State is the operation mode. `water_heater.set_away_mode` takes `away_mode: true|false` on devices that support it.

## Verb index

| Service | For | Read when triggered |
|---|---|---|
| `climate.set_temperature` | a target or a low/high range, optionally with the mode | [setting it](#setting-it) |
| `climate.set_hvac_mode`, `turn_on`, `turn_off` | the mode; on resumes, off is `hvac_mode: off` | [the climate model](#the-climate-model) for the words |
| `climate.set_fan_mode`, `set_swing_mode`, `set_preset_mode`, `set_humidity` | one attribute each, value from its `*_modes` list | the attribute table above |
| `humidifier.*` | `turn_on`, `turn_off`, `toggle`, `set_humidity`, `set_mode` | [humidifiers](#humidifiers) |
| `water_heater.*` | `set_temperature`, `set_operation_mode`, `set_away_mode`, `turn_on`, `turn_off` | [water heaters](#water-heaters) |
| Anything else on these domains | integration-specific services | `hass-cli service list climate` shows what this instance has; `service describe` gives the fields |

## Safety and escalation

- A request for one room's AC is one entity. "Turn off the heating" with several thermostats and no room named: say which ones there are and ask, or turn off all only when the user said "all" or "everything".
- Do not change the mode as a side effect: setting a temperature on a device in `heat` stays in `heat` unless the user named another mode.
- Water heaters and boilers above about 60 °C scald, and below about 50 °C for long periods risk bacteria: set what was asked, but say so when a request is outside that band.
- A thermostat that keeps reverting is driven by a schedule or automation; the call prints a `Note:` when it is a running script or automation, and [the front door](../ha-shared/SKILL.md#did-it-work) says what to do. An integration's own schedule (on the device or its app) is outside Home Assistant: tell the user.
