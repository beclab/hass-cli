---
name: ha-workflow-automation-builder
version: "0.0.0"
description: "Playbook: turn a sentence like 'when I get home turn on the hall light' or 'every night at 11 lock the door' into a saved, working Home Assistant automation."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-workflow-automation-builder

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

A recipe for a new automation: something that should happen by itself when a trigger fires. Running, stopping or looping something now is [`ha-automation`](../ha-automation/SKILL.md), which also holds the ids and the verbs this playbook uses.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "When X happens, do Y" or "every day at T, do Y" | this file | `workflow automation save <id> --data @file.yaml` |
| The same, but Y repeats or loops | this file, then [`ha-automation`](../ha-automation/SKILL.md) | a script for Y, called from the automation |

## Steps

1. **Name every entity from the house, not from the sentence.** For each device in the trigger and the action, find it as [the front door](../ha-shared/SKILL.md#control-a-device) does. Ask when a name matches several.

   ```bash
   hass-cli state list --search 玄关
   hass-cli state list --domain person
   ```

2. **Check an unfamiliar action's fields**: `hass-cli service describe <domain.service>`. The usual `turn_on` / `turn_off` need no check.

3. **Write the config** with plural keys, the form the UI writes:

   ```yaml
   # hall_light_home.yaml
   alias: 回家开玄关灯
   description: Turn on the hall light when Alice arrives home after sunset
   mode: single
   triggers:
     - trigger: state
       entity_id: person.alice
       to: home
   conditions:
     - condition: sun
       after: sunset
   actions:
     - action: light.turn_on
       target:
         entity_id: light.hall
   ```

   | Request | Trigger |
   |---|---|
   | arrives, leaves | `trigger: state` on `person.*`, `to: home` / `from: home` |
   | at a time | `trigger: time`, `at: "23:00:00"` |
   | at sunrise or sunset | `trigger: sun`, `event: sunset`, optional `offset: "-00:30:00"` |
   | a sensor crosses a value | `trigger: numeric_state`, `above:` / `below:` |
   | a door opens, motion starts | `trigger: state`, `to: "on"` on the `binary_sensor.*` |
   | every N minutes | `trigger: time_pattern`, `minutes: "/15"` |

4. **Save it** with a new id. A readable one (`hall_light_home`) is fine; the UI uses 32 hex characters for its own.

   ```bash
   hass-cli workflow automation save hall_light_home --data @hall_light_home.yaml
   hass-cli state list --domain automation --search 回家
   ```

   `save` reloads automations. The entity id comes from the alias, so read it from the list rather than guessing it.

5. **Test the actions**: `hass-cli workflow automation trigger automation.<from step 4>`. This skips the trigger and the conditions; tell the user the trigger itself is untested until it fires, and how to check: [`ha-automation`](../ha-automation/SKILL.md) covers debugging a run.

## Safety and escalation

- Read the automation back to the user in one sentence (trigger, condition, action) before saving one that unlocks, opens, disarms or turns off heating.
- One request, one automation. Do not fold in "while I'm at it" actions the user did not ask for.
- Step 5 really changes the devices. When that would be unwelcome (a 3 a.m. alarm, an open garage door), skip it and say so.
- `HTTP 404` on save: the config API is off on this instance (see [connection](../ha-shared/references/ha-shared-connection.md)). Give the user the YAML for `automations.yaml` instead of writing files on their server.
