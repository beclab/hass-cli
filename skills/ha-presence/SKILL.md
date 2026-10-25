---
name: ha-presence
version: 0.3.0
description: "See who is home and where: people, zones, device trackers and groups."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli state list --help"
---

# ha-presence

People, zones, trackers, groups — the inputs for presence automations.
Connection or auth trouble: [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Who is home

```bash
# person.* state is "home", "not_home", or a zone name
hass-cli state list --domain person
hass-cli state get person.alice -o json    # attributes: source, latitude, longitude
```

## Zones

```bash
hass-cli state list --domain zone   # zone.home etc.
# Create/manage zone helpers like any helper config via the zone integration UI;
# zone.home is built from the instance's configured location (see `config get`).
```

## Device trackers

```bash
hass-cli state get device_tracker.alice_phone -o json
# Manually set a tracker's location/state (REST state machine write):
hass-cli state set device_tracker.test --state home
hass-cli service call device_tracker.see --data '{"dev_id":"test","location_name":"home"}'
```

## Groups (`group.*`)

```bash
hass-cli state get group.family -o json    # attributes.entity_id lists members
```

## Presence automation tips

- Trigger on `person.*` or `device_tracker.*` state changes (`to: home` /
  `to: not_home`), or on zone enter/leave.
- Combine with an `input_boolean` "guest mode" helper (see ha-helpers) as a
  condition.
- Verify by watching: `hass-cli event watch state_changed` while moving a
  tracker, or inspect history via `hass-cli system history --entities person.alice`.
