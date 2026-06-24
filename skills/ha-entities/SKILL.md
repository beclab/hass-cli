---
name: ha-entities
version: 0.1.0
description: "Read and control the general Home Assistant entity domains with hass-cli that don't have a dedicated skill: sensor, binary_sensor, number, select, text, button, update, weather, vacuum, alarm_control_panel, siren, camera, todo lists, and calendars. Use for 'read this sensor', 'set this number/select/text helper-style entity', 'start the vacuum', 'arm the alarm', 'what's on my to-do list', 'next calendar event' requests."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-entities

The long tail of entity domains. **Prerequisites:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md), [`../ha-services/SKILL.md`](../ha-services/SKILL.md).

> Read with `state get`; control with `service call`. Find candidates with
> `hass-cli state list -o json | rg '"entity_id": "<domain>\.'`.

## Read-only domains

```bash
hass-cli state get sensor.outdoor_temp -o json        # state + unit_of_measurement
hass-cli state get binary_sensor.front_door -o json   # on/off (device_class for meaning)
hass-cli state get weather.home -o json               # attributes: temperature, forecast
hass-cli state get update.core -o json                # on = update available
```

Weather forecast (it moved out of attributes) via service:

```bash
hass-cli service call weather.get_forecasts --data '{"entity_id":"weather.home","type":"daily"}'
```

## Settable entity domains

```bash
hass-cli service call number.set_value     --data '{"entity_id":"number.brightness_limit","value":80}'
hass-cli service call select.select_option --data '{"entity_id":"select.mode","option":"away"}'
hass-cli service call text.set_value       --data '{"entity_id":"text.label","value":"hello"}'
hass-cli service call button.press         --arguments entity_id=button.restart
hass-cli service call update.install       --arguments entity_id=update.core
```

## Vacuum / alarm / siren / camera

```bash
hass-cli service call vacuum.start         --arguments entity_id=vacuum.roomba
hass-cli service call vacuum.return_to_base --arguments entity_id=vacuum.roomba

hass-cli service call alarm_control_panel.alarm_arm_away --data '{"entity_id":"alarm_control_panel.home","code":"1234"}'
hass-cli service call alarm_control_panel.alarm_disarm   --data '{"entity_id":"alarm_control_panel.home","code":"1234"}'

hass-cli service call siren.turn_on --arguments entity_id=siren.alarm
hass-cli raw api GET camera_proxy/camera.front   # snapshot bytes (binary; redirect to a file)
```

## To-do lists (`todo.*`)

```bash
hass-cli raw ws todo/item/list --data '{"entity_id":"todo.shopping"}' -o json
hass-cli service call todo.add_item    --data '{"entity_id":"todo.shopping","item":"Milk"}'
hass-cli service call todo.update_item --data '{"entity_id":"todo.shopping","item":"Milk","status":"completed"}'
hass-cli service call todo.remove_item --data '{"entity_id":"todo.shopping","item":"Milk"}'
```

## Calendars (`calendar.*`)

```bash
hass-cli state list -o json | rg '"entity_id": "calendar\.'
# Events in a window (REST):
hass-cli raw api GET 'calendars/calendar.personal?start=2026-06-01T00:00:00Z&end=2026-06-30T00:00:00Z' -o json
# Create an event:
hass-cli service call calendar.create_event --data '{"entity_id":"calendar.personal","summary":"Dentist","start_date_time":"2026-06-25T09:00:00","end_date_time":"2026-06-25T10:00:00"}'
```

## Tags (`tag.*`)

```bash
hass-cli raw ws tag/list -o json     # NFC/QR tags; scans fire tag_scanned events
```

When unsure which service a domain offers, `hass-cli service list -o json` then
`hass-cli service describe <domain.service>`.
