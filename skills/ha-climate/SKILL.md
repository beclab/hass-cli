---
name: ha-climate
version: 0.1.0
description: "Control Home Assistant climate, HVAC, thermostats, humidifiers, and water heaters with hass-cli: set target temperature, HVAC mode (heat/cool/auto/off), fan mode, preset, humidity, and water-heater operation. Use for 'set the thermostat to 21', 'turn on the AC', 'set to eco', 'set humidity to 45%' requests. All control goes through service call."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-climate

Thermostats, HVAC, humidifiers, water heaters. **Prerequisites:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md), [`../ha-services/SKILL.md`](../ha-services/SKILL.md).

## Climate (`climate.*`)

Inspect capabilities first — modes and ranges vary by device:

```bash
hass-cli state get climate.living -o json
# attributes: hvac_modes, fan_modes, preset_modes, min_temp, max_temp,
#             target_temp_step, current_temperature
```

```bash
# Target temperature
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","temperature":21.5}'

# Range mode (heat_cool) uses target_temp_low/high
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","target_temp_low":19,"target_temp_high":24}'

# HVAC mode: off|heat|cool|heat_cool|auto|dry|fan_only
hass-cli service call climate.set_hvac_mode --data '{"entity_id":"climate.living","hvac_mode":"heat"}'

# Fan mode and preset (values must come from the *_modes attributes)
hass-cli service call climate.set_fan_mode  --data '{"entity_id":"climate.living","fan_mode":"auto"}'
hass-cli service call climate.set_preset_mode --data '{"entity_id":"climate.living","preset_mode":"eco"}'

hass-cli service call climate.turn_off --arguments entity_id=climate.living
```

## Humidifier (`humidifier.*`)

```bash
hass-cli service call humidifier.turn_on        --arguments entity_id=humidifier.bedroom
hass-cli service call humidifier.set_humidity   --data '{"entity_id":"humidifier.bedroom","humidity":45}'
hass-cli service call humidifier.set_mode       --data '{"entity_id":"humidifier.bedroom","mode":"auto"}'
```

## Water heater (`water_heater.*`)

```bash
hass-cli service call water_heater.set_temperature   --data '{"entity_id":"water_heater.tank","temperature":50}'
hass-cli service call water_heater.set_operation_mode --data '{"entity_id":"water_heater.tank","operation_mode":"eco"}'
```

Always pick mode/preset/fan values from the entity's advertised `*_modes`
attributes; HA rejects unknown values.
