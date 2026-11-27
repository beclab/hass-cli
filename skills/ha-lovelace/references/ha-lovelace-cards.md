# Card types

Read when the user wants a card other than a plain tile: a graph, gauge, thermostat, a list of entities, text. Back to [`ha-lovelace`](../SKILL.md), which holds the get, edit and save round trip every card here goes through.

Each block is one item of a `cards:` list. Fields beyond these exist; the dashboard's card editor shows them. `config save` does not check cards: a wrong type or field shows up as an error card on the dashboard, so say what was added and where, and ask the user to look.

## A device or a few

```yaml
- type: tile                      # one entity, tap to toggle; the default choice
  entity: light.kitchen
- type: entities                  # a titled list
  title: 卧室
  entities:
    - light.bedroom
    - sensor.bedroom_temperature
    - entity: climate.bedroom
      name: 空调
- type: button                    # runs a script or toggles on tap
  entity: script.movie_night
  name: 电影模式
- type: thermostat
  entity: climate.living_room
```

## Graphs and gauges

```yaml
- type: history-graph             # recent states, from history (days, not months)
  hours_to_show: 24
  entities:
    - sensor.bedroom_temperature
- type: statistics-graph          # long spans, from statistics
  entities:
    - sensor.water_meter
  period: day                     # 5minute, hour, day, week, month
  stat_types: [change]            # change for meters; mean, min, max for measurements
  chart_type: bar
  days_to_show: 30
- type: gauge
  entity: sensor.living_room_humidity
  min: 0
  max: 100
```

For a meter (kWh, m³) use `statistics-graph` with `change`; a `history-graph` of a meter is a line that only rises. Whether a sensor has statistics: [`ha-statistics`](../../ha-statistics/SKILL.md).

## Layout and text

```yaml
- type: heading                   # a title row inside a sections view
  heading: 客厅
- type: markdown
  content: |
    **Welcome home.** Today is {{ now().strftime('%A') }}.
- type: area                      # an area's picture and its main entities
  area: living_room
```

`area` takes an area id from [`ha-registry`](../../ha-registry/SKILL.md). Markdown `content` is a template, rendered live.

## Energy

The Energy dashboard's own cards (`energy-date-selection`, `energy-usage-graph`, `energy-distribution`) can be placed on any dashboard and read the Energy configuration; they show nothing until [`ha-energy`](../../ha-energy/SKILL.md) has sources configured.

## Custom cards

`type: custom:<name>` (for example `custom:mushroom-light-card`) works only when its resource is loaded: `lovelace resource list` shows it. If it is not there, the card is not installed; say so rather than adding a resource URL you were not given.
