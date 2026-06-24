---
name: ha-energy
version: 0.1.0
description: "Inspect and configure Home Assistant's Energy dashboard with hass-cli: read energy preferences (grid/solar/battery/gas/water sources and devices), validate the energy config, and pull consumption statistics. Use for 'show my energy setup', 'how much solar did I produce', 'what's using the most power' questions. Energy data is WebSocket + statistics-based."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli energy --help"
---

# ha-energy

The Energy dashboard config and statistics. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Read the energy setup

```bash
hass-cli energy prefs get        # configured sources + device consumption
hass-cli energy validate         # surface misconfigured sensors
hass-cli energy info             # cost sensors / solar forecast domains
```

`energy prefs get` returns `energy_sources` (each with `stat_energy_from/to`,
`stat_cost`, ...) and `device_consumption` (per-device `stat_consumption`).
Those `stat_*` ids are statistic ids you feed to `hass-cli statistics period`.

> A brand-new instance with no configured Energy dashboard returns
> `not_found: No prefs` until you save prefs once.

## Change the energy config

```bash
hass-cli energy prefs save --data @prefs.json
```

The payload mirrors `energy prefs get`: `energy_sources` + `device_consumption`
(+ `device_consumption_water`). Round-trip: `get` → edit → `save`.

## Pull consumption / cost numbers

Energy uses long-term statistics, not live state (see
[`../ha-statistics/SKILL.md`](../ha-statistics/SKILL.md)):

```bash
hass-cli statistics period \
  --ids sensor.grid_consumption,sensor.solar_production \
  --start 2026-06-01T00:00:00+00:00 --period day
```

For "what used the most", read the device `stat_consumption` ids from
`energy prefs get`, then compare their `sum` deltas over the period.

## Find energy sensors

```bash
hass-cli state list -o json | rg '"device_class": "energy"'
hass-cli statistics list --type sum
```
