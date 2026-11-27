---
name: ha-energy
version: "0.0.0"
description: "Electricity, solar, battery, gas and water on the Energy dashboard: how much power the house used today or this month, what used the most, solar produced, cost; add a device or meter to the Energy dashboard; fix its warnings. 用电量、耗电最多、发电、电费."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-energy

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

The Energy dashboard is a configuration, not a device: it names which statistics are the grid, the solar panels, the battery, the gas and water meters, and the devices worth tracking. The numbers themselves are long-term statistics. A total for a meter that is not on the dashboard, or an average temperature, is [`ha-statistics`](../ha-statistics/SKILL.md). Current power draw ("how many watts is the oven using now") is a sensor's state: [`ha-states`](../ha-states/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| How much electricity today, this week, last month; how much solar | this file | `energy prefs get`, then `statistics period` on its ids, `change` |
| Which device used the most | this file | `device_consumption` ids, one `statistics period`, compared |
| What it cost | this file | the source's `stat_cost`, or `energy info` |
| Add a device or meter to the Energy dashboard | this file | `energy prefs save` with the whole edited list |
| The dashboard shows a warning, numbers look wrong | this file | `energy validate` |

## Where the ids are

```bash
hass-cli energy prefs get -o json
```

- `energy_sources` is a list; each has a `type`: `grid`, `solar`, `battery`, `gas`, `water`. A grid source holds `flow_from` (imported, each with `stat_energy_from` and optionally `stat_cost`) and `flow_to` (exported, `stat_energy_to`). Solar has `stat_energy_from`; a battery has both. Gas and water have `stat_energy_from` and optionally `stat_cost`.
- `device_consumption` lists the tracked devices, each a `stat_consumption` and optionally a `name`. A device with `included_in_stat` is already counted inside another device's meter: leave it out of a house total.
- `device_consumption_water` is the same for water.
- Every `stat_*` value is a statistic id, usually a `sensor.*` entity id.
- A brand-new instance answers `not_found: No prefs` until the dashboard is set up once. That is an empty configuration, not an error to fix.

## Reading the numbers

```bash
hass-cli config get -o json          # time_zone; give --start with its offset
hass-cli statistics period --ids sensor.grid_import,sensor.solar_production \
  --start 2026-11-27T00:00:00+08:00 --period day -o json
hass-cli statistics period --ids sensor.washer_energy,sensor.fridge_energy,sensor.ac_energy \
  --start 2026-10-01T00:00:00+08:00 --end 2026-11-01T00:00:00+08:00 --period month -o json
```

- Each bucket's `change` is the energy used or produced in it, in the statistic's unit (usually kWh). `sum` and `state` are running meter values, never the answer by themselves. Several buckets: add their `change`s.
- House use is grid import minus export plus solar production, plus battery discharge minus charge, when those sources exist. When only a grid import is configured, report it as "from the grid", not as everything the house used.
- "What used the most": rank the `device_consumption` ids by `change` over the same span. Devices without a meter are not in the list; say the ranking covers tracked devices only.
- Cost: query the `stat_cost` id like any other (its unit is the currency). `energy info -o json` lists, under `cost_sensors`, the cost sensors Home Assistant made for sources priced by a fixed or entity price.

## Changing the configuration

```bash
hass-cli energy prefs get -o json > energy.json
# edit: append {"stat_consumption": "sensor.washer_energy"} to device_consumption
hass-cli energy prefs save --data @energy.json
hass-cli energy validate -o json
```

- Each top-level list in the payload (`energy_sources`, `device_consumption`, `device_consumption_water`) replaces the stored list whole; a list left out is kept. So send the full edited list, never only the new item.
- A device or meter must be an energy sensor (kWh, Wh) with a `state_class`. A power sensor (W) cannot be added; an Integral helper (the `integration` config flow) turns watts into kWh: [`ha-integrations`](../ha-integrations/SKILL.md). Find candidates with `hass-cli statistics list --type sum`.
- New entries show values from the next hour; there is no history from before they were added.

## Verb index

| Verb | For | Read when triggered |
|---|---|---|
| `energy prefs get` | the configured sources and devices, and their statistic ids | [where the ids are](#where-the-ids-are) |
| `energy prefs save --data @f` or `--data '{...}'` | replace one or more of the lists | [changing the configuration](#changing-the-configuration); `@file` takes YAML or JSON |
| `energy validate` | warnings per source and per device, in the order of `prefs get` | an empty inner list means that entry is fine; a non-empty one names the entity and the problem (missing, unavailable, wrong unit, no `state_class`) |
| `energy info` | cost sensors and solar forecast integrations | pricing questions |
| `statistics period`, `statistics list` | the numbers | [`ha-statistics`](../ha-statistics/SKILL.md) for buckets beyond energy |

## Safety and escalation

- Reading is free; answer with the number, the unit and the span, and say which sources it covers.
- `prefs save` rewrites the dashboard the household looks at. Change it only when asked, keep every existing entry, and say what was added or removed. Removing a source drops it from the dashboard; its statistics stay.
- Do not add a sensor the user did not name because it "looks like" energy; list the candidates and let them choose.
- Prices and tariffs are the user's: ask for the number rather than guessing one.
