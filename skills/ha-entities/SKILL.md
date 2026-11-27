---
name: ha-entities
version: "0.0.0"
description: "Sensors and the device types without their own skill: door and motion sensors, numbers, selects, buttons, robot vacuums, alarm panels and sirens, the weather forecast, updates, shopping and to-do lists, calendars. 扫地机、传感器、天气、购物清单、日历."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-entities

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

The long tail of entity domains: things that read (`sensor`, `binary_sensor`, `weather`), small settables (`number`, `select`, `text`, `button`), and a few appliances (`vacuum`, `alarm_control_panel`, `siren`, `update`, `todo`, `calendar`). `input_*` helpers are [`ha-helpers`](../ha-helpers/SKILL.md); history of a sensor is [`ha-states`](../ha-states/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| A reading: "how warm is it outside", "is the window open" | this file | `state get`; the state and its unit are the answer |
| The weather forecast | this file | `service call weather.get_forecasts` |
| Start, stop or dock the vacuum | this file | `vacuum.start`, `vacuum.pause`, `vacuum.return_to_base` |
| Arm or disarm the alarm | this file | `alarm_control_panel.alarm_arm_*` or `alarm_disarm` |
| Set a number or pick an option on a device | this file | `number.set_value`, `select.select_option` |
| Press a button entity | this file | `button.press` |
| Add, tick off or read a shopping or to-do list | [lists and calendars](references/ha-entities-lists.md) | `todo.add_item`, `todo.update_item`, `todo.get_items` |
| What is on the calendar, add an event | [lists and calendars](references/ha-entities-lists.md) | `calendar.get_events`, `calendar.create_event` |

## Reading

```bash
hass-cli state list --domain sensor --search outdoor
hass-cli state get sensor.outdoor_temperature -o json
```

- A `sensor`'s state is a string with `attributes.unit_of_measurement`; say both ("12.5 °C"). `unavailable` or `unknown` means the device is not reporting, not a value.
- A `binary_sensor` is `on` / `off`, and `attributes.device_class` says what that means: `door`, `window`, `opening` on is open; `motion`, `occupancy` on is detected; `moisture` on is wet; `battery` on is low. Answer in those words, not "on".
- An `event.*` entity's state is the time of its last event; `attributes.event_type` is what happened (a doorbell press, a remote button).

## Services that answer

`weather.get_forecasts`, `todo.get_items` and `calendar.get_events` return data instead of changing state; `service call` prints that data.

```bash
hass-cli service call weather.get_forecasts --entity weather.home --data '{"type":"daily"}'
```

The forecast is under `<entity_id>.forecast`; `type` is `daily`, `hourly` or `twice_daily`, whichever the integration supports. Today's conditions are the `weather.*` entity's own state and attributes, no service needed.

## Setting

```bash
hass-cli service call number.set_value --data '{"entity_id":"number.dishwasher_delay","value":2}'
hass-cli service call select.select_option --data '{"entity_id":"select.washer_program","option":"Eco"}'
hass-cli service call button.press --entity button.doorbell_chime
```

- `number` stays inside `attributes.min` / `max`, in `step`; `select` takes one of `attributes.options` as written. `text.set_value` takes `value` up to `attributes.max`.
- A `button` has no on or off; its state is when it was last pressed.

## Vacuums

State is `cleaning`, `docked`, `returning`, `idle`, `paused` or `error`. `vacuum.start`, `pause`, `stop`, `return_to_base`, `locate` (it beeps), `clean_spot`; `set_fan_speed` takes one of `attributes.fan_speed_list`. Cleaning one room is integration-specific: `hass-cli service list vacuum` and the integration's own domain (for example a `roborock.*` or `xiaomi_miio.*` service) show what this one has; `service describe` gives the fields.

## Alarm panels

State is `disarmed`, `armed_home`, `armed_away`, `armed_night`, `armed_vacation`, `arming`, `pending` or `triggered`. Services: `alarm_arm_home`, `alarm_arm_away`, `alarm_arm_night`, `alarm_arm_vacation`, `alarm_disarm`, `alarm_trigger`. When `attributes.code_format` is set, `code` is required: ask the user for it each time. `code_arm_required: false` means arming takes no code.

## Verb index

| Service | For | Read when triggered |
|---|---|---|
| `state get`, `state list --domain` | every reading | [reading](#reading) for what `on` means |
| `weather.get_forecasts`, `todo.get_items`, `calendar.get_events` | data back | [services that answer](#services-that-answer) |
| `number.set_value`, `select.select_option`, `text.set_value`, `button.press` | small settables | [setting](#setting) |
| `vacuum.*` | the robot | [vacuums](#vacuums) |
| `alarm_control_panel.*`, `siren.turn_on`, `siren.turn_off` | security | [alarm panels](#alarm-panels) and the rules below |
| `update.install` | install an available update (state `on`) | only on an explicit request; `backup: true` where supported |
| `todo.*`, `calendar.*` | lists and events | [lists and calendars](references/ha-entities-lists.md) |
| Any other domain | `lawn_mower`, `image`, `camera`, `date`, `time`, ... | `hass-cli service list <domain>` shows its services; `service describe` gives the fields |

## Safety and escalation

- Disarming an alarm, turning a siren on or `alarm_trigger` happen only on the user's explicit request for it. Never as a step of another request, never to "test" it. An alarm code is the user's secret: do not repeat or store it.
- `update.install` on Home Assistant Core, the OS or the Supervisor restarts the system: only when the user asked for that update, and say it will be unreachable for a few minutes. Updates in general: [`ha-system`](../ha-system/SKILL.md).
- `button.press` can do anything the device defines (restart a router, open a gate): read the name and press only the one the user meant.
- `state set` overwrites what Home Assistant shows but does not drive a device. Never use it to "set" a sensor, number or vacuum; that is a service call.
- Removing to-do items or calendar events deletes the user's data: only the ones they named.
