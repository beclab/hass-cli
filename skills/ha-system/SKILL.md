---
name: ha-system
version: 0.1.0
description: "Inspect Home Assistant system health, repair issues, error logs, logbook, and state history with hass-cli. Use to diagnose problems, check integration health, find open repairs, read recent errors, or gather the raw native data that audits (dead entities, broken automations) are derived from. Complexity scoring/audit are NOT native HA features; compute them here from native data."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli system --help"
---

# ha-system

Native system insight and the data sources for audits. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Native commands

```bash
hass-cli system health        # per-integration health (system_health/info)
hass-cli system repairs       # open issues (Issue Registry / repairs)
hass-cli system errorlog      # raw error log text
hass-cli system logbook --entity <id>
hass-cli system history --entities a,b --start <iso8601>
```

## What is and isn't native

Home Assistant natively provides: **Repairs** (issue registry), **System
Health**, **traces**, **logbook**, **history**. It does NOT provide complexity
scores, "dead entity" reports, or automation audits — those are derived. Compute
them from native data (see [`../ha-workflow-audit/SKILL.md`](../ha-workflow-audit/SKILL.md)).

## Quick checks

```bash
# Anything broken or needing attention?
hass-cli system repairs -o json | jq '.issues'

# Recent errors only
hass-cli system errorlog | tail -n 40
```
