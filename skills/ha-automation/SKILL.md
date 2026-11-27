---
name: ha-automation
version: "0.0.0"
description: "Scripts, automations and scenes: run one now, stop one that is running (a colour loop, a repeating script), make a device loop or repeat every few seconds, edit, list or delete them."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-automation

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

A script is a sequence that runs when asked; an automation runs it when a trigger fires; a scene is a set of states applied at once. Anything that should happen more than once (a loop, "every few seconds", "for the next ten minutes") is a script, not a series of calls from the agent. New automations built from a sentence ("when I get home...") follow [the builder playbook](../ha-workflow-automation-builder/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Run a script, scene or automation now | this file | `workflow <domain> trigger <entity_id>` |
| Stop what is running: "stop the loop", "stop changing colours" | this file | `workflow script stop script.<id>` |
| A device to loop, cycle or repeat until stopped | [scripts](references/ha-automation-scripts.md) | `workflow script save <id>`, then `trigger` |
| Change an existing script or automation | [scripts](references/ha-automation-scripts.md) | `get`, edit, `save` with the same id |
| Why an automation did not fire, or did the wrong thing | [debugging](references/ha-automation-debug.md) | the trace of its last run |

## Ids

Every script and automation has two names, and each verb takes one of them.

| | Script | Automation | Scene |
|---|---|---|---|
| Config id: `get`, `save`, `delete` | the bare id, `color_cycle` | `attributes.id` of the entity | `attributes.id` of the entity |
| Entity id: `trigger`, `stop`, `service call` | `script.<id>`, always | from the alias, `automation.<alias_slug>` | from the name, `scene.<name_slug>` |

- A script id is lowercase letters, digits and underscores, so a Chinese or spaced name needs an English id: `color_cycle` with `alias: 厨房灯循环变色`. `save` refuses anything else before sending, and prints the entity it made.
- An automation's entity id cannot be predicted from its config id. After `save`, find it with `workflow automation list`; to go the other way, `state get automation.x -o json` shows `attributes.id`.

## Running and stopping

```bash
hass-cli workflow script trigger script.color_cycle
hass-cli workflow scene trigger scene.movie_night
hass-cli workflow automation trigger automation.sunset_lights   # runs its actions, skipping its conditions
hass-cli workflow script stop script.color_cycle
hass-cli workflow automation stop automation.sunset_lights
```

- A script is running while its state is `on`; an automation while `attributes.current` is above 0. `workflow script list` shows the state.
- `script stop` ends the run; the script stays defined. `automation stop` ends the run and leaves the automation enabled, so its trigger still works. To disable an automation for good, `service call automation.turn_off --entity automation.x`.
- A scene has no run to stop. To undo one, set the devices back, or apply another scene.
- When a device keeps changing after the user asked it to stop, something is running: [the front door](../ha-shared/SKILL.md#did-it-work) says what the call's `Note:` means.

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `workflow <domain> list` | nothing | entity ids and whether each is running |
| `workflow <domain> get <id>` | config id | before any edit: change what is there, do not rewrite it from memory |
| `workflow <domain> save <id> --data @file.yaml` | config id; YAML or JSON, inline or `@file` | [scripts](references/ha-automation-scripts.md) for the config shape |
| `workflow <domain> delete <id>` | config id | only on an explicit request |
| `workflow <domain> trigger <entity_id>` | entity id | above |
| `workflow script stop`, `workflow automation stop` | entity id | above |
| `workflow <domain> reload` | nothing | after editing `scripts.yaml` or `automations.yaml` by hand; `save` reloads by itself |

## Safety and escalation

- A loop the agent starts is still running after the reply. Say that it runs until stopped, and how to stop it ("say stop the colour loop").
- `get` before `save` on an existing id: `save` replaces the whole config, so a field left out is a field deleted.
- Do not `delete` a script or automation the user did not name, even one that looks unused; [`ha-workflow-audit`](../ha-workflow-audit/SKILL.md) reports, the user decides.
- An automation that unlocks, opens or disarms something needs the user to confirm the trigger and the action in words before it is saved.
