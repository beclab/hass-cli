---
name: ha-energy
version: 0.1.0
description: "Inspect and configure Home Assistant's Energy dashboard with hass-cli: read energy preferences (grid/solar/battery/gas/water sources and devices), validate the energy config, and pull consumption statistics. Use for 'show my energy setup', 'how much solar did I produce', 'what's using the most power' questions. Energy data is WebSocket + statistics-based."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli raw ws --help"
---

# ha-energy

The Energy dashboard config and statistics. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

> No typed `energy` command yet — use `raw ws` for the energy WS API and
> `raw ws recorder/statistics_during_period` for the numbers.

## Read the energy setup

```bash
# Configured sources/devices (grid, solar, battery, gas, water, individual devices)
hass-cli raw ws energy/get_prefs -o json

# Validate the configuration (surfaces misconfigured sensors)
hass-cli raw ws energy/validate -o json

# Solar forecast (if a forecast provider is configured)
hass-cli raw ws energy/solar_forecast -o json
```

`energy/get_prefs` returns `energy_sources` (each with `stat_energy_from/to`,
`stat_cost`, ...) and `device_consumption` (per-device `stat_consumption`).
Those `stat_*` ids are statistic ids you feed to the statistics query below.

## Pull consumption / cost numbers

Energy uses long-term statistics, not live state. Query a period:

```bash
hass-cli raw ws recorder/statistics_during_period --data '{
  "start_time":"2026-06-01T00:00:00+00:00",
  "statistic_ids":["sensor.grid_consumption","sensor.solar_production"],
  "period":"day"
}' -o json
```

`period` is `5minute|hour|day|week|month`. Each bucket has `sum`/`state`/`mean`
depending on the sensor. For "what used the most", read the device
`stat_consumption` ids from `energy/get_prefs`, then compare their `sum` deltas
over the period.

## Find energy sensors

```bash
hass-cli state list -o json | rg '"device_class": "energy"'
hass-cli raw ws recorder/list_statistic_ids --data '{"statistic_type":"sum"}' -o json
```

Changing the energy config (`energy/save_prefs`) is possible via `raw ws` but is
rarely needed from the CLI; prefer reading + analysis here.
