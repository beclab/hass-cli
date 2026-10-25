---
name: ha-statistics
version: 0.2.0
description: "Query recorder long-term statistics: list statistic ids, read their metadata and aggregated values."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli statistics --help"
---

# ha-statistics

Recorder long-term statistics — the aggregated, downsampled history HA keeps
for `sensor`s with `state_class` (and external stats). Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Statistics vs. history

- **History** (`hass-cli system history`) = raw state changes, short retention.
- **Statistics** (here) = hourly/daily/.. roll-ups (`sum`/`mean`/`min`/`max`),
  kept long-term. Use these for "totals over a month", energy, trends.

## Discover what exists

```bash
hass-cli statistics info                      # recorder health (backlog, running)
hass-cli statistics list                      # all statistic ids + metadata
hass-cli statistics list --type sum           # only cumulative (energy/water/gas)
hass-cli statistics list --type mean          # only averaged (temperature, power)
hass-cli statistics metadata --ids sensor.a,sensor.b
```

`list` entries carry `statistic_id`, `unit_of_measurement`, `source`,
`has_sum`, `has_mean`.

## Pull values over a period

```bash
hass-cli statistics period \
  --ids sensor.energy_total,sensor.water_total \
  --start 2026-06-01T00:00:00+00:00 \
  --end   2026-06-30T23:59:59+00:00 \
  --period day
```

- `--period` bucket: `5minute|hour|day|week|month` (default `hour`).
- `--start` is required (ISO8601); `--end` defaults to now.
- Result maps each id to an array of buckets. `sum` series are cumulative —
  for "how much in June", subtract the first bucket's `sum` from the last
  (or read `change` when present). `mean` series give the average per bucket.

## Notes

- A fresh instance has empty statistics until the recorder accrues data
  (first roll-up after ~1 hour); `list` may return `[]` initially.
- Combine with [`../ha-energy/SKILL.md`](../ha-energy/SKILL.md): take the
  `stat_*` ids from `energy prefs get`, then query them here.
- Audits (e.g. "dead/never-updating statistics") are derived client-side from
  this native data — not a built-in HA feature.
