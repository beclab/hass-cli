---
name: ha-backup
version: "0.0.0"
description: "Home Assistant backups: make a backup now or before a change, list backups and when the last one ran, where they are stored, restore one, delete old ones."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-backup

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Home Assistant's own backups, the ones in Settings > System > Backups. A backup is stored by one or more **agents**: the local disk, Home Assistant Cloud, a network share. A Zigbee coordinator backup is not one of these; it is in [`ha-gateway`](../ha-gateway/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Back up now", a backup before an upgrade or a big change | this file | `backup create`, then `backup list` until it appears |
| When the last backup ran, whether automatic backups work | this file | `backup list` |
| Restore a backup | this file | `backup restore <backup_id>`, on an explicit request |
| Delete old backups | this file | `backup delete <backup_id>` for the ones the user picked |

## Listing

```bash
hass-cli backup list -o json        # backups[], state, last_completed_automatic_backup, next_automatic_backup
hass-cli backup agents -o json      # where backups can go: agent_id and name
hass-cli backup get <backup_id> -o json
```

- Each backup has a `backup_id`, a `name`, a `date`, and `agents`: the agents holding a copy, each with `size` and `protected` (encrypted).
- `with_automatic_settings: true` marks the ones the schedule made. A `last_completed_automatic_backup` days older than the schedule, or `agent_errors` in the list, means automatic backups are failing: say so even when the user only asked for the list.
- The local agent is `backup.local` on a Container install and `hassio.local` on HA OS and Supervised; take it from `backup agents` rather than assuming.

## Creating

```bash
hass-cli backup create --auto
hass-cli backup create --data '{"agent_ids":["backup.local"],"name":"before upgrade"}'
```

- `--auto` uses what the user configured for automatic backups (agents, what is included, encryption). Prefer it: it is what the user already chose.
- `--data` makes a one-off backup to the named agents. It includes Home Assistant and its database; `include_database: false` leaves the history out, `include_all_addons: true` adds the add-ons on HA OS, and `password` encrypts it.
- A backup runs in the background: `create` answers with a job id at once. Ask `backup list` every few seconds; it is done when `state` is back to `idle` and the new backup is in `backups[]`. A large instance takes minutes.

## Restoring

```bash
hass-cli backup get <backup_id> -o json      # which agents hold it, protected or not
hass-cli backup restore <backup_id> --data '{"agent_id":"backup.local"}' --timeout 120
```

- `agent_id` is required: one of the agents in the backup's `agents`.
- A `protected` backup needs `"password"`: the backup encryption key from the backup settings, which only the user has. Ask; do not try guesses.
- A restore replaces the current configuration with the backup's and restarts Home Assistant: anything changed since the backup is lost, and the connection drops. `restore_database: false` keeps the current history; `restore_addons` and `restore_folders` pick parts on HA OS.

## Verb index

| Verb | Takes | Read when triggered |
|---|---|---|
| `backup list`, `backup agents` | nothing | the listing section above |
| `backup get` | backup id | before a restore: agents and `protected` |
| `backup create --auto` / `--data` | the configured settings, or agent ids | the creating section above |
| `backup restore --data` | backup id; `agent_id`, maybe `password` | only on an explicit request |
| `backup delete` | backup id | only for backups the user picked |

## Safety and escalation

- Restore only when the user asked for that backup. Name it with its date, say that changes since then are lost and Home Assistant restarts, and wait for a yes. Offer a fresh backup of the current state first.
- Delete only the backups the user named or picked from a list you showed. "Clean up old backups" is a question: list them with dates and sizes and let the user choose.
- Before a risky change the user asked for (an update, deleting an integration), offer a backup; do not take one silently when it would delay a voice reply by minutes.
- Never print a backup's encryption key or password back to the user, and do not store it.
