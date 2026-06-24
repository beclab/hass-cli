---
name: ha-notify
version: 0.1.0
description: "Send notifications and manage persistent notifications in Home Assistant with hass-cli: push messages to phones/devices via notify services, send TTS to speakers, and create/dismiss the dashboard's persistent notifications. Use for 'notify my phone', 'announce on the kitchen speaker', 'show a dashboard alert' requests."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-notify

Notifications and announcements. **Prerequisites:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md), [`../ha-services/SKILL.md`](../ha-services/SKILL.md).

## Discover notify targets

```bash
hass-cli service list -o json | rg notify        # notify.* services (per device/app)
```

Each mobile app / integration registers its own `notify.<target>` service.

## Push notification

```bash
hass-cli service call notify.notify --data '{"message":"Laundry done","title":"Home"}'

# Specific target (e.g. a phone) with extra data
hass-cli service call notify.mobile_app_pixel --data '{"message":"Door left open","title":"Alert","data":{"priority":"high"}}'
```

## Persistent (dashboard) notifications

```bash
hass-cli service call persistent_notification.create  --data '{"title":"Reminder","message":"Check the filter","notification_id":"filter_check"}'
hass-cli service call persistent_notification.dismiss --data '{"notification_id":"filter_check"}'
```

## Text-to-speech to a speaker

```bash
hass-cli service call tts.speak --data '{"entity_id":"tts.google_translate","media_player_entity_id":"media_player.kitchen","message":"Dinner is ready"}'
# Older setups: tts.google_translate_say with {entity_id: media_player.kitchen, message: "..."}
```

Use `service describe notify.notify` (or the specific target) to confirm
supported `data` fields, which vary by integration.
