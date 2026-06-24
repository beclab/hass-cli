---
name: ha-backup
version: 0.1.0
description: "Create, list, inspect, delete, and restore Home Assistant backups with hass-cli. Use for 'make a backup', 'list my backups', 'back up before I change something', 'delete old backups', 'restore this backup' requests. Backups go to configured agents (local and/or cloud/network)."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli backup --help"
---

# ha-backup

Native Home Assistant backups (the Settings > System > Backups feature).
**Prerequisite:** [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## List & inspect

```bash
hass-cli backup list           # backups + manager state (idle/create_backup/...)
hass-cli backup agents         # storage locations (e.g. backup.local, cloud.cloud)
hass-cli backup get <backup_id>
```

`backup list` returns `backups[]` (each with `backup_id`, `name`, `date`,
`agents`, `with_automatic_settings`) plus the manager `state` and the
last/next automatic-backup timestamps.

## Create

Specify which agents to store to (get ids from `backup agents`):

```bash
hass-cli backup create --data '{"agent_ids":["backup.local"],"name":"before upgrade"}'

# Or use the configured automatic-backup settings:
hass-cli backup create --auto
```

`create` returns a `backup_job_id` and runs **asynchronously**. Poll
`hass-cli backup list` until `state` is `idle` and the new backup appears in
`backups[]`. Optional create fields: `include_database` (bool),
`include_folders` (array), `include_addons` / `include_all_addons` (supervised
installs), `password` (encrypt).

> On Windows PowerShell prefer `--data @file.json` over inline JSON.

## Delete

```bash
hass-cli backup delete <backup_id>
```

## Restore

```bash
hass-cli backup restore <backup_id> --data '{"agent_id":"backup.local"}'
```

Restore params: `agent_id` (required — where to pull the backup from),
`password` (if encrypted), `restore_database` (bool), `restore_addons` /
`restore_folders` (arrays), `restore_homeassistant` (bool). Restoring triggers a
Home Assistant restart; the CLI call returns as the process begins.

## Backup-before-change playbook

1. `hass-cli backup agents` → pick an `agent_id`.
2. `hass-cli backup create --data '{"agent_ids":["backup.local"],"name":"pre-change"}'`.
3. Poll `hass-cli backup list` until the backup shows up.
4. Make the change (e.g. `integration reload`, `workflow automation save`).
5. If it goes wrong: `hass-cli backup restore <backup_id> --data '{"agent_id":"backup.local"}'`.
