---
name: ha-workflow-automation-builder
version: 0.1.0
description: "Playbook for generating a Home Assistant automation from a natural-language request with hass-cli. Use when the user says 'create an automation that...', 'when X happens do Y', 'automate my lights', or wants a new automation/script/scene built and installed. Orchestrates: gather context -> generate config -> save -> reload -> verify."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli workflow --help"
---

# ha-workflow-automation-builder

A repeatable recipe to turn a request into a working automation. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md); see also
[`../ha-automation/SKILL.md`](../ha-automation/SKILL.md).

## Steps

1. **Gather context** — never guess entity ids or services.
   ```bash
   hass-cli state list -o json | jq '[.[].entity_id]'      # available entities
   hass-cli service describe light.turn_on                 # fields for actions
   hass-cli registry area list                             # for area targeting
   ```
2. **Draft the config** as YAML/JSON. Prefer plural keys
   (`triggers`/`conditions`/`actions`) for UI-managed ids. Minimal example:
   ```yaml
   alias: Sunset lights
   triggers:
     - trigger: sun
       event: sunset
   conditions: []
   actions:
     - action: light.turn_on
       target:
         entity_id: light.living_room
   mode: single
   ```
3. **Save** (choose a stable id; 32-hex for UI storage):
   ```bash
   hass-cli workflow automation save sunset_lights --file sunset_lights.yaml
   ```
4. **Reload** so it takes effect:
   ```bash
   hass-cli workflow automation reload
   ```
5. **Verify** — confirm it exists, then test:
   ```bash
   hass-cli workflow automation get sunset_lights
   hass-cli workflow automation trigger automation.sunset_lights
   hass-cli system logbook --entity automation.sunset_lights
   ```

## Guardrails

- Validate every `entity_id` against `state list` before saving.
- Validate every action against `service describe`.
- If `save` returns HTTP 404, the `config` integration is disabled — fall back
  to editing `automations.yaml` and `automation reload`.
- Keep automations small; prefer multiple focused automations over one large one.
