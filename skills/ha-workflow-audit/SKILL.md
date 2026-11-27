---
name: ha-workflow-audit
version: "0.0.0"
description: "Playbook: health-check a Home Assistant house and report what needs attention: offline or unavailable devices, orphaned and duplicate entities, broken or never-run automations, low batteries, open repairs. 检查一下系统、哪些设备离线、没用的自动化."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-workflow-audit

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

A recipe for a report, not a repair. Home Assistant ships no audit, so each finding is computed from what it does ship: its own repair issues, the current states, and the automation configs. The audit reads only; every fix goes to the skill that owns it, after the user picks it. The templates below run on the server with `hass-cli template`, so the answer is a short list rather than every state in the house.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Check my Home Assistant", a full health check | this file | the report of [step 6](#steps) |
| Which devices are offline or unavailable | this file | steps 1 and 2 |
| Broken automations, or ones that never run | this file | steps 4 and 5 |
| Batteries that need changing | this file | step 3 |
| Fix one of the findings | the skill [step 6](#steps) names | that skill's command |

## Steps

1. **What Home Assistant already knows.** Open repair issues, skipping those with `ignored: true`; they come with their own explanation and often a fix in the UI.

   ```bash
   hass-cli system repairs -o json
   ```

2. **Unavailable and orphaned entities.**

   ```bash
   hass-cli template "{{ states | selectattr('state','eq','unavailable') | map(attribute='entity_id') | list }}"
   hass-cli template "{{ states | selectattr('attributes.restored','defined') | map(attribute='entity_id') | list }}"
   ```

   - `unavailable` means the device or its integration is not answering now. Many from one integration at once is that integration failing, not many dead devices: check `hass-cli integration list` for a state other than `loaded` ([`ha-integrations`](../ha-integrations/SKILL.md)).
   - An entity with `restored: true` is still in the registry but no integration provides it any more: the device was removed or renamed. These are the orphans worth cleaning up.
   - `unknown` is not a fault on its own: scenes, buttons and event entities sit at `unknown` until first used, and sensors are `unknown` until they first report. Report `unknown` only for sensors that should have a value.
   - Duplicates: entity ids ending `_2`, `_3` usually come from a device added twice. List them as candidates next to the original; confirm with `hass-cli registry entity list -o json` (same `platform`, different `device_id`) before calling them duplicates.

3. **Low batteries.**

   ```bash
   hass-cli template "{% for s in states.sensor if s.attributes.device_class == 'battery' and s.state | is_number and s.state | float < 20 %}{{ s.entity_id }} {{ s.state }}%
   {% endfor %}"
   ```

4. **Broken automations.**

   ```bash
   hass-cli state list --domain automation
   hass-cli template "{{ states.automation | map(attribute='attributes.id') | list }}"
   hass-cli workflow automation get <config id> -o yaml
   ```

   - An automation in state `unavailable` failed to load; `hass-cli system errorlog` names the error. One that is `off` was turned off by someone; report it, do not count it as broken.
   - For each automation, read its config and check every `entity_id` it names against the house (`state get`, or the unavailable list from step 2). An id that does not exist, or is an orphan, is a broken reference. Ids inside templates are not checked this way; say so.
   - An automation without `attributes.id` lives in YAML only and `get` cannot read it; list it as unchecked.
   - Whether its last run failed is in its trace: [`ha-automation`](../ha-automation/SKILL.md) covers debugging.

5. **Automations and scripts that never run.**

   ```bash
   hass-cli workflow automation list --columns ENTITY=entity_id,STATE=state,LAST=attributes.last_triggered --sort-by attributes.last_triggered
   hass-cli workflow script list --columns ENTITY=entity_id,LAST=attributes.last_triggered --sort-by attributes.last_triggered
   ```

   Never-run ones (empty `LAST`) sort first, then the oldest. "Unused" is a candidate, not a verdict: a smoke alarm or holiday automation is meant to run rarely.

6. **Report**, grouped and short: a count per kind, then the ids with their friendly names, worst first. End each group with the fix and the skill that does it, and let the user choose:

   | Finding | Fix | Skill |
   |---|---|---|
   | Integration not loading, many unavailable | reload it, or re-authenticate | [`ha-integrations`](../ha-integrations/SKILL.md) |
   | Orphaned or duplicate entity | remove the registry entry, or rename the survivor | [`ha-registry`](../ha-registry/SKILL.md) |
   | Automation with a broken reference, or failing | edit its config | [`ha-automation`](../ha-automation/SKILL.md) |
   | Unused automation | disable or delete it, only if the user says so | [`ha-automation`](../ha-automation/SKILL.md) |
   | Statistics problems (unit changed, orphaned statistic) | Developer tools, Statistics tab | [`ha-statistics`](../ha-statistics/SKILL.md) |
   | Repair issue, errors in the log | follow the issue's own text | [`ha-system`](../ha-system/SKILL.md) |

## Safety and escalation

- The audit changes nothing. Do not delete, disable, rename or reload anything while auditing, even an obvious orphan; report it and wait for the user to name what to fix.
- A fix the user picks still follows its own skill's rules: deleting an entity or automation needs that one named, and the user hears what went.
- Say what the report could not see: YAML-only automations, entity ids inside templates, devices that report rarely. "No problems found" covers only what was checked.
- Many unavailable entities right after a restart are normal for a few minutes; when `system repairs` is empty and the instance just started, check again before reporting a mass failure.
