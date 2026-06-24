---
name: ha-workflow-audit
version: 0.1.0
description: "Playbook for auditing a Home Assistant instance for tech debt with hass-cli: dead/unavailable entities, duplicate entities, broken automations referencing missing entities, and unused automations. Use when the user asks to 'clean up', 'find broken/unused automations', 'find dead entities', or 'audit my setup'. These are derived analyses (not native HA features) computed from native data."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli system --help"
---

# ha-workflow-audit

Derive tech-debt reports from native data. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md); data sources in
[`../ha-system/SKILL.md`](../ha-system/SKILL.md).

> Home Assistant does not ship these audits. Compute them client-side from
> `state list`, the entity registry, and `logbook`/`history`. Always start with
> native `system repairs`, which surfaces problems HA already knows about.

## Recipes (pipe `-o json` to `jq`)

```bash
# 0. Native issues first
hass-cli system repairs -o json

# 1. Dead entities: unavailable/unknown state
hass-cli state list -o json | jq '[.[] | select(.state=="unavailable" or .state=="unknown") | .entity_id]'

# 2. Duplicate entities: _2/_3 suffixes (re-added integrations)
hass-cli state list -o json | jq '[.[].entity_id | select(test("_[0-9]+$"))]'

# 3. Broken automations: reference entity_ids that no longer exist
#    a) collect existing entities
hass-cli state list -o json | jq -r '.[].entity_id' | sort > /tmp/entities.txt
#    b) for each automation, get its config and diff referenced entity_ids
hass-cli workflow automation list -o json | jq -r '.[].entity_id'

# 4. Unused automations: never/seldom triggered
hass-cli workflow automation list -o json | jq '[.[] | {id:.entity_id, last:.attributes.last_triggered}]'
```

## Output

Summarize findings as a short report (counts + the offending ids) and propose
fixes. Mutations (rename/delete) go through `ha-registry` and `ha-automation`;
always confirm before deleting.
