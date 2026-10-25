---
name: ha-climate
version: 0.2.0
description: "Set temperature, HVAC, fan and preset modes on thermostats, humidifiers and water heaters."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-climate

Thermostats, HVAC, humidifiers, water heaters. Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Quick control (most requests)

Two commands turn an existing device on or off or set it. Do not load other
skills, run `service describe`, or check the state afterwards.

```bash
# 1. Find it by name: a few rows, not the whole house
hass-cli state list --domain climate --search living

# 2. Call the service on the entity_id from step 1
hass-cli service call climate.set_temperature --data '{"entity_id":"climate.living","temperature":21}'
hass-cli service call climate.set_hvac_mode --data '{"entity_id":"climate.living","hvac_mode":"cool"}'
```

The call prints the entities it changed: that is the confirmation. `No entity
changed state` means it was already that way. No row: search one word, another
language, or the domain alone. Several equally likely rows: ask which one.

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
