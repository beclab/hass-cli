---
name: ha-statistics
version: "0.0.0"
description: "Totals, averages and trends over days, weeks or months from Home Assistant's long-term statistics: how much water or gas last month, average temperature this week, why a sensor has no graph. 用了多少、平均温度、上个月."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-statistics

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

The recorder rolls sensor readings up into statistics: one row per hour, kept indefinitely, plus 5-minute rows kept as long as history (10 days by default). They answer questions over long spans. What a sensor read at a moment today is history, not statistics: [`ha-states`](../ha-states/SKILL.md). Energy dashboard questions ("what used the most power") start at [`ha-energy`](../ha-energy/SKILL.md), which hands its ids to this skill.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| A total over a period: "how much water last month", "这周用了多少气" | this file | `statistics period --period month` (or `day`), add up `change` |
| An average, low or high: "average bedroom temperature this week" | this file | `statistics period --period day`, read `mean` / `min` / `max` |
| Whether a sensor is recorded, why it has no graph | this file | `statistics list`, then `statistics metadata --ids` |
| Electricity, solar, the Energy dashboard's numbers | [`ha-energy`](../ha-energy/SKILL.md) | its `stat_*` ids, queried here |
| What a sensor read at 3 pm, or when a door opened | [`ha-states`](../ha-states/SKILL.md) | `system history` |

## Statistic ids and kinds

- A sensor's statistic id is its entity id, `sensor.water_meter`, with source `recorder`. Integrations that import data (utility accounts, some energy providers) write external statistics, whose ids have a colon, `opower:elec_1234`, and no entity behind them. Take ids from `statistics list`, not from `state list`.
- Only sensors with a `state_class` attribute get statistics. A sensor without one has history but no statistics and no long-term graph; that is the integration's choice, and the usual answer to "why is there no graph".
- **Sum** statistics are meters (`total`, `total_increasing`: kWh, m³, litres). **Mean** statistics are measurements (`measurement`: °C, W, %). `statistics list --type sum` or `--type mean` lists one kind.

## Reading values

```bash
hass-cli config get -o json                     # time_zone: day and month buckets follow it
hass-cli statistics metadata --ids sensor.water_meter -o json
hass-cli statistics period --ids sensor.water_meter \
  --start 2026-10-01T00:00:00+08:00 --end 2026-11-01T00:00:00+08:00 --period month -o json
hass-cli statistics period --ids sensor.bedroom_temperature \
  --start 2026-11-20T00:00:00+08:00 --period day -o json
```

The answer maps each id to a list of buckets. `start` and `end` are epoch milliseconds.

| A bucket of | Has | Means |
|---|---|---|
| a sum statistic | `change`, `sum`, `state` | `change` is what was used in that bucket. `sum` is a running total since recording began and `state` is the meter's own reading; neither is "how much" on its own |
| a mean statistic | `mean`, `min`, `max` | per bucket; average the `mean`s for a span of several buckets |

- "Last month" is one `--period month` bucket from the 1st to the 1st, or the `change`s of its `day` buckets added up. Values are in the metadata's `unit_of_measurement`; say the unit back.
- Give `--start` and `--end` with the house's UTC offset (from `time_zone`); a bare UTC time puts day boundaries in the wrong place for a Chinese household.
- An hour with no readings has no bucket. A short list is a gap in recording, not zero use.
- `--period 5minute` only reaches back as far as history is kept; older spans return nothing at that size. Use `hour` or larger.

## Verb index

| Verb | For | Read when triggered |
|---|---|---|
| `statistics list [--type sum]` | every statistic id, its unit and source; `--type mean` for the other kind | finding the id; external ids are only here |
| `statistics metadata --ids a,b` | unit and kind of named ids | before reading values, to know which field answers |
| `statistics period --ids a,b --start T [--end T] [--period P]` | the buckets | [reading values](#reading-values) |
| `statistics info` | whether the recorder is running, and its backlog | values stopped arriving, or `list` is empty on an instance that has run for hours |
| Problems in the data: unit changed, a meter spike, orphaned ids | `hass-cli raw ws recorder/validate_statistics -o json` lists them | fixing them is Home Assistant's Developer tools, Statistics tab; [safety](#safety-and-escalation) |

## Safety and escalation

- Reading statistics changes nothing; answer with the number, the unit and the span ("about 4.2 m³ of water in October").
- Clearing or adjusting statistics (`recorder/clear_statistics`, `recorder/adjust_sum_statistics`) deletes or rewrites history that cannot be rebuilt. Do it only on the user's explicit request, naming the ids; otherwise point them at the Developer tools Statistics tab.
- When two sensors could answer ("water" matches a meter and a flow rate), use the sum statistic for "how much" and the mean one for "how fast"; ask only when two meters fit equally.
- No statistics for a sensor is not fixable from here: say it has no `state_class`, and that the integration or a template sensor would have to provide one.
